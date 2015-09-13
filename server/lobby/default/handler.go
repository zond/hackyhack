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
		if util.DefaultAttentionLevels.Ignored(h.mcp, ev) {
			return true
		}
		subject := "something"
		verb := ""
		if ev.Request.Header.Source == h.mcp.GetResource() {
			verb = ev.Request.Header.Verb.SecondPerson
			subject = "you"
		} else {
			verb = ev.Request.Header.Verb.ThirdPerson
			subjectDesc, serr := util.GetShortDesc(h.mcp, ev.Request.Header.Source)
			if serr == nil {
				subject = subjectDesc.DefArticlize()
			}
		}
		object := "something"
		if ev.Request.Resource == h.mcp.GetResource() {
			if ev.Request.Header.Source == h.mcp.GetResource() {
				object = "yourself"
			} else {
				object = "you"
			}
		} else {
			objectDesc, err := util.GetShortDesc(h.mcp, ev.Request.Resource)
			if err == nil {
				object = objectDesc.DefArticlize()
			}
		}
		util.SendToClient(h.mcp, util.Capitalize(util.Sprintf("%v %v %v.\n", subject, verb, object)))
	} else {
		util.SendToClient(h.mcp, util.Sprintf("%+v\n", ev))
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
