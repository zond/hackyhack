package mcp

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zond/hackyhack/proc"
	"github.com/zond/hackyhack/proc/messages"
)

var nextRequestId uint64

const (
	clientRestartTimeout = time.Second * 5
)

type MCP struct {
	code              string
	path              string
	childStdin        io.WriteCloser
	childStdinEncoder *json.Encoder
	childStdout       io.ReadCloser
	childStderr       io.ReadCloser
	child             *exec.Cmd
	childLock         sync.RWMutex
	flyingRequests    map[string]*flyingRequest
	flyingConstructs  map[string]*flyingConstruct
	flyingDestructs   map[string]*flyingDestruct
	flyingLock        sync.Mutex
	emitLock          sync.Mutex
	stderrHandler     func([]byte)
	errHandler        func(error)
	debugHandler      proc.Outputter
	resourceFinder    proc.ResourceFinder
	stopped           int32
}

func New(code string, resourceFinder proc.ResourceFinder) (*MCP, error) {
	h := sha1.New()
	if _, err := h.Write([]byte(code)); err != nil {
		return nil, err
	}

	mcp := &MCP{
		code:             code,
		path:             filepath.Join(os.TempDir(), fmt.Sprintf("%s.go", hex.EncodeToString(h.Sum(nil)))),
		flyingRequests:   map[string]*flyingRequest{},
		flyingConstructs: map[string]*flyingConstruct{},
		flyingDestructs:  map[string]*flyingDestruct{},
		stderrHandler: func(b []byte) {
			log.Printf("STDERR: %q", b)
		},
		debugHandler: log.Printf,
		errHandler: func(err error) {
			log.Print(err)
		},
		resourceFinder: resourceFinder,
	}
	return mcp, nil
}

func (m *MCP) StderrHandler(f func([]byte)) *MCP {
	m.stderrHandler = f
	return m
}

type flyingRequest struct {
	waitGroup  sync.WaitGroup
	response   *messages.Response
	resourceId string
}

type flyingConstruct struct {
	waitGroup  sync.WaitGroup
	resourceId string
	construct  *messages.Construct
}

type flyingDestruct struct {
	waitGroup  sync.WaitGroup
	resourceId string
	destruct   *messages.Destruct
}

func (m *MCP) Construct(resourceId string) (bool, error) {
	construct := &messages.Construct{
		ResourceId: resourceId,
		RequestId:  fmt.Sprintf("%X", atomic.AddUint64(&nextRequestId, 1)),
	}

	flying := &flyingConstruct{
		resourceId: resourceId,
	}

	flying.waitGroup.Add(1)
	m.flyingLock.Lock()
	m.flyingConstructs[construct.RequestId] = flying
	m.flyingLock.Unlock()

	if err := m.emit(&messages.Blob{
		Type:      messages.BlobTypeConstruct,
		Construct: construct,
	}); err != nil {
		return false, err
	}

	flying.waitGroup.Wait()

	return flying.construct.Constructed, nil
}

func (m *MCP) Call(resourceId, meth string, params, results interface{}) error {
	request := &messages.Request{
		Header: messages.RequestHeader{
			RequestId:  fmt.Sprintf("%X", atomic.AddUint64(&nextRequestId, 1)),
			ResourceId: resourceId,
			Method:     meth,
		},
	}

	if params != nil {
		paramBytes, err := json.Marshal(params)
		if err != nil {
			return err
		}
		request.Parameters = string(paramBytes)
	}

	flying := &flyingRequest{
		resourceId: resourceId,
	}
	flying.waitGroup.Add(1)
	m.flyingLock.Lock()
	m.flyingRequests[request.Header.RequestId] = flying
	m.flyingLock.Unlock()

	if err := m.emit(&messages.Blob{
		Type:    messages.BlobTypeRequest,
		Request: request,
	}); err != nil {
		return err
	}

	flying.waitGroup.Wait()

	if e := flying.response.Header.Error; e != nil {
		return fmt.Errorf("%v: %v", e.Message, e.Code)
	}

	if results != nil {
		if err := json.Unmarshal([]byte(flying.response.Result), results); err != nil {
			return err
		}
	}

	return nil
}

func (m *MCP) emit(blob *messages.Blob) error {
	m.emitLock.Lock()
	defer m.emitLock.Unlock()
	return m.childStdinEncoder.Encode(blob)
}

func (m *MCP) cleanup() error {
	defer m.debugHandler.Trace("MCP#cleanup")()

	m.childLock.Lock()
	defer m.childLock.Unlock()

	if m.childStdin != nil {
		if err := m.childStdin.Close(); err != nil {
			return err
		}
		m.childStdin = nil
		m.childStdinEncoder = nil
	}
	if m.childStdout != nil {
		if err := m.childStdout.Close(); err != nil {
			return err
		}
		m.childStdout = nil
	}
	if m.childStderr != nil {
		if err := m.childStderr.Close(); err != nil {
			return err
		}
		m.childStderr = nil
	}

	if m.child != nil {
		if m.child.Process != nil && m.child.ProcessState != nil {
			if err := m.child.Process.Kill(); err.Error() == "os: process already finished" {
				err = nil
			} else if err != nil {
				return err
			}
			if _, err := m.child.Process.Wait(); err != nil {
				return err
			}
		}
		m.child = nil
	}

	m.flyingLock.Lock()
	defer m.flyingLock.Unlock()
	m.flyingRequests = map[string]*flyingRequest{}
	m.flyingConstructs = map[string]*flyingConstruct{}
	m.flyingDestructs = map[string]*flyingDestruct{}

	return nil
}

func (m *MCP) startProc() error {
	defer m.debugHandler.Trace("MCP#startProc")()

	m.childLock.Lock()
	defer m.childLock.Unlock()

	_, err := os.Stat(m.path)
	if os.IsNotExist(err) {
		if err := ioutil.WriteFile(m.path, []byte(m.code), 0444); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	m.child = exec.Command("go", "run", m.path)
	if m.childStdin, err = m.child.StdinPipe(); err != nil {
		return err
	}
	m.childStdinEncoder = json.NewEncoder(m.childStdin)
	if m.childStdout, err = m.child.StdoutPipe(); err != nil {
		return err
	}
	if m.childStderr, err = m.child.StderrPipe(); err != nil {
		return err
	}
	decoder := json.NewDecoder(m.childStdout)

	m.debugHandler("MCP#startProc\tgo run %q", m.path)
	if err := m.child.Start(); err != nil {
		return err
	}
	m.debugHandler("MCP#startProc\tstarted pid %v", m.child.Process.Pid)

	go m.restart(m.child.Process)
	go m.loopStdout(decoder)
	go m.loopStderr(m.childStderr)

	return nil
}

func (m *MCP) restart(proc *os.Process) {
	defer m.debugHandler.Trace("MCP#restart")

	_, err := proc.Wait()
	if err != nil {
		m.errHandler(err)
		return
	}
	m.debugHandler("MCP#restart\tchild died")

	time.Sleep(clientRestartTimeout)
	if err := m.cleanup(); err != nil {
		m.errHandler(err)
		return
	}
	m.debugHandler("MCP#restart\tchild cleaned")

	if atomic.LoadInt32(&m.stopped) == 1 {
		return
	}

	if err := m.startProc(); err != nil {
		m.errHandler(err)
		return
	}
	m.debugHandler("MCP#restart\tchild restarted")
}

func (m *MCP) handleRequest(request *messages.Request) {
	if err := proc.HandleRequest(m.emit, m.resourceFinder, request); err != nil {
		if err := m.cleanup(); err != nil {
			log.Fatal(err)
		}
	}
}

func (m *MCP) Stop() error {
	if atomic.CompareAndSwapInt32(&m.stopped, 0, 1) {
		return m.cleanup()
	}
	return nil
}

func (m *MCP) constructDone(c *messages.Construct) {
	m.flyingLock.Lock()
	flying, found := m.flyingConstructs[c.RequestId]
	delete(m.flyingConstructs, c.RequestId)
	m.flyingLock.Unlock()
	if found {
		flying.construct = c
		flying.waitGroup.Done()
	}
}

func (m *MCP) handleResponse(response *messages.Response) {
	m.flyingLock.Lock()
	flying, found := m.flyingRequests[response.Header.RequestId]
	delete(m.flyingRequests, response.Header.RequestId)
	m.flyingLock.Unlock()
	if found {
		flying.response = response
		flying.waitGroup.Done()
	}
}

func (m *MCP) destructDone(d *messages.Destruct) {
	m.flyingLock.Lock()
	flying, found := m.flyingDestructs[d.RequestId]
	delete(m.flyingDestructs, d.RequestId)
	m.flyingLock.Unlock()
	if found {
		flying.destruct = d
		flying.waitGroup.Done()
	}
}

func (m *MCP) loopStdout(dec *json.Decoder) {
	for {
		blob := &messages.Blob{}
		if err := dec.Decode(blob); err != nil {
			m.errHandler(fmt.Errorf("Decoding JSON from child STDIN: %v", err))
			return
		}
		switch blob.Type {
		case messages.BlobTypeRequest:
			go m.handleRequest(blob.Request)
		case messages.BlobTypeConstruct:
			go m.constructDone(blob.Construct)
		case messages.BlobTypeResponse:
			go m.handleResponse(blob.Response)
		case messages.BlobTypeDestruct:
			go m.destructDone(blob.Destruct)
		default:
			m.errHandler(fmt.Errorf("Unknown blob type %v", blob.Type))
			return
		}
	}
}

func (m *MCP) loopStderr(r io.ReadCloser) {
	buf := make([]byte, 1024)
	for {
		r, err := r.Read(buf)
		if err != nil {
			m.errHandler(fmt.Errorf("Reading from child STDERR: %v", err))
			return
		}
		m.stderrHandler(buf[:r])
	}
}

func (m *MCP) Start() error {
	if atomic.LoadInt32(&m.stopped) == 1 {
		return errors.New("Already stopped")
	}
	if err := m.cleanup(); err != nil {
		return err
	}
	if err := m.startProc(); err != nil {
		return err
	}
	return nil
}
