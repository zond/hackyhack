package main

import (
	"github.com/zond/hackyhack/proc/interfaces"
	"github.com/zond/hackyhack/proc/slave"
	"github.com/zond/hackyhack/proc/slave/commands"
)

type commandList struct {
	h *handler
}

func (c *commandList) L() string {
	return "darkness!"
}

type handler struct {
	m  interfaces.MCP
	ch *commands.Handler
}

func New(m interfaces.MCP) interfaces.Describable {
	h := &handler{
		m: m,
	}
	h.ch = commands.New(&commandList{
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

	result := []string{""}
	err := h.ch.Call(cmd, params, result)
	if err == nil {
		h.m.SendToClient(commands.Sprintf("%v\n", result[0]))
	} else {
		h.m.SendToClient(commands.Sprintf("%v\n", err.Error()))
	}
}

func (h *handler) GetShortDesc() string {
	return "anonymous"
}

func main() {
	slave.Register(New)
}
