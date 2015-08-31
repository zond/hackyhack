package client

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/zond/hackyhack/proc/mcp"
	"github.com/zond/hackyhack/proc/messages"
	"github.com/zond/hackyhack/server/lobby"
	"github.com/zond/hackyhack/server/persist"
	"github.com/zond/hackyhack/server/router"
	"github.com/zond/hackyhack/server/user"
)

type Handler interface {
	HandleClientInput(string) error
	UnregisterClient()
}

type Client struct {
	persister *persist.Persister
	router    *router.Router
	conn      net.Conn
	reader    *bufio.Reader
	handler   Handler
}

func New(p *persist.Persister, r *router.Router) *Client {
	return &Client{
		persister: p,
		router:    r,
	}
}

func (c *Client) Send(s string) error {
	_, err := io.WriteString(c.conn, s)
	return err
}

type mcpHandler struct {
	m      *mcp.MCP
	client *Client
	user   *user.User
}

func (m *mcpHandler) HandleClientInput(s string) error {
	var merr *messages.Error
	if err := m.m.Call(m.user.Resource, m.user.Resource, "HandleClientInput", []string{s}, &[]interface{}{&merr}); err != nil {
		return err
	}
	return merr.ToErr()
}

func (m *mcpHandler) UnregisterClient() {
	m.client.router.UnregisterClient(m.user.Resource)
}

func (c *Client) Authorize(user *user.User) error {
	handler := &mcpHandler{
		client: c,
		user:   user,
	}
	var err error
	if handler.m, err = c.router.MCP(user.Resource); err != nil {
		return err
	}
	if _, err = handler.m.Construct(user.Resource); err != nil {
		return err
	}
	c.handler = handler
	c.router.RegisterClient(user.Resource, c)
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
	defer c.handler.UnregisterClient()
	line, err := c.reader.ReadString('\n')
	for ; err == nil; line, err = c.reader.ReadString('\n') {
		line = strings.TrimSpace(line)
		if e := c.handler.HandleClientInput(line); e != nil {
			c.Send(fmt.Sprintf("%v\n", e.Error()))
		}
	}
	if err == io.EOF {
		err = nil
	}
	return err
}
