package main

import (
	"github.com/zond/hackyhack/client/commands"
	"github.com/zond/hackyhack/proc/interfaces"
	"github.com/zond/hackyhack/proc/slave"
	"github.com/zond/hackyhack/proc/slave/delegator"
)

type commandList struct {
	h *handler
}

func (c *commandList) L() {
	c.h.m.SendToClient("Darkness!\n")
}

type handler struct {
	m  interfaces.MCP
	ch *delegator.Delegator
}

func New(m interfaces.MCP) interfaces.Describable {
	h := &handler{
		m: m,
	}
	h.ch = delegator.New(&commandList{
		h: h,
	})
	return h
}

func (h *handler) HandleClientInput(s string) {
	parts := commands.SplitWhitespace(s)
	if len(parts) == 0 {
		return
	}

	var params []string
	if len(parts) > 1 {
		params = parts[1:]
	}

	cmd := commands.Capitalize(parts[0])

	err := h.ch.Call(cmd, params, nil)
	if err != nil {
		h.m.SendToClient(commands.Sprintf("Calling %q: %v\n", cmd, err.Error()))
	}
}

func (h *handler) GetShortDesc() string {
	return "anonymous"
}

func main() {
	slave.Register(New)
}
