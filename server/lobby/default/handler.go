package main

import (
	"github.com/zond/hackyhack/proc/interfaces"
	"github.com/zond/hackyhack/proc/slave"
)

type handler struct {
	m interfaces.MCP
}

func New(m interfaces.MCP) interfaces.Describable {
	return &handler{
		m: m,
	}
}

func (h *handler) GetShortDesc() string {
	return "anonymous"
}

func main() {
	slave.Register(New)
}
