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

func (h *handler) GetShortDesc(viewerId string) (string, *messages.Error) {
	return "the Void", nil
}

func (h *handler) GetLongDesc(viewerId string) (string, *messages.Error) {
	contentDescs, err := util.GetContentDescs(h.m)
	if err != nil {
		return "", err
	}
	return util.Sprintf("The infinite darkness of space.\n\n%v", util.Enumerate(contentDescs)), nil
}

func main() {
	slave.Register(New)
}
