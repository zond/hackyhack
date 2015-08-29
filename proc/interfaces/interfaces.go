package interfaces

type Describable interface {
	GetShortDesc(viewerId string) string
}

type Destructible interface {
	Destroy()
}

type MCP interface {
	Logf(string, ...interface{})
	Log(...interface{})
	Fatal(...interface{})
	Fatalf(string, ...interface{})
	SendToClient(string)
	GetResourceId() string
	GetContainer() string
	GetContent() []string
	Call(resourceId, method string, params, results interface{}) error
}
