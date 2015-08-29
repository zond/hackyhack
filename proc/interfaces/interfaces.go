package interfaces

type Describable interface {
	GetShortDesc() string
}

type Destructible interface {
	Destroy()
}

type MCP interface {
	Log(string)
	SendToClient(string)
	GetResourceId() string
	GetContainer() string
	GetContent() []string
}
