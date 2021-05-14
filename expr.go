package fig

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	errReturn   = errors.New(kwReturn)
	errBreak    = errors.New(kwBreak)
	errContinue = errors.New(kwContinue)
)

type Expr interface {
	Eval(Environment) (Value, error)
	fmt.Stringer
}

type ForeachLoop struct {
	ident Token
	expr  Expr
	body  Expr
	alt   Expr
}

func (f ForeachLoop) String() string {
	return fmt.Sprintf("foreach(%s)", f.expr)
}

func (f ForeachLoop) Eval(e Environment) (Value, error) {
	return nil, nil
}

type ForLoop struct {
	init Expr
	next Expr
	cdt  Expr
	body Expr
	alt  Expr
}

func (f ForLoop) String() string {
	return fmt.Sprintf("for(%s)", f.cdt)
}

func (f ForLoop) Eval(e Environment) (Value, error) {
	_, err := f.init.Eval(e)
	if err != nil {
		return nil, err
	}
	var i int
	for {
		v, err := f.cdt.Eval(e)
		if err != nil {
			return nil, err
		}
		if !v.isTrue() {
			break
		}
		i++
		if v, err = f.body.Eval(e); err != nil {
			if errors.Is(err, errReturn) {
				return v, err
			} else if errors.Is(err, errBreak) {
				break
			} else if errors.Is(err, errContinue) {
				// do nothing
			} else {
				return nil, err
			}
		}
		if _, err = f.next.Eval(e); err != nil {
			return nil, err
		}
	}
	if i == 0 {
		return f.alt.Eval(e)
	}
	return nil, nil
}

type WhileLoop struct {
	cdt Expr
	csq Expr
	alt Expr
}

func (w WhileLoop) String() string {
	return fmt.Sprintf("while(%s)", w.cdt)
}

func (w WhileLoop) Eval(e Environment) (Value, error) {
	var i int
	ec := EnclosedEnv(e)
	for {
		v, err := w.cdt.Eval(ec)
		if err != nil {
			return nil, err
		}
		i++
		if !v.isTrue() {
			break
		}
		if v, err = w.csq.Eval(ec); err != nil {
			if errors.Is(err, errReturn) {
				return v, err
			} else if errors.Is(err, errBreak) {
				break
			} else if errors.Is(err, errContinue) {
				continue
			}
			return nil, err
		}
	}
	if i == 0 {
		return w.alt.Eval(e)
	}
	return nil, nil
}

type BreakLoop struct{}

func (_ BreakLoop) String() string {
	return "break"
}

func (_ BreakLoop) Eval(_ Environment) (Value, error) {
	return nil, errBreak
}

type ContinueLoop struct{}

func (_ ContinueLoop) String() string {
	return "continue"
}

func (_ ContinueLoop) Eval(_ Environment) (Value, error) {
	return nil, errContinue
}

type Block struct {
	expr []Expr
}

func (b Block) Eval(e Environment) (Value, error) {
	ec := EnclosedEnv(e)
	for _, ex := range b.expr {
		v, err := ex.Eval(ec)
		if errors.Is(err, errReturn) {
			return v, err
		}
		if err != nil {
			return v, err
		}
	}
	return nil, nil
}

func (b Block) String() string {
	return "block()"
}

type Assignment struct {
	ident Token
	expr  Expr
	let   bool
}

func (a Assignment) Eval(e Environment) (Value, error) {
	v, err := a.expr.Eval(e)
	if err != nil {
		return nil, err
	}
	if a.let {
		e.Define(a.ident.Input, v)
		return nil, nil
	}
	return nil, e.assign(a.ident.Input, v)
}

func (a Assignment) String() string {
	return fmt.Sprintf("assign(%s, expr: %s)", a.ident.Input, a.expr)
}

type Return struct {
	expr Expr
}

func (r Return) Eval(e Environment) (Value, error) {
	value, err := r.expr.Eval(e)
	if err == nil {
		err = errReturn
	}
	return value, err
}

func (r Return) String() string {
	return fmt.Sprintf("return(%s)", r.expr)
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
	cdt Expr
	csq Expr
	alt Expr
}

func (t Ternary) Eval(e Environment) (Value, error) {
	v, err := t.cdt.Eval(e)
	if err != nil {
		return nil, err
	}
	if v.isTrue() {
		v, err := t.csq.Eval(e)
		return v, err
	}
	if t.alt != nil {
		return t.alt.Eval(e)
	}
	return nil, nil
}

func (t Ternary) String() string {
	return fmt.Sprintf("ternary(%s, csq: %s, alt: %s)", t.cdt, t.csq, t.alt)
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

type Call struct {
	name Token
	args []Expr
}

func (c Call) String() string {
	return fmt.Sprintf("call(%s)", c.name.Input)
}

func (c Call) Eval(e Environment) (Value, error) {
	args, err := c.arguments(e)
	if err != nil {
		return nil, err
	}
	v, err := c.executeUserFunc(e, args)
	if err != nil && errors.Is(err, ErrUndefined) {
		return c.executeBuiltin(e, args)
	}
	return v, err
}

func (c Call) executeUserFunc(e Environment, args []Value) (Value, error) {
	fn, err := e.resolveFunc(c.name.Input)
	if err != nil {
		return nil, err
	}
	if len(fn.args) != len(args) {
		return nil, invalidArgument(fn.name.Input)
	}
	ee := EnclosedEnv(e)
	for i, a := range fn.args {
		ee.Define(a.Input, args[i])
	}
	return fn.Eval(ee)
}

func (c Call) executeBuiltin(e Environment, args []Value) (Value, error) {
	call, ok := builtins[c.name.Input]
	if !ok {
		return nil, undefinedFunction(c.name.Input)
	}
	return call(args...)
}

func (c Call) arguments(e Environment) ([]Value, error) {
	args := make([]Value, len(c.args))
	for i := range c.args {
		a, err := c.args[i].Eval(e)
		if err != nil {
			return nil, err
		}
		args[i] = a
	}
	return args, nil
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
		v = makeSlice(vs)
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
