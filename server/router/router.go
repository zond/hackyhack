package router

import (
	"fmt"
	"log"
	"sync"

	"github.com/zond/hackyhack/proc"
	"github.com/zond/hackyhack/proc/mcp"
	"github.com/zond/hackyhack/proc/messages"
	"github.com/zond/hackyhack/server/persist"
	"github.com/zond/hackyhack/server/resource"
	"github.com/zond/hackyhack/server/router/validator"
	"github.com/zond/hackyhack/server/void"
)

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
	void      *void.Void
	clients   map[string]Client
}

func New(p *persist.Persister) *Router {
	return &Router{
		persister: p,
		handlers:  map[string]*mcp.MCP{},
		void:      void.New(p),
		clients:   map[string]Client{},
	}
}

func (r *Router) RegisterClient(resource string, client Client) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.clients[resource] = client
}

func (r *Router) UnregisterClient(resource string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.clients, resource)
}

func (r *Router) findResource(source, id string) (interface{}, error) {
	if id == source {
		result := &resourceWrapper{
			resource: id,
			router:   r,
		}
		r.lock.RLock()
		client, found := r.clients[id]
		r.lock.RUnlock()
		if found {
			return &clientWrapper{
				resourceWrapper: *result,
				client:          client,
			}, nil
		}
		return result, nil
	}
	if id == "" {
		return r.void, nil
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

func (r *Router) createMCP(resourceId string) (*mcp.MCP, error) {
	res := &resource.Resource{}
	if err := r.persister.Get(resourceId, res); err != nil {
		return nil, err
	}
	if err := validator.Validate(res.Code); err != nil {
		return nil, err
	}
	m, err := mcp.New(res.Code, r.findResource)
	if err != nil {
		return nil, err
	}
	if err := m.Start(); err != nil {
		return nil, err
	}
	r.lock.Lock()
	defer r.lock.Unlock()

	existingM, found := r.handlers[resourceId]
	if found {
		if err := m.Stop(); err != nil {
			log.Fatal(err)
		}
		return existingM, nil
	}

	r.handlers[resourceId] = m
	return m, nil
}

func (r *Router) MCP(resourceId string) (*mcp.MCP, error) {
	r.lock.RLock()
	m, found := r.handlers[resourceId]
	r.lock.RUnlock()
	if !found {
		return r.createMCP(resourceId)
	}
	return m, nil
}
