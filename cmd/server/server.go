package main

import (
	"flag"
	"log"
	"net"

	"github.com/zond/hackyhack/server"
	"github.com/zond/hackyhack/server/persist"
)

func main() {
	addr := flag.String("addr", ":6000", "Where to listen")

	flag.Parse()

	l, err := net.Listen("tcp", *addr)
	if err != nil {
		panic(err)
	}

	s := server.New(&persist.Persister{
		Backend: persist.NewMem(),
	})
	log.Fatal(s.Serve(l))
}
