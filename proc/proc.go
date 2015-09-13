package proc

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/zond/hackyhack/proc/messages"
)

var contextType = reflect.TypeOf(&messages.Context{})

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

type ResourceFinder func(askerId, resourceId string) ([]interface{}, error)

func HandleRequest(emitter Emitter, resourceFinder ResourceFinder, request *messages.Request) error {
	resources, err := resourceFinder(request.Header.Source, request.Resource)
	if err != nil {
		return emitter.Error(request, &messages.Error{
			Message: err.Error(),
			Code:    messages.ErrorCodeNoSuchResource,
		})
	}

	for index, resource := range resources {
		if proxy, ok := resource.(ResourceProxy); ok {

			id := request.Header.Id
			response, err := proxy.SendRequest(request)
			if err != nil {
				return emitter.Error(request, &messages.Error{
					Message: err.Error(),
					Code:    messages.ErrorCodeProxyFailed,
				})
			}

			if response.Header.Error == nil || response.Header.Error.Code != messages.ErrorCodeNoSuchMethod || index == len(resources)-1 {
				response.Header.Id = id

				return emitter(&messages.Blob{
					Type:     messages.BlobTypeResponse,
					Response: response,
				})
			}

		} else {

			resourceVal := reflect.ValueOf(resource)

			m := resourceVal.MethodByName(request.Method)
			if m.IsValid() || index == len(resources)-1 {

				if !m.IsValid() {
					return emitter.Error(request, &messages.Error{
						Message: fmt.Sprintf("No method %q found.", request.Method),
						Code:    messages.ErrorCodeNoSuchMethod,
					})
				}

				mt := m.Type()
				paramVals := []reflect.Value{}

				wantedParams := mt.NumIn()
				indexOffset := 0
				if mt.NumIn() > 0 && mt.In(0) == contextType {
					indexOffset = 1
					wantedParams--
				}

				if wantedParams > 0 {
					params := []interface{}{}
					if err := json.Unmarshal([]byte(request.Parameters), &params); err != nil {
						return emitter.Error(request, &messages.Error{
							Message: fmt.Sprintf("json.Unmarshal of parameters failed: %v", err),
							Code:    messages.ErrorCodeJSONDecodeParameters,
						})
					}

					if wantedParams != len(params) {
						return emitter.Error(request, &messages.Error{
							Message: fmt.Sprintf("Wrong number of parameters; got %v, want %v", len(params), mt.NumIn()),
							Code:    messages.ErrorCodeMethodMismatch,
						})
					}

					paramVals = make([]reflect.Value, wantedParams)

					for index := range params {
						rawJSON, err := json.Marshal(params[index])
						if err != nil {
							return emitter.Error(request, &messages.Error{
								Message: fmt.Sprintf("json.Marshal of parameter %v failed: %v", index, err),
								Code:    messages.ErrorCodeJSONDecodeParameters,
							})
						}

						val := reflect.New(mt.In(index + indexOffset))
						if err := json.Unmarshal(rawJSON, val.Interface()); err != nil {
							emitter.Error(request, &messages.Error{
								Message: fmt.Sprintf("json.Unmarshal of parameter %v failed: %v", index, err),
								Code:    messages.ErrorCodeJSONDecodeParameters,
							})
						}
						paramVals[index] = val.Elem()
					}
				}

				if mt.NumIn() > 0 && mt.In(0) == contextType {
					paramVals = append([]reflect.Value{reflect.ValueOf(&messages.Context{
						Request: request,
					})}, paramVals...)
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
		}
	}
	panic("Should never end up here")
}
