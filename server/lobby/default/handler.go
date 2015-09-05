package main

import (
	"github.com/zond/hackyhack/client/commands"
	"github.com/zond/hackyhack/client/util"
	"github.com/zond/hackyhack/proc/interfaces"
	"github.com/zond/hackyhack/proc/messages"
	"github.com/zond/hackyhack/proc/slave"
	"github.com/zond/hackyhack/proc/slave/delegator"
)

type handler struct {
	m  interfaces.MCP
	ch *delegator.Delegator
}

func New(m interfaces.MCP) interfaces.Describable {
	h := &handler{
		m: m,
	}
	h.ch = delegator.New(&commands.Default{
		M: m,
	})
	return h
}

func (h *handler) HandleClientInput(s string) *messages.Error {
	verb, rest := util.SplitVerb(s)

	if verb != "" {
		cmd := util.Capitalize(verb)

		var merr *messages.Error
		if err := h.ch.Call(cmd, []string{rest}, []interface{}{merr}); err != nil {
			return &messages.Error{
				Message: err.Error(),
			}
		}
		if merr != nil {
			return merr
		}
	}

	return nil
}

func (h *handler) GetShortDesc() (string, *messages.Error) {
	return "anonymous", nil
}

func main() {
	slave.Register(New)
}
