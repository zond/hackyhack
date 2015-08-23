package server

import (
	"net"

	"github.com/zond/hackyhack/server/client"
	"github.com/zond/hackyhack/server/persist"
)

type Server struct {
	persister persist.Persister
}

func New(p persist.Persister) *Server {
	return &Server{
		persister: p,
	}
}

func (s *Server) Serve(l net.Listener) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		client := client.New(s.persister)
		if err := client.Handle(conn); err != nil {
			return err
		}
	}
}
