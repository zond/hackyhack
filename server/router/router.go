package router

import (
	"sync"

	"github.com/zond/hackyhack/proc"
	"github.com/zond/hackyhack/proc/mcp"
	"github.com/zond/hackyhack/server/persist"
	"github.com/zond/hackyhack/server/router/validator"
)

type Router struct {
	persister persist.Persister
	handlers  cache
	lock      sync.RWMutex
}

func New(p persist.Persister) *Router {
	return &Router{
		persister: p,
		handlers:  cache{},
	}
}

func (r *Router) createMCP(resourceFinder proc.ResourceFinder, keys ...string) (*mcp.MCP, error) {
	code, err := r.persister.Get(keys...)
	if err != nil {
		return nil, err
	}
	if err := validator.Validate(code); err != nil {
		return nil, err
	}
	m, err := mcp.New(code, resourceFinder)
	if err != nil {
		return nil, err
	}
	if err := m.Start(); err != nil {
		return nil, err
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	h, err := r.handlers.get(keys...)
	if err == errNotFound {
		h = m
		r.handlers.set(m, keys...)
	} else if err != nil {
		return nil, err
	}
	return h, nil
}

func (r *Router) MCP(resourceFinder proc.ResourceFinder, keys ...string) (*mcp.MCP, error) {
	r.lock.RLock()
	h, err := r.handlers.get(keys...)
	r.lock.RUnlock()
	if err == errNotFound {
		return r.createMCP(resourceFinder, keys...)
	} else if err != nil {
		return nil, err
	}
	return h, nil
}
