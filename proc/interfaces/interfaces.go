package interfaces

type Describable interface {
	GetShortDesc() string
}

type Destructible interface {
	Destroy()
}

type MCP interface {
	GetResourceId() string
	GetContainer() string
	GetContent() []string
}
