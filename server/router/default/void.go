package main

import (
	"github.com/zond/hackyhack/client/util"
	"github.com/zond/hackyhack/proc/interfaces"
	"github.com/zond/hackyhack/proc/messages"
	"github.com/zond/hackyhack/proc/slave"
)

type handler struct {
	m interfaces.MCP
}

func New(m interfaces.MCP) interfaces.Describable {
	h := &handler{
		m: m,
	}
	return h
}

func (h *handler) GetShortDesc() (string, *messages.Error) {
	return "The Void", nil
}

func (h *handler) GetLongDesc() (string, *messages.Error) {
	return "The infinite darkness of space.", nil
}

func (h *handler) GetContent() ([]string, *messages.Error) {
	return util.GetContent(h.m, h.m.GetResource())
}

func main() {
	slave.Register(New)
}
