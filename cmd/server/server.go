package main

import (
	"flag"
	"log"
	"net"

	"github.com/zond/hackyhack/server"
	"github.com/zond/hackyhack/server/persist"
)

func main() {
	loginAddr := flag.String("loginAddr", ":6000", "Where to listen for logins")

	flag.Parse()

	s, err := server.New(&persist.Persister{
		Backend: persist.NewMem(),
	})

	loginListener, err := net.Listen("tcp", *loginAddr)
	if err != nil {
		panic(err)
	}

	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(s.ServeLogin(loginListener))
}
