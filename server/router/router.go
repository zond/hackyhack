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
	"github.com/zond/hackyhack/proc/errors"
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
	sub                  *messages.Subscription
	compiledVerbReg      *regexp.Regexp
	compiledMethReg      *regexp.Regexp
	compiledEventTypeReg *regexp.Regexp
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
	compiledEventTypeReg, err := regexp.Compile(sub.EventTypeReg)
	if err != nil {
		return &messages.Error{
			Message: err.Error(),
			Code:    messages.ErrorCodeRegexp,
		}
	}
	w.router.RegisterSubscriber(w.resource, &subWrapper{
		sub:                  sub,
		compiledVerbReg:      compiledVerbReg,
		compiledMethReg:      compiledMethReg,
		compiledEventTypeReg: compiledEventTypeReg,
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

func (w *resourceWrapper) EmitEvent(ev *messages.Event) *messages.Error {
	if ev.Type == messages.EventTypeRequest {
		return &messages.Error{
			Message: "Can't emit Request events.",
			Code:    messages.ErrorCodeEventType,
		}
	}
	ev.Request = nil
	ev.Source = w.resource
	ev.SourceShortDesc = nil
	res := &resource.Resource{}
	if err := w.router.persister.Get(w.resource, res); err != nil {
		return &messages.Error{
			Message: "Can't find emitting resource.",
			Code:    messages.ErrorCodeNoSuchResource,
		}
	}
	go w.router.Broadcast(res.Container, ev)
	return nil
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

func (r *Router) Decomission(resourceId string) (bool, error) {
	r.handlerLock.RLock()
	hd, found := r.handlerDataByResource[resourceId]
	r.handlerLock.RUnlock()
	if found {
		m, err := r.MCP(resourceId)
		if err != nil {
			return false, err
		}
		var sd *messages.ShortDesc
		var merr *messages.Error
		if err := m.Call(resourceId, resourceId, messages.MethodGetShortDesc, nil, &[]interface{}{&sd, &merr}); err != nil {
			return false, err
		}
		if merr != nil {
			return false, merr.ToErr()
		}
		r.handlerLock.Lock()
		if err := func() error {
			defer r.handlerLock.Unlock()
			hd, found = r.handlerDataByResource[resourceId]
			if found {
				res := &resource.Resource{}
				if err := r.persister.Get(resourceId, res); err != nil {
					return err
				}
				if _, err := hd.m.Destruct(resourceId); err != nil {
					return err
				}
				go r.Broadcast(res.Container, &messages.Event{
					Type:            messages.EventTypeDestruct,
					Source:          resourceId,
					SourceShortDesc: sd,
				})
				delete(r.handlerDataByResource, resourceId)
				if hd.m.Count() == 0 {
					if err := hd.m.Stop(); err != nil {
						return err
					}
					delete(r.handlerByOwnerCode, hd.oc)
				}
			}
			return nil
		}(); err != nil {
			return false, err
		}
	}
	return found, nil
}

func (r *Router) Restart(resource string) error {
	found, err := r.Decomission(resource)
	if err != nil {
		return err
	}
	if found {
		_, err := r.MCP(resource)
		if err != nil {
			return err
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

	src := &resource.Resource{}
	if err := r.persister.Get(source, src); err != nil {
		return nil, err
	}

	res := &resource.Resource{}
	if err := r.persister.Get(id, res); err != nil {
		return nil, err
	}

	if src.Container != res.Id && res.Container != src.Id && src.Container != res.Container {
		return nil, errors.ErrUnavailableResource
	}

	m, err := r.MCP(res.Id)
	if err != nil {
		return nil, err
	}
	result = append(result, proc.ResourceProxy{
		SendRequest: func(req *messages.Request) (*messages.Response, error) {
			resp, err := m.SendRequest(req)
			if err == nil {
				go r.broadcastRequest(req)
			}
			return resp, err
		},
	})

	return result, nil
}

func (r *Router) Broadcast(container string, event *messages.Event) {
	defer r.debugHandler.Trace("Router#Broadcast(%q, %#v)", container, event)()

	cont := &resource.Resource{}
	if err := r.persister.Get(container, cont); err != nil {
		r.debugHandler("*** MISSING CONTAINER WHEN BROADCASTING %#v in %q: %v ***", event, container, err)
		return
	}

	for _, res := range cont.Content {
		go func(res string) {
			r.subscriberLock.RLock()
			wrapper, found := r.subscribers[res]
			r.subscriberLock.RUnlock()
			if found {
				matches := false
				if event.Type == messages.EventTypeRequest {
					matches =
						event.Request.Header.Verb.Matches(wrapper.compiledVerbReg) ||
							wrapper.compiledMethReg.MatchString(event.Request.Method) ||
							wrapper.compiledEventTypeReg.MatchString(string(event.Type))
				} else {
					matches = wrapper.compiledEventTypeReg.MatchString(string(event.Type))
				}
				if matches {
					m, err := r.MCP(res)
					if err != nil {
						r.debugHandler("*** BROKEN MCP WHEN BROADCASTING: %q ***", res)
						return
					}
					var cont bool
					if err := m.Call(res, res, wrapper.sub.HandlerName, []interface{}{
						event,
					}, &[]interface{}{&cont}); err != nil || !cont {
						r.debugHandler("Unsubscribing %q (%v, %v)", err, cont)
						r.UnregisterSubscriber(res)
					}
				}
			}
		}(res)
	}
}

func (r *Router) broadcastRequest(req *messages.Request) {
	res := &resource.Resource{}
	if err := r.persister.Get(req.Header.Source, res); err != nil {
		r.debugHandler("*** MISSING RESOURCE WHEN BROADCASTING %#v: %v ***", req, err)
		return
	}

	r.Broadcast(res.Container, &messages.Event{
		Source:  req.Header.Source,
		Type:    messages.EventTypeRequest,
		Request: req,
	})
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
		if err := r.construct(m, res); err != nil {
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
	if err := r.construct(m, res); err != nil {
		return nil, err
	}

	r.handlerByOwnerCode[oc] = m
	r.handlerDataByResource[res.Id] = handlerData{
		oc: oc,
		m:  m,
	}
	return m, nil
}

func (r *Router) construct(m *mcp.MCP, res *resource.Resource) error {
	if _, err := m.Construct(res.Id); err != nil {
		return err
	}
	if res.Container != "" {
		go r.Broadcast(res.Container, &messages.Event{
			Type:   messages.EventTypeConstruct,
			Source: res.Id,
		})
	}
	return nil
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
