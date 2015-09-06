package server

import (
	"crypto/hmac"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/zond/hackyhack/server/client"
	"github.com/zond/hackyhack/server/dav"
	"github.com/zond/hackyhack/server/persist"
	"github.com/zond/hackyhack/server/router"
	"github.com/zond/hackyhack/server/user"
	"golang.org/x/net/webdav"
)

const (
	realm = "hackyhack"
)

type Server struct {
	persister *persist.Persister
	router    *router.Router
}

func New(p *persist.Persister) (*Server, error) {
	r, err := router.New(p)
	if err != nil {
		return nil, err
	}
	return &Server{
		persister: p,
		router:    r,
	}, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.TLS == nil {
		newURL := r.URL
		newURL.Scheme = "https"
		http.Redirect(w, r, newURL.String(), 301)
		return
	}

	username, passwd, ok := r.BasicAuth()
	if !ok {
		w.Header().Set("WWW-Authenticate", fmt.Sprintf("Basic realm=%q", realm))
		http.Error(w, "Unauthenticated", 401)
		return
	}

	users := []user.User{}
	if err := s.persister.Find(persist.NewF(user.User{
		Username: username,
	}).Add("Username"), &users); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if len(users) == 0 {
		http.Error(w, "Unauthenticated", 401)
		return
	}

	var user *user.User
	for _, found := range users {
		if hmac.Equal([]byte(found.Password), []byte(passwd)) {
			user = &found
			break
		}
	}

	if user == nil {
		http.Error(w, "Unauthenticated", 401)
		return
	}

	handler := &webdav.Handler{
		FileSystem: dav.New(s.persister, s.router, user),
		LockSystem: webdav.NewMemLS(),
		Logger: func(r *http.Request, err error) {
			log.Printf("DAV\t%v\t%v", r.URL.String(), err)
		},
	}

	handler.ServeHTTP(w, r)
}

func (s *Server) ServeLogin(l net.Listener) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		client := client.New(s.persister, s.router)
		go client.Handle(conn)
	}
}
