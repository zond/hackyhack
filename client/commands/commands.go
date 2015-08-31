package commands

import (
	"log"

	"github.com/zond/hackyhack/client/util"
	"github.com/zond/hackyhack/proc/interfaces"
	"github.com/zond/hackyhack/proc/messages"
)

type Default struct {
	M interfaces.MCP
}

func (d *Default) L() {
	containerId, err := util.GetContainer(d.M)
	if err != nil {
		return
	}
	var desc string
	if util.Success(d.M, d.M.Call(containerId, messages.MethodGetLongDesc, []string{d.M.GetResource()}, &[]interface{}{&desc})) {
		util.SendToClient(d.M, util.Sprintf("%v\n", desc))
		return
	}
	if util.Success(d.M, d.M.Call(containerId, messages.MethodGetShortDesc, []string{d.M.GetResource()}, &[]interface{}{&desc})) {
		util.SendToClient(d.M, util.Sprintf("%v\n", desc))
		return
	}
	log.Fatal("No short or long desc of container found")
}
