package router

import (
	"errors"

	"github.com/zond/hackyhack/proc/mcp"
)

var errNotFound = errors.New("Not found")

type cache map[string]interface{}

func (m cache) set(val *mcp.MCP, keys ...string) {
	if len(keys) == 1 {
		m[keys[0]] = val
		return
	}
	current, found := m[keys[0]]
	currentM, ok := current.(cache)
	if found && ok {
		currentM.set(val, keys[1:]...)
		return
	}
	currentM = cache{}
	m[keys[0]] = currentM
	currentM.set(val, keys[1:]...)
}

func (m cache) get(keys ...string) (*mcp.MCP, error) {
	value, found := m[keys[0]]
	if !found {
		return nil, errNotFound
	}
	if len(keys) == 1 {
		valueH, ok := value.(*mcp.MCP)
		if !ok {
			return nil, errNotFound
		}
		return valueH, nil
	}
	valueM, ok := value.(cache)
	if !ok {
		return nil, errNotFound
	}
	return valueM.get(keys[1:]...)
}
