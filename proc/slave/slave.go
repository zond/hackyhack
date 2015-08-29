package slave

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/zond/hackyhack/proc"
	"github.com/zond/hackyhack/proc/errors"
	"github.com/zond/hackyhack/proc/interfaces"
	"github.com/zond/hackyhack/proc/messages"
)

func setrlimit(i int, r *syscall.Rlimit) {
	if err := syscall.Setrlimit(i, r); err != nil {
		panic(err)
	}
}

const (
	RLIMIT_AS     = 1 << 22
	RLIMIT_CORE   = 0
	RLIMIT_CPU    = 1
	RLIMIT_DATA   = 1 << 22
	RLIMIT_FSIZE  = 0
	RLIMIT_NOFILE = 3
	RLIMIT_STACK  = 1 << 23
)

func init() {
	setrlimit(syscall.RLIMIT_AS, &syscall.Rlimit{RLIMIT_AS, RLIMIT_AS})
	setrlimit(syscall.RLIMIT_CORE, &syscall.Rlimit{RLIMIT_CORE, RLIMIT_CORE})
	setrlimit(syscall.RLIMIT_CPU, &syscall.Rlimit{RLIMIT_CPU, RLIMIT_CPU})
	setrlimit(syscall.RLIMIT_DATA, &syscall.Rlimit{RLIMIT_DATA, RLIMIT_DATA})
	setrlimit(syscall.RLIMIT_FSIZE, &syscall.Rlimit{RLIMIT_FSIZE, RLIMIT_FSIZE})
	setrlimit(syscall.RLIMIT_NOFILE, &syscall.Rlimit{RLIMIT_NOFILE, RLIMIT_NOFILE})
	setrlimit(syscall.RLIMIT_STACK, &syscall.Rlimit{RLIMIT_STACK, RLIMIT_STACK})
}

var (
	driver        *slaveDriver
	registerOnce  sync.Once
	nextRequestId uint64
)

type mcp struct {
	driver     *slaveDriver
	resourceId string
}

func (m *mcp) Call(resourceId, method string, params, results interface{}) error {
	return m.driver.emitRequest(m.resourceId, resourceId, method, params, results)
}

func (m *mcp) Fatal(i ...interface{}) {
	log.Fatal(i...)
}

func (m *mcp) Fatalf(f string, i ...interface{}) {
	log.Fatalf(f, i...)
}

func (m *mcp) Logf(f string, i ...interface{}) {
	fmt.Fprintf(os.Stderr, f, i...)
}

func (m *mcp) Log(i ...interface{}) {
	fmt.Fprintln(os.Stderr, i...)
}

func (m *mcp) GetResourceId() string {
	return m.resourceId
}

func (m *mcp) GetContainer() string {
	result := ""
	if err := m.driver.emitRequest(m.resourceId, m.resourceId, messages.MethodGetContainer, nil, &result); err != nil {
		log.Fatal(err)
	}
	return result
}

func (m *mcp) SendToClient(s string) {
	if err := m.driver.emitRequest(m.resourceId, m.resourceId, messages.MethodSendToClient, []string{s}, nil); err != nil {
		log.Fatal(err)
	}
}

func (m *mcp) GetContent() []string {
	result := []string{}
	if err := m.driver.emitRequest(m.resourceId, m.resourceId, messages.MethodGetContent, nil, &result); err != nil {
		log.Fatal(err)
	}
	return result
}

type flyingRequest struct {
	waitGroup  sync.WaitGroup
	response   *messages.Response
	resourceId string
}

type slaveDriver struct {
	encoder            *json.Encoder
	generator          SlaveGenerator
	slaves             map[string]interfaces.Describable
	slaveLock          sync.RWMutex
	emitLock           sync.Mutex
	flyingRequests     map[string]*flyingRequest
	flyingRequestsLock sync.Mutex
}

type SlaveGenerator func(interfaces.MCP) interfaces.Describable

func newDriver(gen SlaveGenerator) *slaveDriver {
	driver := &slaveDriver{
		encoder:        json.NewEncoder(os.Stdout),
		slaves:         map[string]interfaces.Describable{},
		generator:      gen,
		flyingRequests: map[string]*flyingRequest{},
	}
	return driver
}

func Register(gen SlaveGenerator) {
	registerOnce.Do(func() {
		driver = newDriver(gen)
	})
	driver.loop()
}

func (s *slaveDriver) logErr(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, string(debug.Stack()))
	}
}

func (s *slaveDriver) findSlave(resourceId string) (interface{}, error) {
	s.slaveLock.RLock()
	slave, found := s.slaves[resourceId]
	s.slaveLock.RUnlock()
	if !found {
		return nil, fmt.Errorf("No slave %q found", resourceId)
	}
	return slave, nil
}

func (s *slaveDriver) handleRequest(request *messages.Request) {
	s.logErr(proc.HandleRequest(s.emit, s.findSlave, request))
}

func (s *slaveDriver) emitRequest(srcResourceId, dstResourceId, method string, params, result interface{}) error {
	s.slaveLock.RLock()
	_, found := s.slaves[srcResourceId]
	s.slaveLock.RUnlock()
	if !found {
		log.Fatal(fmt.Errorf("Unregistered resource id %q", srcResourceId))
	}

	request := &messages.Request{
		Header: messages.RequestHeader{
			RequestId:  fmt.Sprintf("%X", atomic.AddUint64(&nextRequestId, 1)),
			ResourceId: dstResourceId,
			Method:     method,
		},
	}

	if params != nil {
		paramBytes, err := json.Marshal(params)
		if err != nil {
			log.Fatal(err)
		}
		request.Parameters = string(paramBytes)
	}

	flying := &flyingRequest{
		resourceId: srcResourceId,
	}
	flying.waitGroup.Add(1)
	s.flyingRequestsLock.Lock()
	s.flyingRequests[request.Header.RequestId] = flying
	s.flyingRequestsLock.Unlock()

	s.emit(&messages.Blob{
		Type:    messages.BlobTypeRequest,
		Request: request,
	})

	flying.waitGroup.Wait()

	if result != nil {
		if err := json.Unmarshal([]byte(flying.response.Result), result); err != nil {
			log.Fatal(err)
		}
	}

	if herr := flying.response.Header.Error; herr != nil {
		return herr
	}

	return nil
}

func (s *slaveDriver) handleResponse(response *messages.Response) {
	if response.Header.Error != nil {
		log.Fatal(fmt.Errorf("%v: %v", response.Header.Error.Message, response.Header.Error.Code))
	}

	s.flyingRequestsLock.Lock()
	flying, found := s.flyingRequests[response.Header.RequestId]
	delete(s.flyingRequests, response.Header.RequestId)
	s.flyingRequestsLock.Unlock()
	if found {
		flying.response = response
		flying.waitGroup.Done()
	}
}

func (s *slaveDriver) emit(blob *messages.Blob) error {
	s.emitLock.Lock()
	defer s.emitLock.Unlock()
	if err := s.encoder.Encode(blob); err != nil {
		log.Fatal(err)
	}
	return nil
}

func (s *slaveDriver) destruct(d *messages.Destruct) {
	s.slaveLock.Lock()
	slave, found := s.slaves[d.ResourceId]
	if found {
		if destructible, ok := slave.(interfaces.Destructible); ok {
			go destructible.Destroy()
		}
		d.Destroyed = true
	}
	delete(s.slaves, d.ResourceId)
	s.slaveLock.Unlock()

	s.flyingRequestsLock.Lock()
	for requestId, flying := range s.flyingRequests {
		if flying.resourceId == d.ResourceId {
			delete(s.flyingRequests, requestId)
		}
	}
	s.flyingRequestsLock.Unlock()
}

func (s *slaveDriver) construct(c *messages.Construct) {
	s.slaveLock.Lock()
	_, found := s.slaves[c.ResourceId]
	if !found {
		s.slaves[c.ResourceId] = s.generator(&mcp{
			driver:     s,
			resourceId: c.ResourceId,
		})
		c.Constructed = true
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
			go s.handleRequest(blob.Request)
		case messages.BlobTypeConstruct:
			go s.construct(blob.Construct)
		case messages.BlobTypeResponse:
			go s.handleResponse(blob.Response)
		case messages.BlobTypeDestruct:
			go s.destruct(blob.Destruct)
		default:
			log.Fatal(errors.ErrUnknownBlobType)
		}
	}
}
