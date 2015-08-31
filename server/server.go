package server

import (
	"net"

	"github.com/zond/hackyhack/server/client"
	"github.com/zond/hackyhack/server/persist"
	"github.com/zond/hackyhack/server/router"
)

type Server struct {
	persister *persist.Persister
	router    *router.Router
}

func New(p *persist.Persister) *Server {
	return &Server{
		persister: p,
		router:    router.New(p),
	}
}

func (s *Server) Serve(l net.Listener) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		client := client.New(s.persister, s.router)
		if err := client.Handle(conn); err != nil {
			return err
		}
	}
}
