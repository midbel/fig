package fig

import (
	"fmt"
	"strings"
)

type Expr interface {
	Eval() error
	fmt.Stringer
}

type Unary struct {
	right Expr
	op    rune
}

func (u Unary) String() string {
	return fmt.Sprintf("unary(%s)", u.right)
}

func (u Unary) Eval() error {
	return nil
}

type Binary struct {
	left  Expr
	right Expr
	op    rune
}

func (b Binary) String() string {
	return fmt.Sprintf("binary(left: %s, right: %s)", b.left, b.right)
}

func (b Binary) Eval() error {
	return nil
}

type Literal struct {
	tok Token
}

func makeLiteral(tok Token) Literal {
	return Literal{tok: tok}
}

func (i Literal) String() string {
	return fmt.Sprintf("literal(%s)", i.tok.Input)
}

func (i Literal) Eval() error {
	return nil
}

type Variable struct {
	tok Token
}

func makeVariable(tok Token) Variable {
	return Variable{tok: tok}
}

func (v Variable) String() string {
	return fmt.Sprintf("variable(%s)", v.tok.Input)
}

func (v Variable) Eval() error {
	return nil
}

type Array struct {
	expr []Expr
}

func (a Array) String() string {
	if len(a.expr) == 0 {
		return "array()"
	}
	var str []string
	for _, e := range a.expr {
		str = append(str, e.String())
	}
	return fmt.Sprintf("array(%s)", strings.Join(str, ", "))
}

func (a Array) Eval() error {
	return nil
}

type Node interface{}

type Note struct {
	pre  []string
	post string
}

type Object struct {
	name  Token
	nodes map[string]Node

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

func (o *Object) Has(str string) bool {
	return false
}

func (o *Object) Get(str string) (Expr, error) {
	return nil, nil
}

func (o *Object) IsRoot() bool {
	return o.name.isZero()
}

func (o *Object) IsEmpty() bool {
	return len(o.nodes) == 0
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
