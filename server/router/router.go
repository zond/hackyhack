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
	"regexp"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/zond/hackyhack/logging"
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

type subWrapper struct {
	sub             *messages.Subscription
	compiledVerbReg *regexp.Regexp
	compiledMethReg *regexp.Regexp
}

func (w *resourceWrapper) Subscribe(sub *messages.Subscription) *messages.Error {
	compiledVerbReg, err := regexp.Compile(sub.VerbReg)
	if err != nil {
		return &messages.Error{
			Message: err.Error(),
			Code:    messages.ErrorCodeRegexp,
		}
	}
	compiledMethReg, err := regexp.Compile(sub.MethReg)
	if err != nil {
		return &messages.Error{
			Message: err.Error(),
			Code:    messages.ErrorCodeRegexp,
		}
	}
	w.router.RegisterSubscriber(w.resource, &subWrapper{
		sub:             sub,
		compiledVerbReg: compiledVerbReg,
		compiledMethReg: compiledMethReg,
	})
	return nil
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
	handlerLock           sync.RWMutex
	clientLock            sync.RWMutex
	clients               map[string]*clientWrapper
	subscriberLock        sync.RWMutex
	subscribers           map[string]*subWrapper
	debugHandler          logging.Outputter
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
		subscribers:           map[string]*subWrapper{},
		debugHandler: func(f string, i ...interface{}) {
			log.Print(spew.Sprintf(f, i...))
		},
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

func (r *Router) RegisterSubscriber(resource string, sub *subWrapper) {
	r.subscriberLock.Lock()
	defer r.subscriberLock.Unlock()
	r.subscribers[resource] = sub
}

func (r *Router) UnregisterSubscriber(resource string) {
	r.subscriberLock.Lock()
	defer r.subscriberLock.Unlock()
	delete(r.subscribers, resource)
}

func (r *Router) RegisterClient(resource string, client Client) {
	r.clientLock.Lock()
	defer r.clientLock.Unlock()
	r.clients[resource] = &clientWrapper{
		resourceWrapper: resourceWrapper{
			resource: resource,
			router:   r,
		},
		client: client,
	}
}

func (r *Router) UnregisterClient(resource string) {
	r.clientLock.Lock()
	defer r.clientLock.Unlock()
	delete(r.clients, resource)
}

func (r *Router) Restart(resource string) error {
	r.handlerLock.RLock()
	hd, found := r.handlerDataByResource[resource]
	r.handlerLock.RUnlock()
	if found {
		r.handlerLock.Lock()
		if err := func() error {
			defer r.handlerLock.Unlock()
			hd, found = r.handlerDataByResource[resource]
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
			}
			return nil
		}(); err != nil {
			return err
		}
		if found {
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
		r.clientLock.RLock()
		client, found := r.clients[id]
		r.clientLock.RUnlock()
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
		SendRequest: func(req *messages.Request) (*messages.Response, error) {
			resp, err := m.SendRequest(req)
			if err == nil {
				go r.broadcast(req)
			}
			return resp, err
		},
	})

	return result, nil
}

func (r *Router) broadcast(req *messages.Request) {
	defer r.debugHandler.Trace("Router#broadcast(%#v)", req)()

	res := &resource.Resource{}
	if err := r.persister.Get(req.Header.Source, res); err != nil {
		r.debugHandler("*** MISSING RESOURCE WHEN BROADCASTING %#v: %v ***", req, err)
		return
	}
	cont := &resource.Resource{}
	if err := r.persister.Get(res.Container, cont); err != nil {
		r.debugHandler("*** MISSING CONTAINER WHEN BROADCASTING %#v: %v ***", req, err)
		return
	}
	for _, res := range cont.Content {
		go func(res string) {
			r.subscriberLock.RLock()
			wrapper, found := r.subscribers[res]
			r.subscriberLock.RUnlock()
			if found {
				if req.Header.Verb.Matches(wrapper.compiledVerbReg) || wrapper.compiledMethReg.MatchString(req.Method) {
					m, err := r.MCP(res)
					if err != nil {
						r.debugHandler("*** BROKEN MCP WHEN BROADCASTING: %q ***", res)
						return
					}
					var cont bool
					if err := m.Call(res, res, wrapper.sub.HandlerName, []interface{}{
						&messages.Event{
							Type:    messages.EventTypeRequest,
							Request: req,
						},
					}, &[]interface{}{&cont}); err != nil || !cont {
						r.debugHandler("Unsubscribing %q (%v, %v)", err, cont)
						r.UnregisterSubscriber(res)
					}
				}
			}
		}(res)
	}
}

func (r *Router) createMCP(resourceId string) (*mcp.MCP, error) {
	r.handlerLock.Lock()
	defer r.handlerLock.Unlock()

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
	r.handlerLock.RLock()
	hd, found := r.handlerDataByResource[resourceId]
	r.handlerLock.RUnlock()
	if !found {
		return r.createMCP(resourceId)
	}
	return hd.m, nil
}
