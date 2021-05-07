package fig

import (
	"errors"
	"fmt"
	"sync"
)

var (
	ErrUndefined = errors.New("undefined")
	ErrResolved  = errors.New("variable cannot be resolved")
)

type Environment interface {
	Resolve(string) (Value, error)
	resolveLocal(string) (Value, error)
}

type Env struct {
	rw     sync.RWMutex
	values map[string]Value
	parent Environment
}

func EmptyEnv() *Env {
	return EnclosedEnv(nil)
}

func EnclosedEnv(env Environment) *Env {
	e := Env{
		values: make(map[string]Value),
		parent: env,
	}
	return &e
}

func (e *Env) Delete(str string) {
	e.rw.Lock()
	defer e.rw.Unlock()
	delete(e.values, str)
}

func (e *Env) Define(str string, value Value) {
	e.rw.Lock()
	defer e.rw.Unlock()
	e.values[str] = value
}

func (e *Env) Resolve(str string) (Value, error) {
	e.rw.RLock()
	defer e.rw.RUnlock()
	v, ok := e.values[str]
	if ok {
		return v, nil
	}
	if e.parent != nil {
		return e.parent.Resolve(str)
	}
	return nil, undefinedVariable(str)
}

func (e *Env) resolveLocal(str string) (Value, error) {
	return e.Resolve(str)
}

type env struct {
	parent Environment
	list   []*Object
}

func createEnv(list []*Object, other Environment) Environment {
	e := env{
		list:   list,
		parent: other,
	}
	return &e
}

func (e *env) Resolve(str string) (Value, error) {
	if e.parent == nil {
		return nil, undefinedVariable(str)
	}
	return e.parent.Resolve(str)
}

func (e *env) resolveLocal(str string) (Value, error) {
	for i, obj := range e.list {
		opt, err := obj.getOption(str)
		if err != nil {
			if errors.Is(err, ErrUndefined) {
				continue
			}
			return nil, err
		}
		e.list[i].unregister(str)
		v, err := opt.expr.Eval(createEnv(e.list[i:], e.parent))
		if err == nil {
			e.list[i].replace(opt, v)
		}
		return v, err
	}
	return nil, undefinedVariable(str)
}

func undefinedVariable(str string) error {
	return fmt.Errorf("%s: %w variable", str, ErrUndefined)
}

func undefinedFunction(str string) error {
	return fmt.Errorf("%s: %w function", str, ErrUndefined)
}
