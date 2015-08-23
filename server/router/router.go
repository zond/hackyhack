package router

import (
	"sync"

	"github.com/zond/hackyhack/server/persist"
)

type Handler interface {
	Handle(string) error
}

type Sender interface {
	Send(string) error
}
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

func (r *Router) spinHandler(s Sender, keys ...string) (Handler, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	h, err := r.handlers.get(keys...)
	if err == errNotFound {

	} else if err != nil {
		return nil, err
	}
	return h, nil
}

func (r *Router) Handler(s Sender, keys ...string) (Handler, error) {
	r.lock.RLock()
	h, err := r.handlers.get(keys...)
	r.lock.RUnlock()
	if err == errNotFound {
		return r.spinHandler(s, keys...)
	} else if err != nil {
		return nil, err
	}
	return h, nil
}
