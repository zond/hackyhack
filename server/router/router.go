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
	"time"

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

type ownerCode struct {
	owner    string
	codeHash string
}

func newOwnerCode(owner, code string) (ownerCode, error) {
	codeHash := sha1.New()
	if _, err := io.WriteString(codeHash, code); err != nil {
		return ownerCode{}, err
	}
	return ownerCode{
		owner:    owner,
		codeHash: base64.StdEncoding.EncodeToString(codeHash.Sum(nil)),
	}, nil
}

type handlerData struct {
	oc ownerCode
	m  *mcp.MCP
}

type Router struct {
	persister             *persist.Persister
	handlerByOwnerCode    map[ownerCode]*mcp.MCP
	handlerDataByResource map[string]handlerData
	lock                  sync.RWMutex
	clients               map[string]*clientWrapper
}

func (r *Router) ensureVoid() error {
	void := &resource.Resource{}
	if err := r.persister.Get(messages.VoidResource, void); err == persist.ErrNotFound {
		void.Id = messages.VoidResource
		void.Code = initialVoid
		void.UpdatedAt = time.Now()
		void.CreatedAt = void.UpdatedAt
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
		persister:             p,
		handlerByOwnerCode:    map[ownerCode]*mcp.MCP{},
		handlerDataByResource: map[string]handlerData{},
		clients:               map[string]*clientWrapper{},
	}

	if err := r.ensureVoid(); err != nil {
		return nil, err
	}

	_, err := r.MCP(messages.VoidResource)
	if err != nil {
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

func (r *Router) Restart(resource string) error {
	r.lock.RLock()
	_, found := r.handlerDataByResource[resource]
	r.lock.RUnlock()
	if found {
		r.lock.Lock()
		defer r.lock.Unlock()
		hd, found := r.handlerDataByResource[resource]
		if found {
			if _, err := hd.m.Destruct(resource); err != nil {
				return err
			}
			delete(r.handlerDataByResource, resource)
			if hd.m.Count() == 0 {
				if err := hd.m.Stop(); err != nil {
					return err
				}
				delete(r.handlerByOwnerCode, hd.oc)
			}
			m, err := r.MCP(resource)
			if err != nil {
				return err
			}
			if _, err := m.Construct(resource); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Router) findResource(source, id string) ([]interface{}, error) {
	var result []interface{}

	if id == source {
		r.lock.RLock()
		client, found := r.clients[id]
		r.lock.RUnlock()
		if found {
			result = append(result, client)
		} else {
			wrapper := &resourceWrapper{
				resource: id,
				router:   r,
			}
			result = append(result, wrapper)
		}
	}

	res := &resource.Resource{}
	if err := r.persister.Get(id, res); err != nil {
		return nil, err
	}
	m, err := r.MCP(res.Id)
	if err != nil {
		return nil, err
	}
	result = append(result, proc.ResourceProxy{
		SendRequest: m.SendRequest,
	})

	return result, nil
}

func (r *Router) createMCP(resourceId string) (*mcp.MCP, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	hd, found := r.handlerDataByResource[resourceId]
	if found {
		return hd.m, nil
	}

	res := &resource.Resource{}
	if err := r.persister.Get(resourceId, res); err != nil {
		return nil, err
	}
	if err := validator.Validate(res.Code); err != nil {
		return nil, err
	}
	oc, err := newOwnerCode(res.Owner, res.Code)
	if err != nil {
		return nil, err
	}

	m, found := r.handlerByOwnerCode[oc]
	if found {
		if _, err := m.Construct(res.Id); err != nil {
			return nil, err
		}
		r.handlerDataByResource[res.Id] = handlerData{
			oc: oc,
			m:  m,
		}
		return m, nil
	}

	m, err = mcp.New(res.Code, r.findResource)
	if err != nil {
		return nil, err
	}
	if err := m.Start(); err != nil {
		return nil, err
	}
	if _, err := m.Construct(res.Id); err != nil {
		return nil, err
	}

	r.handlerByOwnerCode[oc] = m
	r.handlerDataByResource[res.Id] = handlerData{
		oc: oc,
		m:  m,
	}
	return m, nil
}

func (r *Router) MCP(resourceId string) (*mcp.MCP, error) {
	r.lock.RLock()
	hd, found := r.handlerDataByResource[resourceId]
	r.lock.RUnlock()
	if !found {
		return r.createMCP(resourceId)
	}
	return hd.m, nil
}
