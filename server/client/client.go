package client

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"github.com/zond/hackyhack/proc/mcp"
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

type mcpHandler struct {
	m          *mcp.MCP
	client     *Client
	username   string
	resourceId string
}

func (m *mcpHandler) HandleClientInput(s string) error {
	return m.m.Call(m.resourceId, "HandleClientInput", []string{s}, nil)
}

func (m *mcpHandler) resourceFinder(resourceId string) (interface{}, error) {
	if resourceId == m.resourceId {
		return &selfObject{
			h: m,
		}, nil
	}
	return nil, nil
}

type selfObject struct {
	h *mcpHandler
}

func (s *selfObject) SendToClient(msg string) {
	if err := s.h.client.Send(msg); err != nil {
		s.h.client.conn.Close()
	}
}

func (s *selfObject) GetContainer() string {
	containerId, err := s.h.client.persister.Get("resources", s.h.resourceId, "container")
	if err == persist.ErrNotFound {
		return ""
	} else if err != nil {
		if err := s.h.m.Stop(); err != nil {
			log.Fatal(err)
		}
	}
	return containerId
}

func (c *Client) Authorize(username, resourceId string) error {
	handler := &mcpHandler{
		client:     c,
		username:   username,
		resourceId: resourceId,
	}
	var err error
	if handler.m, err = c.router.MCP(handler.resourceFinder, "resources", resourceId, "handler"); err != nil {
		return err
	}
	if _, err = handler.m.Construct(resourceId); err != nil {
		return err
	}
	c.handler = handler
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
