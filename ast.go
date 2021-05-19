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

func (a Argument) String() string {
	if a.expr == nil {
		return fmt.Sprintf("arg(%s, pos: %d)", a.name.Input, a.pos)
	}
	return fmt.Sprintf("arg(%s, pos: %d, expr: %s)", a.name.Input, a.pos, a.expr)
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

func (f Func) String() string {
	args := make([]string, len(f.args))
	for i := range f.args {
		args[i] = f.args[i].String()
	}
	return fmt.Sprintf("func(%s, args: %s)", f.name.Input, strings.Join(args, ", "))
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

type Object struct {
	name     Token
	priority int64
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

func (o *Object) Decode(v interface{}) error {
	return nil
}

func (o *Object) MarshalJSON() ([]byte, error) {
	// TODO
	return nil, nil
}

func (o *Object) String() string {
	return fmt.Sprintf("object(%s)", o.name.Input)
}

func (o *Object) merge(node Node) error {
	switch n := node.(type) {
	case *Object:
		o.mergeObject(n)
	case List:
	case Option:
		return o.register(n)
	default:
		return fmt.Errorf("unexpected node type")
	}
	return nil
}

func (o *Object) mergeObject(other *Object) {
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
		return nil, fmt.Errorf("%s: can not be inserted", tok)
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
			return fmt.Errorf("%s: try to add option to object array is %w", opt, ErrAllow)
		}
		x.nodes = append(x.nodes, opt)
		o.nodes[opt.name.Input] = x
	default:
		return fmt.Errorf("%s: can not be inserted", opt)
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
		return fmt.Errorf("%s: try to replace option by function", fn.name.Input)
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

func (o *Object) getObject(str string) (*Object, error) {
	n, ok := o.nodes[str]
	if !ok {
		return nil, fmt.Errorf("%s: %w object", str, ErrUndefined)
	}
	obj, ok := n.(*Object)
	if !ok {
		return nil, fmt.Errorf("%s: not an object (%T)", str, n)
	}
	return obj, nil
}

func (o *Object) getOption(str string) (Option, error) {
	node, ok := o.nodes[str]
	if !ok {
		return Option{}, fmt.Errorf("%s: %w option", str, ErrUndefined)
	}
	opt, ok := node.(Option)
	if !ok {
		return Option{}, fmt.Errorf("%s: not an option", str)
	}
	return opt, nil
}

func (o *Object) copy() *Object {
	list := make(map[string]Node)
	for k, n := range o.nodes {
		switch n.(type) {
		case Option, Func:
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

func (i List) String() string {
	return fmt.Sprintf("list(%s)", i.name)
}

func (i List) MarshalJSON() ([]byte, error) {
	// TODO
	return nil, nil
}

type Option struct {
	name Token
	expr Expr

	Note
}

func (o Option) String() string {
	return fmt.Sprintf("option(%s, %s)", o.name.Input, o.expr)
}

func (o Option) MarshalJSON() ([]byte, error) {
	return nil, nil
}
