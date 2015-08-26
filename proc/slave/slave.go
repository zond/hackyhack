package slave

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"sync"

	"github.com/zond/hackyhack/proc/errors"
	"github.com/zond/hackyhack/proc/interfaces"
	"github.com/zond/hackyhack/proc/messages"
)

var (
	driver       *slaveDriver
	registerOnce sync.Once
)

type mcp struct {
	driver     *slaveDriver
	resourceId string
}

func (m *mcp) GetResourceId() string {
	return m.resourceId
}

func (m *mcp) GetContainer() interfaces.Container {
	return nil
}

func (m *mcp) GetContent() interfaces.Nameds {
	return nil
}

type slaveDriver struct {
	encoder   *json.Encoder
	generator SlaveGenerator
	slaves    map[string]interfaces.Named
	slaveLock sync.RWMutex
	emitLock  sync.Mutex
}

type SlaveGenerator func(interfaces.MCP) interfaces.Named

func newDriver(gen SlaveGenerator) *slaveDriver {
	return &slaveDriver{
		encoder:   json.NewEncoder(os.Stdout),
		slaves:    map[string]interfaces.Named{},
		generator: gen,
	}
}

func Register(gen SlaveGenerator) {
	registerOnce.Do(func() {
		driver = newDriver(gen)
	})
	driver.loop()
}

func (s *slaveDriver) handleRequest(request *messages.Request) {
	s.slaveLock.RLock()
	slave, found := s.slaves[request.Header.ResourceId]
	s.slaveLock.RUnlock()
	if !found {
		s.emitError(request, &messages.Error{
			Message: fmt.Sprintf("No resource %q found.", request.Header.ResourceId),
			Code:    messages.ErrorCodeNoSuchResource,
		})
		return
	}
	slaveVal := reflect.ValueOf(slave)

	m := slaveVal.MethodByName(request.Header.Method)
	if !m.IsValid() {
		s.emitError(request, &messages.Error{
			Message: fmt.Sprintf("No method %q found.", request.Header.Method),
			Code:    messages.ErrorCodeNoSuchMethod,
		})
		return
	}

	mt := m.Type()
	params := make([]interface{}, mt.NumIn())
	paramVals := make([]reflect.Value, len(params))

	if len(params) > 0 {
		if err := json.Unmarshal([]byte(request.Parameters), &params); err != nil {
			s.emitError(request, &messages.Error{
				Message: err.Error(),
				Code:    messages.ErrorCodeJSONDecodeParameters,
			})
			return
		}

		for index := range params {
			rawJSON, err := json.Marshal(params[index])
			if err != nil {
				s.emitError(request, &messages.Error{
					Message: err.Error(),
					Code:    messages.ErrorCodeJSONDecodeParameters,
				})
			}

			val := reflect.New(mt.In(index))
			if err := json.Unmarshal(rawJSON, val.Interface()); err != nil {
				s.emitError(request, &messages.Error{
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
		s.emitError(request, &messages.Error{
			Message: err.Error(),
			Code:    messages.ErrorCodeJSONEncodeResult,
		})
		return
	}

	s.emit(&messages.Blob{
		Type: messages.BlobTypeResponse,
		Response: &messages.Response{
			Header: messages.ResponseHeader{
				RequestId: request.Header.RequestId,
			},
			Result: string(resultBytes),
		},
	})
}

func (s *slaveDriver) emitError(request *messages.Request, err *messages.Error) {
	s.emit(&messages.Blob{
		Type: messages.BlobTypeResponse,
		Response: &messages.Response{
			Header: messages.ResponseHeader{
				RequestId: request.Header.RequestId,
				Error:     err,
			},
		},
	})
}

func (s *slaveDriver) emit(blob *messages.Blob) {
	s.emitLock.Lock()
	defer s.emitLock.Unlock()
	if err := s.encoder.Encode(blob); err != nil {
		log.Fatal(err)
	}
}

func (s *slaveDriver) construct(c *messages.Construct) {
	s.slaveLock.Lock()
	_, found := s.slaves[c.ResourceId]
	if !found {
		s.slaves[c.ResourceId] = s.generator(&mcp{
			driver:     s,
			resourceId: c.ResourceId,
		})
	}
	s.slaveLock.Unlock()

	s.emit(&messages.Blob{
		Type:      messages.BlobTypeConstruct,
		Construct: c,
	})
}

func (s *slaveDriver) loop() {
	decoder := json.NewDecoder(os.Stdin)
	for {
		blob := &messages.Blob{}
		if err := decoder.Decode(blob); err != nil {
			log.Fatal(err)
		}
		switch blob.Type {
		case messages.BlobTypeRequest:
			s.handleRequest(blob.Request)
		case messages.BlobTypeConstruct:
			s.construct(blob.Construct)
		case messages.BlobTypeResponse:
		default:
			log.Fatal(errors.ErrUnknownBlobType)
		}
	}
}
