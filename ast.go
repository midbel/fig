package fig

import (
	"errors"
	"fmt"
	"strings"
)

type Node interface{}

type Note struct {
	pre  []string
	post string
}

type Argument struct {
	name     Token
	expr     Expr
	pos      int
	variadic bool
}

func (a Argument) isPositional() bool {
	return a.expr == nil || a.name.isZero()
}

func replaceArg(a Argument, args []Argument) error {
	for i := range args {
		if a.name.Input == args[i].name.Input {
			args[i].expr = a.expr
			return nil
		}
	}
	return fmt.Errorf("%s: not found", a.name.Input)
}

type Func struct {
	name Token
	args []Argument
	body Expr
}

func (f Func) Eval(e Environment) (Value, error) {
	v, err := f.body.Eval(e)
	if errors.Is(err, errReturn) {
		err = nil
	}
	return v, err
}

func (f Func) copyArgs() []Argument {
	as := make([]Argument, len(f.args))
	copy(as, f.args)
	return as
}

const (
	InsAppend  = "append"
	InsMerge   = "merge"
	InsRewrite = "rewrite"
)

type Object struct {
	name     Token
	priority int64
	insmode  string
	nodes    map[string]Node

	Note
}

func createObject() *Object {
	var tok Token
	return createObjectWithToken(tok)
}

func createObjectWithToken(tok Token) *Object {
	return &Object{
		name:  tok,
		nodes: make(map[string]Node),
	}
}

func (o *Object) merge(node Node) error {
	switch n := node.(type) {
	case *Object:
		switch strings.ToLower(n.insmode) {
		case InsAppend:
			return o.appendObject(n)
		case InsMerge:
			return o.mergeObject(n)
		case InsRewrite, "":
			o.rewriteObject(n)
		default:
			return fmt.Errorf("%s: unknown insert mode", n.insert)
		}
	case List:
	case Option:
		return o.register(n)
	default:
		return fmt.Errorf("unexpected node type: %T", n)
	}
	return nil
}

func (o *Object) mergeObject(other *Object) error {
	var err error
	for k, n := range other.nodes {
		if _, ok := o.nodes[k]; !ok {
			o.nodes[k] = n
			continue
		}
		switch n := n.(type) {
		case List:
		case Option:
			err = o.register(n)
		case *Object:
			err = o.appendObject(n)
		default:
		}
		if err != nil {
			break
		}
	}
	return err
}

func (o *Object) appendObject(other *Object) error {
	n, ok := o.nodes[other.name.Input]
	if !ok {
		o.nodes[other.name.Input] = other
		return nil
	}
	switch x := n.(type) {
	case *Object:
		i := List{
			name:  x.name,
			nodes: []Node{n, other},
		}
		o.nodes[x.name.Input] = i
	case List:
		x.nodes = append(x.nodes, other)
	default:
		return fmt.Errorf("unexpected node type: %T", n)
	}
	return nil
}

func (o *Object) rewriteObject(other *Object) {
	if other.name.IsIdent() {
		o.nodes[other.name.Input] = other
		return
	}
	for k, v := range other.nodes {
		o.nodes[k] = v
	}
}

func (o *Object) get(tok Token) (*Object, error) {
	n, ok := o.nodes[tok.Input]
	if !ok {
		obj := createObjectWithToken(tok)
		o.nodes[tok.Input] = obj
		return obj, nil
	}
	switch n := n.(type) {
	case *Object:
		return n, nil
	case List:
		x := n.nodes[len(n.nodes)-1]
		if obj, ok := x.(*Object); ok {
			return obj, nil
		}
	default:
	}
	return nil, fmt.Errorf("%s: not an object (%T)", tok.Input, n)
}

func (o *Object) insert(tok Token) (*Object, error) {
	n, ok := o.nodes[tok.Input]
	if !ok {
		obj := createObjectWithToken(tok)
		o.nodes[tok.Input] = obj
		return obj, nil
	}
	var obj *Object
	switch x := n.(type) {
	case *Object:
		obj = createObjectWithToken(tok)
		i := List{
			name:  tok,
			nodes: []Node{n, obj},
		}
		o.nodes[tok.Input] = i
	case List:
		if _, ok := x.nodes[0].(*Object); !ok {
			return nil, fmt.Errorf("%s: try to add object to option array is %w", tok, ErrAllow)
		}
		obj = createObjectWithToken(tok)
		x.nodes = append(x.nodes, obj)
		o.nodes[tok.Input] = x
	default:
		return nil, fmt.Errorf("%w: %s can not be inserted", ErrAllow, tok)
	}
	return obj, nil
}

func (o *Object) register(opt Option) error {
	n, ok := o.nodes[opt.name.Input]
	if !ok {
		o.nodes[opt.name.Input] = opt
		return nil
	}
	switch x := n.(type) {
	case Option:
		i := List{
			name:  opt.name,
			nodes: []Node{x, opt},
		}
		o.nodes[opt.name.Input] = i
	case List:
		if _, ok := x.nodes[0].(Option); !ok {
			return fmt.Errorf("%s: try to add option to object array is %w", opt.name.Input, ErrAllow)
		}
		x.nodes = append(x.nodes, opt)
		o.nodes[opt.name.Input] = x
	default:
		return fmt.Errorf("%w: %s can not be inserted", ErrAllow, opt.name.Input)
	}
	return nil
}

func (o *Object) registerFunc(fn Func) error {
	n, ok := o.nodes[fn.name.Input]
	if !ok {
		o.nodes[fn.name.Input] = fn
		return nil
	}
	if _, ok := n.(Func); !ok {
		return fmt.Errorf("%w: try to replace option by function %s", ErrAllow, fn.name.Input)
	}
	o.nodes[fn.name.Input] = fn
	return nil
}

func (o *Object) replace(opt Option, expr Value) error {
	opt.expr = value{inner: expr}
	o.nodes[opt.name.Input] = opt
	return nil
}

func (o *Object) getFunction(str string) (Func, error) {
	n, ok := o.nodes[str]
	if !ok {
		return Func{}, fmt.Errorf("%s: %w function", str, ErrUndefined)
	}
	fn, ok := n.(Func)
	if !ok {
		return Func{}, fmt.Errorf("%s: not a function (%T)", str, n)
	}
	return fn, nil
}

func (o *Object) getNode(str string) (Node, error) {
	n, ok := o.nodes[str]
	if !ok {
		return nil, fmt.Errorf("%s: %w object", str, ErrUndefined)
	}
	return n, nil
}

func (o *Object) getOption(str string) (Option, error) {
	node, ok := o.nodes[str]
	if !ok {
		return Option{}, fmt.Errorf("%s: %w option", str, ErrUndefined)
	}
	switch n := node.(type) {
	case Option:
		return n, nil
	case List:
		return n.asOption()
	default:
		return Option{}, fmt.Errorf("%s: not an option", str)
	}
}

func (o *Object) copy() *Object {
	list := make(map[string]Node)
	for k, n := range o.nodes {
		switch n.(type) {
		case Option, Func, List:
			list[k] = n
		default:
		}
	}
	obj := Object{
		name:  o.name,
		nodes: list,
	}
	return &obj
}

func (o *Object) unregister(opt string) {
	delete(o.nodes, opt)
}

type List struct {
	name  Token
	nodes []Node
}

func (i List) asOption() (Option, error) {
	var (
		es  []Expr
		opt Option
	)
	opt.name = i.name
	for _, n := range i.nodes {
		o, ok := n.(Option)
		if !ok {
			return o, fmt.Errorf("%s: not an option", i.name.Input)
		}
		es = append(es, o.expr)
	}
	opt.expr = Array{expr: es}
	return opt, nil
}

type Option struct {
	name Token
	expr Expr

	Note
}
