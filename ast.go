package fig

import (
	"fmt"
	"strconv"
	"strings"
)

type Expr interface {
	Eval(Environment) (Value, error)
	fmt.Stringer
}

type Unary struct {
	right Expr
	op    rune
}

func (u Unary) String() string {
	return fmt.Sprintf("unary(%s)", u.right)
}

func (u Unary) Eval(e Environment) (Value, error) {
	right, err := u.right.Eval(e)
	if err != nil {
		return nil, err
	}
	switch u.op {
	case Not:
		right, err = right.not()
	case Bnot:
		right, err = right.binnot()
	case Sub:
		right, err = right.reverse()
	case Add:
	default:
		err = ErrUnsupported
	}
	return right, err
}

type Binary struct {
	left  Expr
	right Expr
	op    rune
}

func (b Binary) String() string {
	return fmt.Sprintf("binary(left: %s, right: %s)", b.left, b.right)
}

func (b Binary) Eval(e Environment) (Value, error) {
	left, err := b.left.Eval(e)
	if err != nil {
		return nil, err
	}
	right, err := b.right.Eval(e)
	if err != nil {
		return nil, err
	}
	switch b.op {
	case Add:
		left, err = left.add(right)
	case Sub:
		left, err = left.subtract(right)
	case Div:
		left, err = left.divide(right)
	case Mul:
		left, err = left.multiply(right)
	case Mod:
		left, err = left.modulo(right)
	case Pow:
		left, err = left.power(right)
	case And:
		left, err = left.and(right)
	case Or:
		left, err = left.or(right)
	case Gt:
		var cmp int
		cmp, err = left.compare(right)
		if err == nil {
			left = makeBool(cmp > 0)
		}
	case Ge:
		var cmp int
		cmp, err = left.compare(right)
		if err == nil {
			left = makeBool(cmp >= 0)
		}
	case Lt:
		var cmp int
		cmp, err = left.compare(right)
		if err == nil {
			left = makeBool(cmp < 0)
		}
	case Le:
		var cmp int
		cmp, err = left.compare(right)
		if err == nil {
			left = makeBool(cmp <= 0)
		}
	case Equal:
		var cmp int
		cmp, err = left.compare(right)
		if err == nil {
			left = makeBool(cmp == 0)
		}
	case NotEqual:
		var cmp int
		cmp, err = left.compare(right)
		if err == nil {
			left = makeBool(cmp != 0)
		}
	case Lshift:
		left, err = left.leftshift(right)
	case Rshift:
		left, err = left.rightshift(right)
	case Band:
		left, err = left.binand(right)
	case Bor:
		left, err = left.binor(right)
	case Bnot:
		left, err = left.binxor(right)
	default:
		err = ErrUnsupported
	}
	return left, err
}

type Literal struct {
	tok Token
	mul float64
}

func makeLiteral(tok Token) Literal {
	return Literal{tok: tok}
}

func (i Literal) String() string {
	return fmt.Sprintf("literal(%s, multiplier: %.2f)", i.tok.Input, i.mul)
}

func (i Literal) Eval(_ Environment) (Value, error) {
	var (
		val Value
		err error
	)
	switch i.tok.Type {
	case Integer:
		var n int64
		if n, err = strconv.ParseInt(i.tok.Input, 0, 64); err == nil {
			val = makeInt(n)
		}
	case Float:
		var n float64
		if n, err = strconv.ParseFloat(i.tok.Input, 64); err == nil {
			val = makeDouble(n)
		}
	case String, Ident:
		val = makeText(i.tok.Input)
	case Date, DateTime:
		val = Moment{}
	case Boolean:
		switch i.tok.Input {
		case kwTrue, kwYes, kwOn:
			val = makeBool(true)
		case kwFalse, kwNo, kwOff:
			val = makeBool(false)
		default:
			err = fmt.Errorf("%s: invalid boolean value", i.tok.Input)
		}
	default:
		err = ErrUnsupported
	}
	if i.mul != 0 {
		val, err = val.multiply(makeDouble(i.mul))
	}
	return val, err
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

func (v Variable) Eval(e Environment) (Value, error) {
	if v.tok.Type == EnvVar {
		e, ok := e.(*env)
		if !ok {
			return nil, undefinedVariable(v.tok.Input)
		}
		return e.parent.Resolve(v.tok.Input)
	}
	return e.Resolve(v.tok.Input)
}

type Func struct {
	name Token
	args []Expr
}

func (f Func) String() string {
	return fmt.Sprintf("function(%s)", f.name.Input)
}

func (f Func) Eval(_ Environment) (Value, error) {
	return nil, nil
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

func (a Array) Eval(_ Environment) (Value, error) {
	return nil, nil
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

func (o *Object) IsRoot() bool {
	return o.name.isZero()
}

func (o *Object) IsEmpty() bool {
	return len(o.nodes) == 0
}

func (o *Object) get(tok Token) (*Object, error) {
	n, ok := o.nodes[tok.Input]
	if !ok {
		return nil, fmt.Errorf("%s: object not found", tok.Input)
	}
	obj, ok := n.(*Object)
	if !ok {
		return nil, fmt.Errorf("%s should be an object", tok.Input)
	}
	return obj, nil
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
