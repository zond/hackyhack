package persist

import "errors"

var ErrNotFound = errors.New("Not found")

type Persister interface {
	Get(keys ...string) (value string, err error)
	Set(val string, keys ...string) error
}
