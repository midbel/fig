package fig

import (
	"fmt"
)

type Resolver interface {
	Resolve(string) (interface{}, error)
}

type env struct {
	parent *env
	values map[string]interface{}
}

func emptyEnv() *env {
	return enclosedEnv(nil)
}

func enclosedEnv(parent *env) *env {
	return &env{
		parent: parent,
		values: make(map[string]interface{}),
	}
}

func combinedEnv(e ...*env) *env {
	return nil
}

func (e *env) Resolve(ident string) (interface{}, error) {
	return e.resolve(ident)
}

func (e *env) resolve(ident string) (interface{}, error) {
	v, ok := e.values[ident]
	if ok {
		return v, nil
	}
	if e.parent != nil {
		return e.parent.resolve(ident)
	}
	return nil, fmt.Errorf("%s not defined", ident)
}

func (e *env) define(ident string, value interface{}) {
	e.values[ident] = value
}

func (e *env) unwrap() *env {
	return e.parent
}
