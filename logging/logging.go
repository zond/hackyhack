package logging

import (
	"fmt"
	"time"
)

type Outputter func(string, ...interface{})

func (o Outputter) Trace(f string, i ...interface{}) func() {
	o(fmt.Sprintf("%s ->", f), i...)
	start := time.Now()
	return func() {
		o(fmt.Sprintf("%s <-\t(%v)", f, time.Now().Sub(start)), i...)
	}
}

func (o Outputter) Tracef(f func(bool) string) func() {
	o("%s", f(true))
	start := time.Now()
	return func() {
		o(fmt.Sprintf("%s <-\t(%v)", f(false), time.Now().Sub(start)))
	}
}
