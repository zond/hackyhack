package commands

import (
	"fmt"
	"log"

	"github.com/zond/hackyhack/proc/interfaces"
	"github.com/zond/hackyhack/proc/messages"
)

type Default struct {
	M interfaces.MCP
}

func (d *Default) tryCall(err error) bool {
	if perr, ok := err.(*messages.Error); ok && perr.Code == messages.ErrorCodeNoSuchMethod {
		return false
	} else if err != nil {
		d.M.Fatal(err)
	}
	return true
}

func (d *Default) L() {
	results := []string{""}
	if d.tryCall(d.M.Call(d.M.GetContainer(), messages.MethodGetLongDesc, nil, &results)) {
		d.M.SendToClient(fmt.Sprintf("%v\n", results[0]))
	}
	if d.tryCall(d.M.Call(d.M.GetContainer(), messages.MethodGetShortDesc, nil, &results)) {
		d.M.SendToClient(fmt.Sprintf("%v\n", results[0]))
	}
	log.Fatal("No short or long desc of container found")
}
