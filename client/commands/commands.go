package commands

import (
	"github.com/zond/hackyhack/client/util"
	"github.com/zond/hackyhack/proc/interfaces"
	"github.com/zond/hackyhack/proc/messages"
)

type Default struct {
	M interfaces.MCP
}

func (d *Default) Ident(what string) *messages.Error {
	matches, err := util.Identify(d.M, what)
	if err != nil {
		return err
	}
	return util.SendToClient(d.M, util.Sprintf("%+v\n", matches))
}

func (d *Default) L(what string) *messages.Error {
	containerId, err := util.GetContainer(d.M, d.M.GetResource())
	if err != nil {
		return err
	}
	shortDesc, err := util.GetShortDesc(d.M, containerId)
	if err != nil {
		return err
	}
	longDesc, err := util.GetLongDesc(d.M, containerId)
	if err != nil && !util.IsNoSuchMethod(err) {
		return err
	}
	siblings, err := util.GetContent(d.M, containerId)
	if err != nil && !util.IsNoSuchMethod(err) {
		return err
	}
	descs, err := util.GetShortDescs(d.M, siblings)
	if err != nil {
		return err
	}

	if longDesc != "" {
		return util.SendToClient(d.M, util.Sprintf("%v\n%v\n\n%v\n", shortDesc, longDesc, util.Enumerate(descs)))
	} else {
		return util.SendToClient(d.M, util.Sprintf("%v\n\n%v\n", shortDesc, util.Enumerate(descs)))
	}
}
