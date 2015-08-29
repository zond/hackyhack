package main

import (
	"github.com/zond/hackyhack/client/commands"
	"github.com/zond/hackyhack/client/util"
	"github.com/zond/hackyhack/proc/interfaces"
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

func (h *handler) HandleClientInput(s string) {
	parts := util.SplitWhitespace(s)
	if len(parts) == 0 {
		return
	}

	var params []string
	if len(parts) > 1 {
		params = parts[1:]
	}

	cmd := util.Capitalize(parts[0])

	err := h.ch.Call(cmd, params, nil)
	if err != nil {
		h.m.SendToClient(util.Sprintf("Calling %q: %v\n", cmd, err.Error()))
	}
}

func (h *handler) GetShortDesc(viewerId string) string {
	return "anonymous"
}

func main() {
	slave.Register(New)
}
