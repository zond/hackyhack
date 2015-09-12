package interfaces

import "github.com/zond/hackyhack/proc/messages"

type Describable interface {
	GetShortDesc() (*messages.ShortDesc, *messages.Error)
}

type Subscriber interface {
	Event(*messages.Event) error
}

type Destructible interface {
	Destroy()
}

type MCP interface {
	GetResource() string
	Call(verb *messages.Verb, resourceId, method string, params, results interface{}) *messages.Error
}
