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
				Id:    request.Header.Id,
				Error: err,
			},
		},
	})
}

type RequestHandler func(*messages.Request) (*messages.Response, error)

type ResourceProxy struct {
	SendRequest RequestHandler
}

type ResourceFinder func(askerId, resourceId string) (interface{}, error)

func HandleRequest(emitter Emitter, resourceFinder ResourceFinder, request *messages.Request) error {
	resource, err := resourceFinder(request.Header.Source, request.Resource)
	if err != nil {
		return emitter.Error(request, &messages.Error{
			Message: err.Error(),
			Code:    messages.ErrorCodeNoSuchResource,
		})
	}

	if proxy, ok := resource.(ResourceProxy); ok {
		response, err := proxy.SendRequest(request)
		if err != nil {
			return emitter.Error(request, &messages.Error{
				Message: err.Error(),
				Code:    messages.ErrorCodeProxyFailed,
			})
		}

		return emitter(&messages.Blob{
			Type:     messages.BlobTypeResponse,
			Response: response,
		})
	}

	resourceVal := reflect.ValueOf(resource)

	m := resourceVal.MethodByName(request.Method)
	if !m.IsValid() {
		return emitter.Error(request, &messages.Error{
			Message: fmt.Sprintf("No method %q found.", request.Method),
			Code:    messages.ErrorCodeNoSuchMethod,
		})
	}

	mt := m.Type()
	params := make([]interface{}, mt.NumIn())
	paramVals := make([]reflect.Value, len(params))

	if len(params) > 0 {
		if err := json.Unmarshal([]byte(request.Parameters), &params); err != nil {
			return emitter.Error(request, &messages.Error{
				Message: fmt.Sprintf("json.Unmarshal of parameters failed: %v", err),
				Code:    messages.ErrorCodeJSONDecodeParameters,
			})
		}

		for index := range params {
			rawJSON, err := json.Marshal(params[index])
			if err != nil {
				return emitter.Error(request, &messages.Error{
					Message: fmt.Sprintf("json.Marshal of parameter %v failed: %v", index, err),
					Code:    messages.ErrorCodeJSONDecodeParameters,
				})
			}

			val := reflect.New(mt.In(index))
			if err := json.Unmarshal(rawJSON, val.Interface()); err != nil {
				emitter.Error(request, &messages.Error{
					Message: fmt.Sprintf("json.Unmarshal of parameter %v failed: %v", index, err),
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
				Id: request.Header.Id,
			},
			Result: string(resultBytes),
		},
	})
}
