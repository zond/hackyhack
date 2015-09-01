package lobby

import (
	"crypto/hmac"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/zond/gosafe"
	"github.com/zond/hackyhack/proc/messages"
	"github.com/zond/hackyhack/server/persist"
	"github.com/zond/hackyhack/server/resource"
	"github.com/zond/hackyhack/server/user"
)

var initialHandler string

func init() {
	path := filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "zond", "hackyhack", "server", "lobby", "default", "handler.go")
	b, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("Unable to load default handler file %q: %v", path, err)
	}
	initialHandler = string(b)
	rand.Seed(time.Now().UnixNano())
}

type Client interface {
	Send(string) error
	Authorize(*user.User) error
}

type state int

const (
	welcome state = iota
	createUser
)

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
	persister *persist.Persister
	state     state
	user      *user.User
	compiler  *gosafe.Compiler
}

func New(p *persist.Persister, c Client) *Lobby {
	lobby := &Lobby{
		client:    c,
		persister: p,
		compiler:  gosafe.NewCompiler(),
	}
	lobby.compiler.Allow("github.com/zond/gosafe/child")
	return lobby
}

func (l *Lobby) UnregisterClient() {
}

var loginReg = regexp.MustCompile("^login (\\w+) (\\w+)$")

func (l *Lobby) HandleClientInput(s string) error {
	switch l.state {
	case createUser:
		switch strings.ToLower(s) {
		case "y":
			if err := l.persister.Transact(func(p *persist.Persister) error {
				if err := p.Put(l.user.Username, l.user); err != nil {
					return err
				}
				r := &resource.Resource{
					Id:        l.user.Resource,
					Code:      initialHandler,
					Container: messages.VoidResource,
				}
				if err := p.Put(r.Id, r); err != nil {
					return err
				}
				voidRes := &resource.Resource{}
				if err := p.Get(messages.VoidResource, voidRes); err != nil {
					return err
				}
				voidRes.Content = append(voidRes.Content, r.Id)
				if err := p.Put(messages.VoidResource, voidRes); err != nil {
					return err
				}
				return nil
			}); err != nil {
				return err
			}
			return l.client.Authorize(l.user)
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
			users := []user.User{}
			if err := l.persister.Find(persist.NewF(user.User{
				Username: match[1],
			}).Add("Username"), &users); err != nil {
				return err
			}
			if len(users) == 0 {
				l.state = createUser
				l.user = &user.User{
					Username: match[1],
					Password: match[2],
					Resource: fmt.Sprintf("%x%x", rand.Int63(), rand.Int63()),
				}
				return l.client.Send(`
User not found, create? (y/n)
`)
			}
			for index := range users {
				if hmac.Equal([]byte(match[2]), []byte(users[index].Password)) {
					return l.client.Authorize(&users[index])
				}
			}
			return l.client.Send(`
Incorrect password.
`)
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
