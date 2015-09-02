package router

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/zond/hackyhack/proc"
	"github.com/zond/hackyhack/proc/mcp"
	"github.com/zond/hackyhack/proc/messages"
	"github.com/zond/hackyhack/server/persist"
	"github.com/zond/hackyhack/server/resource"
	"github.com/zond/hackyhack/server/router/validator"
)

var initialVoid string

func init() {
	path := filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "zond", "hackyhack", "server", "router", "default", "void.go")
	b, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("Unable to load default void file %q: %v", path, err)
	}
	initialVoid = string(b)
}

type resourceWrapper struct {
	router   *Router
	resource string
}

func (w *resourceWrapper) GetContainer() (string, *messages.Error) {
	res := &resource.Resource{}
	if err := w.router.persister.Get(w.resource, res); err != nil {
		return "", &messages.Error{
			Message: fmt.Sprintf("persister.Get failed: %v", err),
			Code:    messages.ErrorCodeDatabase,
		}
	}
	return res.Container, nil
}

func (w *resourceWrapper) GetContent() ([]string, *messages.Error) {
	res := &resource.Resource{}
	if err := w.router.persister.Get(w.resource, res); err != nil {
		return nil, &messages.Error{
			Message: fmt.Sprintf("persister.Get failed: %v", err),
			Code:    messages.ErrorCodeDatabase,
		}
	}
	return res.Content, nil
}

type Client interface {
	Send(string) error
}

type clientWrapper struct {
	client Client
	resourceWrapper
}

func (w *clientWrapper) SendToClient(s string) *messages.Error {
	if err := w.client.Send(s); err != nil {
		w.resourceWrapper.router.UnregisterClient(w.resourceWrapper.resource)
		return &messages.Error{
			Message: fmt.Sprintf("client.Send failed: %v", err),
			Code:    messages.ErrorCodeSendToClient,
		}
	}
	return nil
}

type Router struct {
	persister *persist.Persister
	handlers  map[string]*mcp.MCP
	lock      sync.RWMutex
	clients   map[string]*clientWrapper
}

func (r *Router) ensureVoid() error {
	void := &resource.Resource{}
	if err := r.persister.Get(messages.VoidResource, void); err == persist.ErrNotFound {
		void.Id = messages.VoidResource
		void.Code = initialVoid
		if err := r.persister.Put(messages.VoidResource, void); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

func New(p *persist.Persister) (*Router, error) {
	r := &Router{
		persister: p,
		handlers:  map[string]*mcp.MCP{},
		clients:   map[string]*clientWrapper{},
	}

	if err := r.ensureVoid(); err != nil {
		return nil, err
	}

	m, err := r.MCP(messages.VoidResource)
	if err != nil {
		return nil, err
	}
	if _, err = m.Construct(messages.VoidResource); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Router) RegisterClient(resource string, client Client) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.clients[resource] = &clientWrapper{
		resourceWrapper: resourceWrapper{
			resource: resource,
			router:   r,
		},
		client: client,
	}
}

func (r *Router) UnregisterClient(resource string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.clients, resource)
}

func (r *Router) findResource(source, id string) (interface{}, error) {
	if id == source {
		r.lock.RLock()
		client, found := r.clients[id]
		r.lock.RUnlock()
		if found {
			return client, nil
		}
		result := &resourceWrapper{
			resource: id,
			router:   r,
		}
		return result, nil
	}
	res := &resource.Resource{}
	if err := r.persister.Get(id, res); err != nil {
		return nil, err
	}
	m, err := r.MCP(res.Id)
	if err != nil {
		return nil, err
	}
	return proc.ResourceProxy{
		SendRequest: m.SendRequest,
	}, nil
}

func (r *Router) createMCP(res *resource.Resource, key string) (*mcp.MCP, error) {
	m, err := mcp.New(res.Code, r.findResource)
	if err != nil {
		return nil, err
	}
	if err := m.Start(); err != nil {
		return nil, err
	}
	r.lock.Lock()
	defer r.lock.Unlock()

	existingM, found := r.handlers[key]
	if found {
		if err := m.Stop(); err != nil {
			log.Fatal(err)
		}
		return existingM, nil
	}

	r.handlers[key] = m
	return m, nil
}

func (r *Router) MCP(resourceId string) (*mcp.MCP, error) {
	res := &resource.Resource{}
	if err := r.persister.Get(resourceId, res); err != nil {
		return nil, err
	}
	if err := validator.Validate(res.Code); err != nil {
		return nil, err
	}
	codeHash := sha1.New()
	if _, err := io.WriteString(codeHash, res.Code); err != nil {
		return nil, err
	}
	key := fmt.Sprintf("%s.%s", res.Owner, base64.StdEncoding.EncodeToString(codeHash.Sum(nil)))

	r.lock.RLock()
	m, found := r.handlers[key]
	r.lock.RUnlock()
	if !found {
		return r.createMCP(res, key)
	}
	return m, nil
}
