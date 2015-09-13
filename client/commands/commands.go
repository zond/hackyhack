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
	if len(matches) == 0 {
		util.SendToClient(d.M, "Identify what?\n")
	}
	for _, match := range matches {
		util.SendToClient(d.M, util.Sprintf("%+v\n", match))
	}

	return nil
}

func (d *Default) look() *messages.Error {
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
		util.SendToClient(d.M, util.Sprintf("%v\n%v\n\n%v\n", util.Capitalize(shortDesc.IndefArticlize()), longDesc, descs.Enumerate()))
	} else {
		util.SendToClient(d.M, util.Sprintf("%v\n\n%v\n", util.Capitalize(shortDesc.IndefArticlize()), descs.Enumerate()))
	}

	return nil
}

func (d *Default) L(what string) *messages.Error {
	if what == "" {
		return d.look()
	}

	matches, err := util.Identify(d.M, what)
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		util.SendToClient(d.M, "Look at what?\n")
	}
	for _, resource := range matches {
		shortDesc, err := util.GetShortDesc(d.M, resource)
		if err != nil {
			return err
		}
		longDesc, err := util.GetLongDesc(d.M, resource)
		if err != nil && !util.IsNoSuchMethod(err) {
			return err
		}
		if longDesc != "" {
			util.SendToClient(d.M, util.Sprintf("%v\n%v\n", util.Capitalize(shortDesc.IndefArticlize()), longDesc))
		} else {
			util.SendToClient(d.M, util.Sprintf("%v\n", shortDesc.IndefArticlize()))
		}
	}
	return nil
}
