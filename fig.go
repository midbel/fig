package fig

import (
	"errors"
	"fmt"
	"io"
	"time"
)

var ErrUndefined = errors.New("undefined")

type Environment interface {
	Resolve(string) (Value, error)
}

type Env struct {
	values map[string]Value
	parent Environment
}

func EmptyEnv() Environment {
	return EnclosedEnv(nil)
}

func EnclosedEnv(env Environment) Environment {
	e := Env{
		values: make(map[string]Value),
		parent: env,
	}
	return &e
}

func (e *Env) Delete(str string) {
	delete(e.values, str)
}

func (e *Env) Define(str string, value Value) {
	e.values[str] = value
}

func (e *Env) Resolve(str string) (Value, error) {
	v, ok := e.values[str]
	if ok {
		return v, nil
	}
	if e.parent != nil {
		return e.parent.Resolve(str)
	}
	return nil, undefinedVariable(str)
}

type env struct {
	list []*Object
}

func createEnv(list []*Object) Environment {
	size := len(list)
	for i, j := 0, size-1; i < size/2; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
	e := env{
		list: list,
	}
	return &e
}

func (e *env) Resolve(str string) (Value, error) {
	for i, obj := range e.list {
		n, ok := obj.nodes[str]
		if !ok {
			continue
		}
		opt, ok := n.(Option)
		if !ok {
			return nil, fmt.Errorf("%s: not an option", str)
		}
		if i == 0 {
			continue
		}
		return opt.expr.Eval(createEnv(e.list[:i+1]))
	}
	return nil, undefinedVariable(str)
}

type Document struct {
	root *Object
}

func ParseDocument(r io.Reader) (*Document, error) {
	root, err := Parse(r)
	if err != nil {
		return nil, err
	}
	doc := Document{
		root: root,
	}
	return &doc, nil
}

func (d *Document) Expr(paths ...string) (Expr, error) {
	e, _, err := d.find(paths...)
	return e, err
}

func (d *Document) Int(paths ...string) (int64, error) {
	v, err := d.eval(paths...)
	if err != nil {
		return 0, err
	}
	i, err := v.toInt()
	if err != nil {
		return 0, err
	}
	return toInt(i)
}

func (d *Document) Float(paths ...string) (float64, error) {
	v, err := d.eval(paths...)
	if err != nil {
		return 0, err
	}
	f, err := v.toDouble()
	if err != nil {
		return 0, err
	}
	return toFloat(f)
}

func (d *Document) Bool(paths ...string) (bool, error) {
	v, err := d.eval(paths...)
	if err != nil {
		return false, err
	}
	b, err := v.toBool()
	if err != nil {
		return false, err
	}
	return b.isTrue(), nil
}

func (d *Document) Time(paths ...string) (time.Time, error) {
	v, err := d.eval(paths...)
	if err != nil {
		return time.Time{}, err
	}
	t, err := v.toMoment()
	if err != nil {
		return time.Time{}, err
	}
	_ = t
	return time.Now(), nil
}

func (d *Document) Text(paths ...string) (string, error) {
	v, err := d.eval(paths...)
	if err != nil {
		return "", err
	}
	t, err := v.toText()
	if err != nil {
		return "", err
	}
	return toText(t)
}

func (d *Document) Document(paths ...string) (*Document, error) {
	var n = d.root
	for i := 0; i < len(paths); i++ {
		obj, ok := n.nodes[paths[i]]
		if !ok {
			return nil, fmt.Errorf("%s: object not found", paths[i])
		}
		n, ok = obj.(*Object)
		if !ok {
			return nil, fmt.Errorf("%s: not an object", paths[i])
		}
	}
	doc := Document{
		root: n,
	}
	return &doc, nil
}

func (d *Document) eval(paths ...string) (Value, error) {
	e, env, err := d.find(paths...)
	if err != nil {
		return nil, err
	}
	return e.Eval(env)
}

func (d *Document) find(paths ...string) (Expr, Environment, error) {
	var (
		z    = len(paths) - 1
		n    = d.root
		o    string
		list []*Object
	)
	if z < 0 {
		return nil, nil, fmt.Errorf("empty path!")
	}
	o = paths[z]
	paths = paths[:z]
	list = append(list, n)
	for i := 0; i < z; i++ {
		obj, ok := n.nodes[paths[i]]
		if !ok {
			return nil, nil, fmt.Errorf("%s: object not found", paths[i])
		}
		n, ok = obj.(*Object)
		if !ok {
			return nil, nil, fmt.Errorf("%s: not an object", paths[i])
		}
		list = append(list, n)
	}
	node, ok := n.nodes[o]
	if !ok {
		return nil, nil, fmt.Errorf("%s: option not found", o)
	}
	opt, ok := node.(Option)
	if !ok {
		return nil, nil, fmt.Errorf("%s: not an option", o)
	}
	return opt.expr, createEnv(list), nil
}

func undefinedVariable(str string) error {
	return fmt.Errorf("%s: %w variable", str, ErrUndefined)
}
