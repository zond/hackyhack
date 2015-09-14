package events

import (
	"github.com/zond/hackyhack/client/util"
	"github.com/zond/hackyhack/proc/interfaces"
	"github.com/zond/hackyhack/proc/messages"
)

type DefaultHandler struct {
	M interfaces.MCP
}

func (h *DefaultHandler) Event(ctx *messages.Context, ev *messages.Event) bool {
	if ctx.Request.Header.Source != h.M.GetResource() {
		return true
	}
	switch ev.Type {
	case messages.EventTypeSay:
		subject := "something"
		verb := ""
		if ev.Source == h.M.GetResource() {
			subject = "you"
			verb = "say"
		} else {
			subjectDesc, serr := util.GetShortDesc(h.M, ev.Source)
			if serr == nil {
				subject = subjectDesc.DefArticlize()
			}
			verb = "says"
		}
		util.SendToClient(h.M, util.Capitalize(util.Sprintf("%v %v %q.\n", subject, verb, ev.Metadata[messages.MetadataPayload])))
	case messages.EventTypeDestruct:
		util.SendToClient(h.M, util.Capitalize(util.Sprintf("%v disappears.\n", ev.SourceShortDesc.IndefArticlize())))
	case messages.EventTypeConstruct:
		object := "something"
		objectDesc, err := util.GetShortDesc(h.M, ev.Source)
		if err == nil {
			object = objectDesc.IndefArticlize()
		}
		util.SendToClient(h.M, util.Capitalize(util.Sprintf("%v appears.\n", object)))
	case messages.EventTypeRequest:
		if util.DefaultAttentionLevels.Ignored(h.M, ev) {
			return true
		}
		subject := "something"
		verb := ""
		if ev.Request.Header.Source == h.M.GetResource() {
			verb = ev.Request.Header.Verb.SecondPerson
			subject = "you"
		} else {
			verb = ev.Request.Header.Verb.ThirdPerson
			subjectDesc, serr := util.GetShortDesc(h.M, ev.Request.Header.Source)
			if serr == nil {
				subject = subjectDesc.DefArticlize()
			}
		}
		object := "something"
		if ev.Request.Resource == h.M.GetResource() {
			if ev.Request.Header.Source == h.M.GetResource() {
				object = "yourself"
			} else {
				object = "you"
			}
		} else {
			objectDesc, err := util.GetShortDesc(h.M, ev.Request.Resource)
			if err == nil {
				object = objectDesc.DefArticlize()
			}
		}
		util.SendToClient(h.M, util.Capitalize(util.Sprintf("%v %v %v.\n", subject, verb, object)))
	default:
		util.SendToClient(h.M, util.Sprintf("%+v\n", ev))
	}
	return true
}
