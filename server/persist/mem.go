package persist

import (
	"fmt"
	"reflect"
	"sync"
)

type Mem struct {
	lock sync.RWMutex
	m    map[[2]string]interface{}
}

func NewMem() *Mem {
	return &Mem{
		m: map[[2]string]interface{}{},
	}
}

func (m *Mem) cpy(a, b interface{}) error {
	valA := reflect.ValueOf(a)
	valB := reflect.ValueOf(b)
	typA := valA.Type()
	typB := valB.Type()
	if typA != typB {
		return fmt.Errorf("Incompatible types; %v and %v", valA.Type(), valB.Type())
	}
	if typB.Kind() != reflect.Ptr {
		return fmt.Errorf("Not pointer types; %v and %v", valA.Type(), valB.Type())
	}
	valB.Elem().Set(valA.Elem())
	return nil
}

func (m *Mem) Transact(f func(p Backend) error) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	return f(&Mem{
		m: m.m,
	})
}

func (m *Mem) Get(kind, key string, value interface{}) error {
	m.lock.RLock()
	defer m.lock.RUnlock()
	val, found := m.m[[2]string{kind, key}]
	if !found {
		return ErrNotFound
	}
	return m.cpy(val, value)
}

func (m *Mem) Put(kind, key string, value interface{}) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.m[[2]string{kind, key}] = value
	return nil
}

func (m *Mem) matches(filter *F, val reflect.Value) bool {
	for name, wanted := range filter.m {
		field := val.FieldByName(name)
		if !field.IsValid() {
			return false
		}
		if field.Interface() != wanted {
			return false
		}
	}
	return true
}

func (m *Mem) Find(kind string, filter *F, result interface{}) error {
	resultVal := reflect.ValueOf(result).Elem()
	m.lock.RLock()
	defer m.lock.RUnlock()
	for k, v := range m.m {
		if k[0] == kind {
			vVal := reflect.ValueOf(v).Elem()
			if vVal.Type() != resultVal.Type().Elem() {
				return fmt.Errorf("Incompatible types; %v and %v", vVal.Type(), resultVal.Type().Elem())
			}
			if m.matches(filter, vVal) {
				resultVal.Set(reflect.Append(resultVal, vVal))
			}
		}
	}
	return nil
}
