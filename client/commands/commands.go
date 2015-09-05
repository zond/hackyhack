package commands

import (
	"github.com/zond/hackyhack/client/util"
	"github.com/zond/hackyhack/proc/interfaces"
	"github.com/zond/hackyhack/proc/messages"
)

type Default struct {
	M interfaces.MCP
}

func (d *Default) Edit(what string) {

}

func (d *Default) L(x string) *messages.Error {
	containerId, err := util.GetContainer(d.M, d.M.GetResource())
	if err != nil {
		return err
	}
	desc, err := util.GetLongDesc(d.M, containerId)
	if err != nil {
		if util.IsNoSuchMethod(err) {
			desc, err = util.GetShortDesc(d.M, containerId)
		}
		if err != nil {
			return err
		}
	}
	siblings, err := util.GetContent(d.M, containerId)
	if err != nil && !util.IsNoSuchMethod(err) {
		return err
	}
	descs, err := util.GetShortDescs(d.M, siblings)
	if err != nil {
		return err
	}
	return util.SendToClient(d.M, util.Sprintf("%v\n\n%v\n", desc, util.Enumerate(descs)))
}
