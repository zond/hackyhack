package persist

import (
	"errors"
	"fmt"
	"reflect"
)

var ErrNotFound = errors.New("Not found")

type Persister struct {
	Backend Backend
}

func (p *Persister) Put(key string, value interface{}) error {
	valueType := reflect.TypeOf(value)
	if valueType.Kind() != reflect.Ptr || valueType.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("Value not pointer to struct")
	}
	return p.Backend.Put(valueType.Elem().Name(), key, value)
}

func (p *Persister) Get(key string, result interface{}) error {
	resultType := reflect.TypeOf(result)
	if resultType.Kind() != reflect.Ptr || resultType.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("Result not pointer to struct")
	}
	return p.Backend.Get(reflect.TypeOf(result).Elem().Name(), key, result)
}

func (p *Persister) Find(filter, result interface{}) error {
	resultType := reflect.TypeOf(result)
	if resultType.Kind() != reflect.Ptr || resultType.Elem().Kind() != reflect.Slice || resultType.Elem().Elem().Kind() != reflect.Struct {
		return fmt.Errorf("Result not pointer to slice of struct")
	}
	if reflect.TypeOf(filter).Kind() != reflect.Struct {
		return fmt.Errorf("Filter not struct type")
	}
	return p.Backend.Find(resultType.Elem().Elem().Name(), filter, result)
}

func (p *Persister) Transact(f func(*Persister) error) error {
	return p.Backend.Transact(func(b Backend) error {
		return f(&Persister{
			Backend: b,
		})
	})
}

type Backend interface {
	Put(kind, key string, value interface{}) error
	Get(kind, key string, value interface{}) error
	Find(kind string, filter, result interface{}) error
	Transact(func(Backend) error) error
}
