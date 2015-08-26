package lobby

import (
	"crypto/hmac"
	"regexp"
	"strings"

	"github.com/zond/gosafe"
	"github.com/zond/hackyhack/server/persist"
)

const (
	initialHandler = `
package main

import (
	"github.com/zond/hackyhack/proc/interfaces"
	"github.com/zond/hackyhack/proc/slave"
)

type handler struct {
	mcp interfaces.MCP
}

func new(m interfaces.MCP) interfaces.Named {
	return &handler{
		mcp: m,
	}
}

func (h *handler) GetName() string {
	return "anonymous"
}

func main() {
	slave.Register(new)
}
`
)

type Client interface {
	Send(string) error
	Authorize(string) error
}

type state int

const (
	welcome state = iota
	createUser
)

type user struct {
	username string
	password string
}

type gosafeHandler struct {
	cmd *gosafe.Cmd
}

func (h *gosafeHandler) Input(s string) error {
	result, err := h.cmd.Call("input", s)
	if err != nil {
		return err
	}
	if result != nil {
		return result.(error)
	}
	return nil
}

type Lobby struct {
	client    Client
	persister persist.Persister
	state     state
	user      user
	compiler  *gosafe.Compiler
}

func New(p persist.Persister, c Client) *Lobby {
	lobby := &Lobby{
		client:    c,
		persister: p,
		compiler:  gosafe.NewCompiler(),
	}
	lobby.compiler.Allow("github.com/zond/gosafe/child")
	return lobby
}

var loginReg = regexp.MustCompile("^login (\\w+) (\\w+)$")

func (l *Lobby) Handle(s string) error {
	switch l.state {
	case createUser:
		switch strings.ToLower(s) {
		case "y":
			if err := l.persister.Set(l.user.password, "users", l.user.username, "password"); err != nil {
				return err
			}
			if err := l.persister.Set(initialHandler, "users", l.user.username, "handler"); err != nil {
				return err
			}
			return l.client.Authorize(l.user.username)
		case "n":
			l.state = welcome
			return l.client.Send(`
Usage:
login USERNAME PASSWORD
`)
		}
		return l.client.Send(`
(y/n)
`)
	case welcome:
		if match := loginReg.FindStringSubmatch(s); match == nil {
			return l.client.Send(`
Usage:
login USERNAME PASSWORD
`)
		} else {
			password, err := l.persister.Get("users", match[1], "password")
			if err == persist.ErrNotFound {
				l.state = createUser
				l.user = user{
					username: match[1],
					password: match[2],
				}
				return l.client.Send(`
User not found, create? (y/n)
`)
			} else if err != nil {
				return err
			}
			if hmac.Equal([]byte(password), []byte(match[2])) {
				return l.client.Authorize(match[1])
			} else {
				return l.client.Send(`
Incorrect password.
`)
			}
		}
	}
	return nil
}

func (l *Lobby) Welcome() error {
	if err := l.client.Send(`
Welcome
`); err != nil {
		return err
	}
	return l.client.Send(`
Usage:
login USERNAME PASSWORD
`)
}
