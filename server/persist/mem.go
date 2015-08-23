package persist

import (
	"sync"

	"github.com/davecgh/go-spew/spew"
)

type Mem struct {
	node node
	lock sync.RWMutex
}

func NewMem() *Mem {
	return &Mem{}
}

func (m *Mem) Set(val string, keys ...string) error {
	defer spew.Dump(m)
	return m.node.set(val, keys...)
}

func (m *Mem) Get(keys ...string) (string, error) {
	return m.node.get(keys...)
}

type node struct {
	val *string
	m   map[string]node
}

func (n *node) set(val string, keys ...string) error {
	if len(keys) == 0 {
		n.val = &val
		return nil
	}
	if n.m == nil {
		n.m = map[string]node{}
	}
	next := n.m[keys[0]]
	defer func() {
		n.m[keys[0]] = next
	}()
	return next.set(val, keys[1:]...)
}

func (n *node) get(keys ...string) (string, error) {
	if len(keys) == 0 {
		if n.val == nil {
			return "", ErrNotFound
		}
		return *n.val, nil
	}
	if n.m == nil {
		return "", ErrNotFound
	}
	next := n.m[keys[0]]
	return next.get(keys[1:]...)
}
