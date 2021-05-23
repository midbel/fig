package fig

import (
	"errors"
	"fmt"
	// "sort"
	"strconv"
	"strings"
	"time"
)

var (
	errReturn   = errors.New(kwReturn)
	errBreak    = errors.New(kwBreak)
	errContinue = errors.New(kwContinue)
)

const (
	iecKilo = 1024
	iecMega = iecKilo * iecKilo
	iecGiga = iecMega * iecKilo
	iecTera = iecGiga * iecKilo

	siKilo = 1000
	siMega = siKilo * siKilo
	siGiga = siMega * siKilo
	siTera = siGiga * siKilo
)

const (
	millis     = 1 / 1000
	secPerMin  = 60
	secPerHour = secPerMin * 60
	secPerDay  = secPerHour * 24
)

var multipliers = map[string]float64{
	"ms":   millis,
	"s":    0,
	"sec":  0,
	"min":  secPerMin,
	"h":    secPerHour,
	"hour": secPerHour,
	"d":    secPerDay,
	"day":  secPerDay,
	"b":    0,
	"B":    0,
	"k":    iecKilo,
	"K":    siKilo,
	"m":    iecMega,
	"M":    siMega,
	"g":    iecGiga,
	"G":    siGiga,
	"t":    iecTera,
	"T":    siTera,
}

type Expr interface {
	Eval(Environment) (Value, error)
}

type ForeachLoop struct {
	ident Token
	loop  Token
	expr  Expr
	body  Expr
	alt   Expr

	Note
}

func (f ForeachLoop) Eval(e Environment) (Value, error) {
	v, err := f.expr.Eval(e)
	if err != nil {
		return nil, err
	}
	vs, ok := v.(Slice)
	if !ok {
		return nil, ErrUnsupported
	}
	var (
		loop int
		last Value
	)
	for loop, last = range vs.inner {
		env := EnclosedEnv(e)
		env.Define(f.ident.Input, last)
		if !f.loop.isZero() {
			env.Define(f.loop.Input, makeInt(int64(loop)))
		}

		if last, err = f.body.Eval(env); err != nil {
			if errors.Is(err, errReturn) {
				return last, err
			} else if errors.Is(err, errBreak) {
				break
			} else if errors.Is(err, errContinue) {
				continue
			}
			return nil, err
		}
	}
	if loop == 0 && f.alt != nil {
		return f.alt.Eval(e)
	}
	return last, nil
}

type ForLoop struct {
	init Expr
	next Expr
	cdt  Expr
	body Expr
	alt  Expr

	Note
}

func (f ForLoop) Eval(e Environment) (Value, error) {
	ee := EnclosedEnv(e)
	if f.init != nil {
		if _, err := f.init.Eval(ee); err != nil {
			return nil, err
		}
	}
	var (
		loop int
		last Value
		err  error
	)
	for {
		if f.cdt != nil {
			if last, err = f.cdt.Eval(ee); err != nil {
				return nil, err
			}
			if !last.isTrue() {
				break
			}
		}
		loop++
		if last, err = f.body.Eval(EnclosedEnv(ee)); err != nil {
			if errors.Is(err, errReturn) {
				return last, err
			} else if errors.Is(err, errBreak) {
				break
			} else if errors.Is(err, errContinue) {
				// do nothing
			} else {
				return nil, err
			}
		}
		if f.next != nil {
			if _, err = f.next.Eval(ee); err != nil {
				return nil, err
			}
		}
	}
	if loop == 0 {
		return f.alt.Eval(e)
	}
	return last, nil
}

type WhileLoop struct {
	cdt Expr
	csq Expr
	alt Expr

	Note
}

func (w WhileLoop) Eval(e Environment) (Value, error) {
	var (
		loop int
		err  error
		last Value
		env  = EnclosedEnv(e)
	)
	for {
		last, err = w.cdt.Eval(env)
		if err != nil {
			return nil, err
		}
		loop++
		if !last.isTrue() {
			break
		}
		if last, err = w.csq.Eval(env); err != nil {
			if errors.Is(err, errReturn) {
				return last, err
			} else if errors.Is(err, errBreak) {
				break
			} else if errors.Is(err, errContinue) {
				continue
			}
			return nil, err
		}
	}
	if loop == 0 && w.alt != nil {
		return w.alt.Eval(e)
	}
	return last, nil
}

type BreakLoop struct {
	Note
}

func (_ BreakLoop) Eval(_ Environment) (Value, error) {
	return nil, errBreak
}

type ContinueLoop struct {
	Note
}

func (_ ContinueLoop) Eval(_ Environment) (Value, error) {
	return nil, errContinue
}

type Block struct {
	expr []Expr
	Note
}

func (b Block) Eval(e Environment) (Value, error) {
	var (
		env  = EnclosedEnv(e)
		err  error
		last Value
	)
	for _, ex := range b.expr {
		last, err = ex.Eval(env)
		if err != nil {
			return last, err
		}
	}
	return last, nil
}

type Assignment struct {
	ident Token
	expr  Expr
	let   bool

	Note
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

type Return struct {
	expr Expr

	Note
}

func (r Return) Eval(e Environment) (Value, error) {
	value, err := r.expr.Eval(e)
	if err == nil {
		err = errReturn
	}
	return value, err
}

type Unary struct {
	right Expr
	op    rune

	Note
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

	Note
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

	Note
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

type Literal struct {
	tok Token
	mul float64

	Note
}

func makeLiteral(tok Token) Literal {
	return Literal{tok: tok}
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

type Identifier struct {
	tok Token

	Note
}

func makeIdentifier(tok Token) Identifier {
	return Identifier{tok: tok}
}

func (i Identifier) Eval(e Environment) (Value, error) {
	return e.Resolve(i.tok.Input)
}

type Variable struct {
	tok Token

	Note
}

func makeVariable(tok Token) Variable {
	return Variable{tok: tok}
}

func (v Variable) Eval(e Environment) (Value, error) {
	if v.tok.Type == LocalVar {
		return e.resolveLocal(v.tok.Input)
	}
	return e.Resolve(v.tok.Input)
}

type Call struct {
	name Token
	args []Argument

	Note
}

func (c Call) Eval(e Environment) (Value, error) {
	v, err := c.executeUserFunc(e)
	if err != nil && errors.Is(err, ErrUndefined) {
		return c.executeBuiltin(e)
	}
	return v, err
}

func (c Call) executeUserFunc(e Environment) (Value, error) {
	fn, err := e.resolveFunc(c.name.Input)
	if err != nil {
		return nil, err
	}
	ee, err := c.applyArguments(fn.copyArgs(), e)
	if err != nil {
		return nil, err
	}
	v, err := fn.Eval(ee)
	if err != nil {
		return nil, fmt.Errorf("error while executing %s: %s", fn.name.Input, err)
	}
	return v, err
}

func (c Call) executeBuiltin(e Environment) (Value, error) {
	fn, ok := builtins[c.name.Input]
	if !ok {
		return nil, undefinedFunction(c.name.Input)
	}
	ee, err := c.applyArguments(fn.copyArgs(), e)
	if err != nil {
		return nil, err
	}
	v, err := fn.Eval(ee)
	if err != nil {
		return nil, fmt.Errorf("error while executing %s: %s", fn.name, err)
	}
	return v, err
}

func (c Call) applyArguments(args []Argument, e Environment) (Environment, error) {
	for i := range c.args {
		if i >= len(args) {
			return nil, fmt.Errorf("%w (%d instead of %d)", invalidArgument(c.name.Input), len(c.args), len(args))
		}
		if c.args[i].isPositional() && c.args[i].pos == args[i].pos {
			if !args[i].variadic {
				args[i].expr = c.args[i].expr
				continue
			}
			var arr Array
			for j := i; j < len(c.args); j++ {
				if !c.args[j].isPositional() {
					return nil, fmt.Errorf("unexpected keyword argument")
				}
				arr.expr = append(arr.expr, c.args[j].expr)
			}
			args[i].expr = arr
			break
		}
		if err := replaceArg(c.args[i], args); err != nil {
			return nil, err
		}
	}
	ee := EnclosedEnv(e)
	for _, a := range args {
		if a.expr == nil {
			return nil, fmt.Errorf("%s: no value", a.name.Input)
		}
		v, err := a.expr.Eval(e)
		if err != nil {
			return nil, err
		}
		ee.Define(a.name.Input, v)
	}
	return ee, nil
}

type value struct {
	inner Value

	Note
}

func (v value) Eval(_ Environment) (Value, error) {
	return v.inner, nil
}

type Array struct {
	expr []Expr

	Note
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

	Note
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

type Template struct {
	expr []Expr
}

func (t Template) Eval(e Environment) (Value, error) {
	var str strings.Builder
	for _, ex := range t.expr {
		v, err := ex.Eval(e)
		if err != nil {
			return nil, err
		}
		t, err := v.toText()
		if err != nil {
			return nil, err
		}
		s, _ := toText(t)
		str.WriteString(s)
	}
	return makeText(str.String()), nil
}

func parseTemplate(str string) (Expr, error) {
	createLiteral := func(ws *strings.Builder, tpl *Template) {
		defer ws.Reset()
		if ws.Len() == 0 {
			return
		}
		i := makeLiteral(makeToken(ws.String(), String))
		tpl.expr = append(tpl.expr, i)
	}
	createVariable := func(ws *strings.Builder, tpl *Template, marker rune) error {
		defer ws.Reset()
		kind := EnvVar
		if marker == dollar {
			kind = LocalVar
		}
		i := makeVariable(makeToken(ws.String(), kind))
		tpl.expr = append(tpl.expr, i)
		return nil
	}

	var (
		rs  = strings.NewReader(str)
		ws  strings.Builder
		tpl Template
	)
	for rs.Len() > 0 {
		r, _, _ := rs.ReadRune()
		if isVariable(r) {
			if n, _, _ := rs.ReadRune(); n == r {
				ws.WriteRune(r)
				continue
			}
			rs.UnreadRune()
			createLiteral(&ws, &tpl)
			marker := r
			if r, _, _ := rs.ReadRune(); !isLetter(r) {
				return nil, ErrSyntax
			}
			rs.UnreadRune()
			for rs.Len() > 0 {
				r, _, _ := rs.ReadRune()
				if !isIdent(r) {
					if err := createVariable(&ws, &tpl, marker); err != nil {
						return nil, err
					}
					ws.WriteRune(r)
					break
				}
				ws.WriteRune(r)
			}
			continue
		}
		ws.WriteRune(r)
	}
	createLiteral(&ws, &tpl)
	return tpl, nil
}
