package proc

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/zond/hackyhack/proc/messages"
)

type Outputter func(string, ...interface{})

func (o Outputter) Trace(name string) func() {
	o("%s ->", name)
	start := time.Now()
	return func() {
		o("%s <-\t(%v)", name, time.Now().Sub(start))
	}
}

type Emitter func(*messages.Blob) error

func (e Emitter) Error(request *messages.Request, err *messages.Error) error {
	return e(&messages.Blob{
		Type: messages.BlobTypeResponse,
		Response: &messages.Response{
			Header: messages.ResponseHeader{
				RequestId: request.Header.RequestId,
				Error:     err,
			},
		},
	})
}

func HandleRequest(emitter Emitter, slave interface{}, request *messages.Request) error {
	slaveVal := reflect.ValueOf(slave)

	m := slaveVal.MethodByName(request.Header.Method)
	if !m.IsValid() {
		return emitter.Error(request, &messages.Error{
			Message: fmt.Sprintf("No method %q found.", request.Header.Method),
			Code:    messages.ErrorCodeNoSuchMethod,
		})
	}

	mt := m.Type()
	params := make([]interface{}, mt.NumIn())
	paramVals := make([]reflect.Value, len(params))

	if len(params) > 0 {
		if err := json.Unmarshal([]byte(request.Parameters), &params); err != nil {
			return emitter.Error(request, &messages.Error{
				Message: err.Error(),
				Code:    messages.ErrorCodeJSONDecodeParameters,
			})
		}

		for index := range params {
			rawJSON, err := json.Marshal(params[index])
			if err != nil {
				return emitter.Error(request, &messages.Error{
					Message: err.Error(),
					Code:    messages.ErrorCodeJSONDecodeParameters,
				})
			}

			val := reflect.New(mt.In(index))
			if err := json.Unmarshal(rawJSON, val.Interface()); err != nil {
				emitter.Error(request, &messages.Error{
					Message: err.Error(),
					Code:    messages.ErrorCodeJSONDecodeParameters,
				})
			}
			paramVals[index] = val.Elem()
		}
	}

	resultVals := m.Call(paramVals)
	result := make([]interface{}, len(resultVals))
	for index := range result {
		result[index] = resultVals[index].Interface()
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return emitter.Error(request, &messages.Error{
			Message: err.Error(),
			Code:    messages.ErrorCodeJSONEncodeResult,
		})
	}

	return emitter(&messages.Blob{
		Type: messages.BlobTypeResponse,
		Response: &messages.Response{
			Header: messages.ResponseHeader{
				RequestId: request.Header.RequestId,
			},
			Result: string(resultBytes),
		},
	})
}
