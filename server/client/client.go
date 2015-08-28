package client

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/zond/hackyhack/server/lobby"
	"github.com/zond/hackyhack/server/persist"
	"github.com/zond/hackyhack/server/router"
)

type Handler interface {
	HandleClientInput(string) error
}

type Client struct {
	persister persist.Persister
	router    *router.Router
	conn      net.Conn
	reader    *bufio.Reader
	handler   Handler
}

func New(p persist.Persister, r *router.Router) *Client {
	return &Client{
		persister: p,
		router:    r,
	}
}

func (c *Client) Send(s string) error {
	_, err := io.WriteString(c.conn, s)
	return err
}

func (c *Client) Authorize(username string) error {
	_, err := c.router.MCP("users", username, "handler")
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) Handle(conn net.Conn) error {
	c.conn = conn
	c.reader = bufio.NewReader(conn)
	lobby := lobby.New(c.persister, c)
	if err := lobby.Welcome(); err != nil {
		return err
	}
	c.handler = lobby
	line, err := c.reader.ReadString('\n')
	for ; err == nil; line, err = c.reader.ReadString('\n') {
		line = strings.TrimSpace(line)
		if err := c.handler.HandleClientInput(line); err != nil {
			return c.Send(fmt.Sprintf("%v\n", err.Error()))
		}
	}
	if err == io.EOF {
		err = nil
	}
	return err
}
