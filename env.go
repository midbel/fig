package fig

import (
	"fmt"
)

type Resolver interface {
	Resolve(string) (interface{}, error)
}

type Env struct {
	parent *Env
	values map[string]interface{}
}

func EmptyEnv() *Env {
	return EnclosedEnv(nil)
}

func EnclosedEnv(parent *Env) *Env {
	return &Env{
		parent: parent,
		values: make(map[string]interface{}),
	}
}

func (e *Env) Resolve(ident string) (interface{}, error) {
	return e.resolve(ident)
}

func (e *Env) resolve(ident string) (interface{}, error) {
	v, ok := e.values[ident]
	if ok {
		return v, nil
	}
	if e.parent != nil {
		return e.parent.resolve(ident)
	}
	return nil, fmt.Errorf("%s not defined", ident)
}

func (e *Env) Define(ident string, value interface{}) {
	e.define(ident, value)
}

func (e *Env) define(ident string, value interface{}) {
	e.values[ident] = value
}

func (e *Env) unwrap() *Env {
	return e.parent
}

type nested struct {
	Resolver
	env *Env
}

func wrapResolver(env *Env, res Resolver) Resolver {
	return nested{
		env:      env,
		Resolver: res,
	}
}

func (n nested) Resolve(ident string) (interface{}, error) {
	v, err := n.Resolver.Resolve(ident)
	if err == nil {
		return v, nil
	}
	return n.env.Resolve(ident)
}
