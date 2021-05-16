package fig

import (
	"fmt"
	"io"
	"time"
)

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

func (d *Document) DefineInt(str string, i int64) {
	if e, ok := d.env.(*Env); ok {
		e.Define(str, makeInt(i))
	}
}

func (d *Document) DefineBool(str string, b bool) {
	if e, ok := d.env.(*Env); ok {
		e.Define(str, makeBool(b))
	}
}

func (d *Document) DefineDouble(str string, f float64) {
	if e, ok := d.env.(*Env); ok {
		e.Define(str, makeDouble(f))
	}
}

func (d *Document) DefineText(str string, t string) {
	if e, ok := d.env.(*Env); ok {
		e.Define(str, makeText(str))
	}
}

func (d *Document) DefineTime(str string, t time.Time) {
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
	return toTime(t)
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

func (d *Document) Value(paths ...string) (interface{}, error) {
	v, err := d.eval(paths...)
	if err != nil {
		return nil, err
	}
	return toInterface(v), nil
}

func (d *Document) Slice(paths ...string) ([]interface{}, error) {
	_, err := d.eval(paths...)
	if err != nil {
		return nil, err
	}
	return nil, nil
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
