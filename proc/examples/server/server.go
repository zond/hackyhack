package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/zond/hackyhack/proc/mcp"
)

type host struct{}

func (h *host) GetContainer() string {
	return "the void"
}

func (h *host) GetContent() []string {
	return nil
}

func main() {
	slave := flag.String("slave", "", "Path to the file to run as a slave.")

	flag.Parse()

	if *slave == "" {
		flag.Usage()
		os.Exit(1)
	}

	slaveBytes, err := ioutil.ReadFile(*slave)
	if err != nil {
		panic(err)
	}
	m, err := mcp.New(string(slaveBytes), func(string) (interface{}, error) {
		return &host{}, nil
	})
	if err != nil {
		panic(err)
	}

	if err := m.Start(); err != nil {
		panic(err)
	}

	if _, err := m.Construct("foo"); err != nil {
		panic(err)
	}

	resp := []string{""}
	if err := m.Call("foo", "GetShortDesc", nil, &resp); err != nil {
		panic(err)
	}
	fmt.Printf("shortdesc is %v\n", resp[0])

	c := make(chan bool)
	<-c

}
