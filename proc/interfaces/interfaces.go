package interfaces

type Describable interface {
	GetShortDesc() string
}

type Destructible interface {
	Destroy()
}

type MCP interface {
	Logf(string, ...interface{})
	Log(...interface{})
	SendToClient(string)
	GetResourceId() string
	GetContainer() string
	GetContent() []string
}
