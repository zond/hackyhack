package interfaces

import "github.com/zond/hackyhack/proc/messages"

type Describable interface {
	GetShortDesc(viewerId string) (string, *messages.Error)
}

type Destructible interface {
	Destroy()
}

type MCP interface {
	GetResource() string
	Call(resourceId, method string, params, results interface{}) *messages.Error
}
