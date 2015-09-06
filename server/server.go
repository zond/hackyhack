package server

import (
	"net"
	"net/http"

	"github.com/zond/hackyhack/server/client"
	"github.com/zond/hackyhack/server/persist"
	"github.com/zond/hackyhack/server/router"
	"github.com/zond/hackyhack/server/web"
)

const (
	realm = "hackyhack"
)

type Server struct {
	persister *persist.Persister
	router    *router.Router
	web       *web.Web
}

func New(p *persist.Persister) (*Server, error) {
	r, err := router.New(p)
	if err != nil {
		return nil, err
	}
	server := &Server{
		persister: p,
		router:    r,
		web:       web.New(p, r),
	}
	return server, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.web.ServeHTTP(w, r)
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
