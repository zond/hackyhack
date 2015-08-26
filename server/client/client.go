package client

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/zond/hackyhack/server/lobby"
	"github.com/zond/hackyhack/server/persist"
)

type Handler interface {
	HandleClientInput(string) error
}

type Client struct {
	persister persist.Persister
	conn      net.Conn
	reader    *bufio.Reader
	handler   Handler
}

func New(p persist.Persister) *Client {
	return &Client{
		persister: p,
	}
}

func (c *Client) Send(s string) error {
	_, err := io.WriteString(c.conn, s)
	return err
}

func (c *Client) Authorize(username string) error {
	fmt.Println("authorized as", username)
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
			return err
		}
	}
	if err == io.EOF {
		err = nil
	}
	return err
}
