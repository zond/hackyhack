package interfaces

type Named interface {
	GetName() string
}

type MCP interface {
	GetResourceId() string
	GetContainer() string
	GetContent() []string
}
