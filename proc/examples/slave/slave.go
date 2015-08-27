package main

import (
	"github.com/zond/hackyhack/proc/interfaces"
	"github.com/zond/hackyhack/proc/slave"
)

type handler struct {
	mcp interfaces.MCP
}

func New(m interfaces.MCP) interfaces.Describable {
	return &handler{
		mcp: m,
	}
}

func (h *handler) GetShortDesc() string {
	return slave.Sprintf("anonymous with %+v in my pockets", h.mcp.GetContent())
}

func main() {
	slave.Register(New)
}
