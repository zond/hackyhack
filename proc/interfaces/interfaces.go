package interfaces

type Describable interface {
	GetShortDesc() string
}

type Destructible interface {
	Destroy()
}

type MCP interface {
	Log(string)
	GetResourceId() string
	GetContainer() string
	GetContent() []string
}
