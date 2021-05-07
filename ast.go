package fig

import (
	"fmt"
)

type Node interface{}

type Note struct {
	pre  []string
	post string
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

func (o *Object) String() string {
	return fmt.Sprintf("object(%s)", o.name.Input)
}

func (o *Object) IsRoot() bool {
	return o.name.isZero()
}

func (o *Object) IsEmpty() bool {
	return len(o.nodes) == 0
}

func (o *Object) get(tok Token) (*Object, error) {
	n, ok := o.nodes[tok.Input]
	if !ok {
		obj := createObjectWithToken(tok)
		o.nodes[tok.Input] = obj
		return obj, nil
		// return nil, fmt.Errorf("%s: object not found", tok.Input)
	}
	obj, ok := n.(*Object)
	if !ok {
		return nil, fmt.Errorf("%s should be an object", tok.Input)
	}
	return obj, nil
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

func (o *Object) replace(opt Option, expr Value) error {
	opt.expr = value{expr}
	o.nodes[opt.name.Input] = opt
	return nil
}

func (o *Object) getObject(str string) (*Object, error) {
	n, ok := o.nodes[str]
	if !ok {
		return nil, fmt.Errorf("%s: %w object", str, ErrUndefined)
	}
	obj, ok := n.(*Object)
	if !ok {
		return nil, fmt.Errorf("%s: not an object", str)
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
		if opt, ok := n.(Option); ok {
			list[k] = opt
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

type Option struct {
	name Token
	expr Expr

	Note
}

func (o Option) String() string {
	return fmt.Sprintf("option(%s, %s)", o.name.Input, o.expr)
}
