package main

import (
	"github.com/zond/hackyhack/client/commands"
	"github.com/zond/hackyhack/client/events"
	"github.com/zond/hackyhack/client/util"
	"github.com/zond/hackyhack/proc/interfaces"
	"github.com/zond/hackyhack/proc/messages"
	"github.com/zond/hackyhack/proc/slave"
	"github.com/zond/hackyhack/proc/slave/delegator"
)

type handler struct {
	mcp              interfaces.MCP
	eventHandler     *events.DefaultHandler
	commandDelegator *delegator.Delegator
}

func New(m interfaces.MCP) interfaces.Describable {
	h := &handler{
		mcp: m,
		eventHandler: &events.DefaultHandler{
			M: m,
		},
	}
	h.commandDelegator = delegator.New(&commands.Default{
		M: m,
	})
	go func() {
		if err := util.Subscribe(m, &messages.Subscription{
			HandlerName: "Event",
		}); err != nil {
			util.Fatal(err)
		}
	}()
	return h
}

func (h *handler) Event(ctx *messages.Context, ev *messages.Event) bool {
	return h.eventHandler.Event(ctx, ev)
}

func (h *handler) HandleClientInput(s string) *messages.Error {
	verb, rest := util.SplitVerb(s)

	if verb != "" {
		cmd := util.Capitalize(verb)

		var merr *messages.Error
		if err := h.commandDelegator.Call(cmd, []string{rest}, []interface{}{merr}); err != nil {
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

func (h *handler) GetLongDesc() (string, *messages.Error) {
	return "An anonymous blob of logic.", nil
}

func (h *handler) GetShortDesc() (*messages.ShortDesc, *messages.Error) {
	return &messages.ShortDesc{
		Value: util.Capitalize("{{.Username}}"),
		Name:  true,
	}, nil
}

func main() {
	slave.Register(New)
}
