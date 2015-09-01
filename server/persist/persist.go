package persist

import (
	"fmt"
	"reflect"
)

var ErrNotFound = fmt.Errorf("Not found")

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

type errors []error

func (e errors) Error() string {
	return fmt.Sprintf("%+v", []error(e))
}

type F struct {
	val  reflect.Value
	m    map[string]interface{}
	errs errors
}

func NewF(tmpl interface{}) *F {
	f := &F{
		val: reflect.ValueOf(tmpl),
		m:   map[string]interface{}{},
	}
	if f.val.Kind() != reflect.Struct {
		f.errs = append(f.errs, fmt.Errorf("%v isn't a struct", f.val.Type()))
	}
	return f
}

func (f *F) Add(field string) *F {
	if len(f.errs) > 0 {
		return f
	}
	val := f.val.FieldByName(field)
	if val.IsValid() {
		f.m[field] = val.Interface()
	} else {
		f.errs = append(f.errs, fmt.Errorf("%v has no field %q", f.val.Type(), field))
	}
	return f
}

func (p *Persister) Find(filter *F, result interface{}) error {
	if len(filter.errs) > 0 {
		return filter.errs
	}

	resultType := reflect.TypeOf(result)
	if resultType.Kind() != reflect.Ptr || resultType.Elem().Kind() != reflect.Slice || resultType.Elem().Elem().Kind() != reflect.Struct {
		return fmt.Errorf("Result not pointer to slice of struct")
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
	Find(kind string, filter *F, result interface{}) error
	Transact(func(Backend) error) error
}
