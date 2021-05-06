package fig

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
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

type Document struct {
	root *Object
	env  Environment
}

func ParseDocument(r io.Reader) (*Document, error) {
	return ParseDocumentWithEnv(r, EmptyEnv())
}

func ParseDocumentWithEnv(r io.Reader, env Environment) (*Document, error) {
	root, err := Parse(r)
	if err != nil {
		return nil, err
	}
	doc := Document{
		root: root,
		env:  env,
	}
	return &doc, nil
}

func (d *Document) SetInt(str string, i int64) {
	if e, ok := d.env.(*Env); ok {
		e.Define(str, makeInt(i))
	}
}

func (d *Document) SetBool(str string, b bool) {
	if e, ok := d.env.(*Env); ok {
		e.Define(str, makeBool(b))
	}
}

func (d *Document) SetDouble(str string, f float64) {
	if e, ok := d.env.(*Env); ok {
		e.Define(str, makeDouble(f))
	}
}

func (d *Document) SetText(str string, t string) {
	if e, ok := d.env.(*Env); ok {
		e.Define(str, makeText(str))
	}
}

func (d *Document) SetTime(str string, t time.Time) {
	if e, ok := d.env.(*Env); ok {
		e.Define(str, makeMoment(t))
	}
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

func (d *Document) Decode(v interface{}) error {
	return d.DecodeWithEnv(v, d.env)
}

func (d *Document) DecodeWithEnv(v interface{}, env Environment) error {
	return nil
}

func (d *Document) eval(paths ...string) (Value, error) {
	e, env, err := d.find(paths...)
	if err != nil {
		return nil, err
	}
	return e.Eval(env)
}

func (d *Document) find(paths ...string) (Expr, Environment, error) {
	if len(paths) == 0 {
		return nil, nil, fmt.Errorf("empty path!")
	}
	var (
		curr = d.root
		err  error
		list []*Object
	)
	list = append(list, curr.copy())
	for i := 0; i < len(paths)-1; i++ {
		curr, err = curr.getObject(paths[i])
		if err != nil {
			return nil, nil, err
		}
		list = append(list, curr.copy())
	}
	opt, err := curr.getOption(paths[len(paths)-1])
	if err != nil {
		return nil, nil, err
	}
	list[len(list)-1].unregister(opt.name.Input)
	return opt.expr, createEnv(reverseList(list), d.env), nil
}

func reverseList(list []*Object) []*Object {
	size := len(list)
	for i, j := 0, size-1; i < size/2; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
	return list
}

func undefinedVariable(str string) error {
	return fmt.Errorf("%s: %w variable", str, ErrUndefined)
}

func undefinedFunction(str string) error {
	return fmt.Errorf("%s: %w function", str, ErrUndefined)
}
