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
	Define(string, Value)
	assign(string, Value) error
	resolveLocal(string) (Value, error)
	resolveFunc(string) (Func, error)
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

// func chain(env ...Environment) Environment {
//
// }

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

func (e *Env) assign(str string, value Value) error {
	_, ok := e.values[str]
	if !ok {
		if e.parent != nil {
			return e.parent.assign(str, value)
		}
		return undefinedVariable(str)
	}
	e.values[str] = value
	return nil
}

func (e *Env) resolveLocal(str string) (Value, error) {
	return e.Resolve(str)
}

func (e *Env) resolveFunc(str string) (Func, error) {
	return Func{}, undefinedFunction(str)
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

func (e *env) Define(str string, v Value) {
	// nothing to do - immutable env
}

func (e *env) assign(str string, v Value) error {
	// nothing to do - immutable env
	return undefinedVariable(str)
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

func (e *env) resolveFunc(str string) (Func, error) {
	for _, obj := range e.list {
		fn, err := obj.getFunction(str)
		if err != nil {
			if errors.Is(err, ErrUndefined) {
				continue
			}
			return fn, err
		}
		return fn, nil
	}
	return Func{}, undefinedFunction(str)
}

func stringFromEnv(e Environment, str string) (string, error) {
	v, err := e.Resolve(str)
	if err != nil {
		return "", err
	}
	return toText(v)
}

func intFromEnv(e Environment, str string) (int64, error) {
	v, err := e.Resolve(str)
	if err != nil {
		return 0, err
	}
	return toInt(v)
}

func doubleFromEnv(e Environment, str string) (float64, error) {
	v, err := e.Resolve(str)
	if err != nil {
		return 0, err
	}
	return toFloat(v)
}

func boolFromEnv(e Environment, str string) (bool, error) {
	v, err := e.Resolve(str)
	if err != nil {
		return false, err
	}
	return v.isTrue(), nil
}

func sliceFromEnv(e Environment, str string) ([]Value, error) {
	v, err := e.Resolve(str)
	if err != nil {
		return nil, err
	}
	vs, ok := v.(Slice)
	if !ok {
		return nil, ErrIncompatible
	}
	return vs.inner, nil
}

func undefinedVariable(str string) error {
	return fmt.Errorf("%s: %w variable", str, ErrUndefined)
}

func undefinedFunction(str string) error {
	return fmt.Errorf("%s: %w function", str, ErrUndefined)
}
