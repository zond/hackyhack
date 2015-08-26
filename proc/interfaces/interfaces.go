package interfaces

type Named interface {
	Name() string
}

type Nameds []Named

type Container interface {
	Content() Nameds
}

type MCP interface {
	GetResourceId() string
	GetContainer() Container
	GetContent() Nameds
}
