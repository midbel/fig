package fig

import (
	"fmt"
	"strconv"
	"strings"
	"time"
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
	var op string
	switch b.op {
	case Add:
		op = "add"
	case Sub:
		op = "sub"
	case Mul:
		op = "mul"
	case Div:
		op = "div"
	case Mod:
		op = "mod"
	case Pow:
		op = "pow"
	case And:
		op = "and"
	case Or:
		op = "or"
	case Gt:
		op = "gt"
	case Lt:
		op = "lt"
	case Ge:
		op = "ge"
	case Le:
		op = "le"
	case Equal:
		op = "eq"
	case NotEqual:
		op = "ne"
	default:
		op = "other"
	}
	return fmt.Sprintf("binary(%s, left: %s, right: %s)", op, b.left, b.right)
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

type Ternary struct {
	cond Expr
	csq  Expr
	alt  Expr
}

func (t Ternary) Eval(e Environment) (Value, error) {
	v, err := t.cond.Eval(e)
	if err != nil {
		return nil, err
	}
	if v.isTrue() {
		return t.csq.Eval(e)
	}
	return t.alt.Eval(e)
}

func (t Ternary) String() string {
	return fmt.Sprintf("ternary(cdt: %s, csq: %s, alt: %s)", t.cond, t.csq, t.alt)
}

var timePattern = []string{
	"2006-01-02T15:04:05",
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05.000Z",
	"2006-01-02T15:04:05.000000Z",
	"2006-01-02T15:04:05.000000000Z",
	"2006-01-02T15:04:05-07:00",
	"2006-01-02T15:04:05.000-07:00",
	"2006-01-02T15:04:05.000000-07:00",
	"2006-01-02T15:04:05.000000000-07:00",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04:05Z",
	"2006-01-02 15:04:05.000Z",
	"2006-01-02 15:04:05.000000Z",
	"2006-01-02 15:04:05.000000000Z",
	"2006-01-02 15:04:05-07:00",
	"2006-01-02 15:04:05.000-07:00",
	"2006-01-02 15:04:05.000000-07:00",
	"2006-01-02 15:04:05.000000000-07:00",
	"2006-01-02",
	"15:04:05",
	"15:04:05.000",
	"15:04:05.000000",
	"15:04:05.000000000",
}

type Literal struct {
	tok Token
	mul float64
}

func makeLiteral(tok Token) Literal {
	return Literal{tok: tok}
}

func (i Literal) String() string {
	return fmt.Sprintf("literal(%s)", i.tok.Input)
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
	case Date, DateTime, Time:
		var when time.Time
		for _, pattern := range timePattern {
			when, err = time.Parse(pattern, i.tok.Input)
			if err == nil {
				break
			}
		}
		if err == nil {
			val = makeMoment(when)
		}
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
	var prefix string
	if v.tok.Type == LocalVar {
		prefix = "local"
	} else {
		prefix = "env"
	}
	return fmt.Sprintf("%s(%s)", prefix, v.tok.Input)
}

func (v Variable) Eval(e Environment) (Value, error) {
	if v.tok.Type == LocalVar {
		return e.resolveLocal(v.tok.Input)
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

func (f Func) Eval(e Environment) (Value, error) {
	call, ok := builtins[f.name.Input]
	if !ok {
		return nil, undefinedFunction(f.name.Input)
	}
	args := make([]Value, len(f.args))
	for i := range f.args {
		a, err := f.args[i].Eval(e)
		if err != nil {
			return nil, err
		}
		args[i] = a
	}
	return call(args...)
}

type value struct {
	inner Value
}

func (v value) Eval(_ Environment) (Value, error) {
	return v.inner, nil
}

func (v value) String() string {
	return fmt.Sprintf("value(%s)", v.inner)
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

func (a Array) Eval(e Environment) (Value, error) {
	var (
		vs  = make([]Value, len(a.expr))
		err error
	)
	for i := range a.expr {
		vs[i], err = a.expr[i].Eval(e)
		if err != nil {
			break
		}
	}
	var v Value
	if err == nil {
		v = Slice{inner: vs}
	}
	return v, err
}

type Index struct {
	arr Expr
	ptr Expr
}

func (i Index) String() string {
	return fmt.Sprintf("index(arr: %s, index: %s)", i.arr, i.ptr)
}

func (i Index) Eval(e Environment) (Value, error) {
	arr, err := i.arr.Eval(e)
	if err != nil {
		return nil, err
	}
	ptr, err := i.ptr.Eval(e)
	if err != nil {
		return nil, err
	}
	return arr.at(ptr)
}
