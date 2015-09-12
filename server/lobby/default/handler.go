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
	mcp              interfaces.MCP
	commandDelegator *delegator.Delegator
}

func New(m interfaces.MCP) interfaces.Describable {
	h := &handler{
		mcp: m,
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

func (h *handler) Event(ev *messages.Event) bool {
	if ev.Type == messages.EventTypeRequest {
		if ev.Request.Header.Source == h.mcp.GetResource() {
			targetDesc, err := util.GetShortDesc(h.mcp, ev.Request.Resource)
			if err != nil {
				util.SendToClient(h.mcp, util.Sprintf("You %v something.", ev.Request.Header.Verb.SecondPerson))
			} else {
				util.SendToClient(h.mcp, util.Sprintf("You %v %v.", ev.Request.Header.Verb.SecondPerson, targetDesc.DefArticlize()))
			}
		} else {
			source := "something"
			sourceDesc, serr := util.GetShortDesc(h.mcp, ev.Request.Header.Source)
			if serr == nil {
				source = sourceDesc.DefArticlize()
			}
			target := "something"
			targetDesc, terr := util.GetShortDesc(h.mcp, ev.Request.Resource)
			if terr == nil {
				target = targetDesc.DefArticlize()
			}
			util.SendToClient(h.mcp, util.Capitalize(util.Sprintf("%v %v %v.", source, ev.Request.Header.Verb.ThirdPerson, target)))
		}
	} else {
		util.SendToClient(h.mcp, util.Sprintf("%+v", ev))
	}
	return true
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

func (h *handler) GetShortDesc() (*messages.ShortDesc, *messages.Error) {
	return &messages.ShortDesc{
		Value: util.Capitalize("{{.Username}}"),
		Name:  true,
	}, nil
}

func main() {
	slave.Register(New)
}
