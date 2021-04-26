package fig

import (
	"fmt"
	"io"
	"time"
)

type Env struct {
}

type Document struct {
	root *Object
	env  Env
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
	return d.find(paths...)
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
	e, err := d.find(paths...)
	if err != nil {
		return nil, err
	}
	return e.Eval()
}

func (d *Document) find(paths ...string) (Expr, error) {
	var (
		z = len(paths) - 1
		n = d.root
		o string
	)
	if z < 0 {
		return nil, fmt.Errorf("empty path!")
	}
	o = paths[z]
	paths = paths[:z]
	for i := 0; i < z; i++ {
		obj, ok := n.nodes[paths[i]]
		if !ok {
			return nil, fmt.Errorf("%s: object not found", paths[i])
		}
		n, ok = obj.(*Object)
		if !ok {
			return nil, fmt.Errorf("%s: not an object", paths[i])
		}
	}
	node, ok := n.nodes[o]
	if !ok {
		return nil, fmt.Errorf("%s: option not found", o)
	}
	opt, ok := node.(Option)
	if !ok {
		return nil, fmt.Errorf("%s: not an option", o)
	}
	return opt.expr, nil
}
