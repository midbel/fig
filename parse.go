package fig

import (
	"errors"
	"fmt"
	"io"
)

var (
	ErrUnexpected = errors.New("unexpected token")
	ErrSyntax     = errors.New("syntax error")
	ErrAllow      = errors.New("not allowed")
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

const (
	bindLowest = iota
	bindAssign
	bindCdt
	bindRel
	bindCmp
	bindBin
	bindShift
	bindAdd
	bindMul
	bindPow
	bindIndex
	bindCall
)

var powers = map[rune]int{
	Lshift:   bindShift,
	Rshift:   bindShift,
	Band:     bindBin,
	Bor:      bindBin,
	And:      bindRel,
	Or:       bindRel,
	Lt:       bindCmp,
	Le:       bindCmp,
	Gt:       bindCmp,
	Ge:       bindCmp,
	Equal:    bindCmp,
	NotEqual: bindCmp,
	Add:      bindAdd,
	Sub:      bindAdd,
	Mul:      bindMul,
	Div:      bindMul,
	Mod:      bindMul,
	Pow:      bindPow,
	BegGrp:   bindCall,
	BegArr:   bindIndex,
	Assign:   bindAssign,
	Question: bindCdt,
}

type Parser struct {
	scan *Scanner
	curr Token
	peek Token

	infix  map[rune]func(Expr) (Expr, error)
	prefix map[rune]func() (Expr, error)
	macros map[string]func(map[string]Expr) (Node, error)

	loop   int
	userfn int
}

func NewParser(r io.Reader) (*Parser, error) {
	sc, err := Scan(r)
	if err != nil {
		return nil, err
	}

	var p Parser
	p.scan = sc
	p.macros = map[string]func(map[string]Expr) (Node, error){
		"include": include,
	}
	p.prefix = map[rune]func() (Expr, error){
		Add:      p.parseUnary,
		Sub:      p.parseUnary,
		Not:      p.parseUnary,
		Bnot:     p.parseUnary,
		Ident:    p.parseLiteral,
		Integer:  p.parseLiteral,
		Float:    p.parseLiteral,
		String:   p.parseLiteral,
		Heredoc:  p.parseLiteral,
		Date:     p.parseLiteral,
		Time:     p.parseLiteral,
		DateTime: p.parseLiteral,
		Boolean:  p.parseLiteral,
		LocalVar: p.parseVariable,
		EnvVar:   p.parseVariable,
		BegGrp:   p.parseGroup,
		BegArr:   p.parseArray,
	}
	p.infix = map[rune]func(Expr) (Expr, error){
		Add:      p.parseInfix,
		Sub:      p.parseInfix,
		Div:      p.parseInfix,
		Mul:      p.parseInfix,
		Mod:      p.parseInfix,
		Pow:      p.parseInfix,
		And:      p.parseInfix,
		Or:       p.parseInfix,
		Gt:       p.parseInfix,
		Lt:       p.parseInfix,
		Ge:       p.parseInfix,
		Le:       p.parseInfix,
		Equal:    p.parseInfix,
		NotEqual: p.parseInfix,
		Lshift:   p.parseInfix,
		Rshift:   p.parseInfix,
		Band:     p.parseInfix,
		Bor:      p.parseInfix,
		Bnot:     p.parseInfix,
		BegGrp:   p.parseCall,
		BegArr:   p.parseIndex,
		Question: p.parseTernary,
		Assign:   p.parseAssignment,
	}
	p.next()
	p.next()

	return &p, nil
}

func Parse(r io.Reader) (*Object, error) {
	p, err := NewParser(r)
	if err != nil {
		return nil, err
	}
	return p.Parse()
}

func (p *Parser) Parse() (*Object, error) {
	for p.curr.Type == EOL {
		p.next()
	}
	obj := createObject()
	for !p.done() {
		if p.curr.Type == Macro {
			if err := p.parseMacro(obj); err != nil {
				return nil, err
			}
			continue
		}
		if err := p.parse(obj); err != nil {
			return nil, err
		}
	}
	return obj, nil
}

func (p *Parser) parse(obj *Object) error {
	var cs []string
	for p.curr.IsComment() {
		cs = append(cs, p.curr.Input)
		p.next()
	}
	if p.curr.Type == EndObj {
		// TODO: lines before end-obj should not be lost
		return nil
	}
	if p.curr.Type == Ident && p.peek.Type == BegGrp {
		fn, err := p.parseFunction()
		if err == nil {
			err = obj.registerFunc(fn)
		}
		return err
	}
	if p.curr.IsIdent() && p.peek.Type == Assign {
		var (
			opt Option
			err error
		)
		opt.name = p.curr
		opt.pre = append(opt.pre, cs...)
		p.next()
		if opt.expr, err = p.parseValue(); err != nil {
			return err
		}
		if p.curr.IsComment() {
			opt.post = p.curr.Input
			p.next()
		}
		return obj.register(opt)
	}
	if p.curr.Type == Macro {
		if err := p.parseMacro(obj); err != nil {
			return err
		}
		return nil
	}
	var err error
	for i := 0; !p.done() && p.curr.Type != BegObj; i++ {
		if !p.curr.IsIdent() {
			return p.unexpectedToken()
		}
		if p.peek.Type == BegObj {
			obj, err = obj.insert(p.curr)
		} else {
			obj, err = obj.get(p.curr)
		}
		if err != nil {
			return err
		}
		p.next()
	}
	if p.curr.Type != BegObj {
		return p.unexpectedToken()
	}
	return p.parseObject(obj)
}

func (p *Parser) parseObject(obj *Object) error {
	p.next()
	for !p.done() && p.curr.Type != EndObj {
		if err := p.parse(obj); err != nil {
			return err
		}
	}
	if p.curr.Type != EndObj {
		return p.unexpectedToken()
	}
	p.next()
	if p.curr.IsComment() {
		p.next()
	}
	return nil
}

func (p *Parser) parseValue() (Expr, error) {
	p.next()
	if p.curr.Type == EOL {
		return nil, p.syntaxError()
	}
	expr, err := p.parseExpr(bindLowest)
	if err != nil {
		return nil, fmt.Errorf("parsing expression: %w", err)
	}
	if p.curr.Type == EOL {
		p.next()
	}
	return expr, err
}

func (p *Parser) parseFunction() (Func, error) {
	p.enterFunction()
	defer p.leaveFunction()

	var (
		fn  Func
		err error
	)
	fn.name = p.curr
	p.next()
	if p.curr.Type != BegGrp {
		return fn, p.unexpectedToken()
	}
	p.next()
	for p.curr.Type != EndGrp {
		if p.curr.Type != Ident {
			return fn, p.unexpectedToken()
		}
		fn.args = append(fn.args, p.curr)
		p.next()
		switch p.curr.Type {
		case EndGrp:
		case Comma:
			p.next()
			if p.curr.Type == EOL {
				p.next()
			}
		case EOL:
			if p.peek.Type != EndGrp {
				return fn, p.unexpectedToken()
			}
			p.next()
		default:
			return fn, p.unexpectedToken()
		}
	}
	if p.curr.Type != EndGrp {
		return fn, p.unexpectedToken()
	}
	p.next()
	if fn.body, err = p.parseBody(); err != nil {
		return fn, err
	}
	return fn, nil
}

func (p *Parser) parseBody() (Expr, error) {
	if p.curr.Type != BegObj {
		return nil, p.unexpectedToken()
	}
	p.next()
	var b Block
	for p.curr.Type != EndObj {
		var (
			expr Expr
			err  error
		)
		switch p.curr.Type {
		case Let:
			expr, err = p.parseLet()
		case Ret:
			expr, err = p.parseReturn()
		case If:
			expr, err = p.parseIf()
		case For:
			expr, err = p.parseFor()
		case While:
			expr, err = p.parseWhile()
		case Foreach:
			expr, err = p.parseForeach()
		case Break:
			if !p.inLoop() {
				return nil, p.syntaxError()
			}
			expr = BreakLoop{}
			p.next()
			if p.curr.Type != EOL {
				return nil, p.unexpectedToken()
			}
			p.next()
		case Continue:
			if !p.inLoop() {
				return nil, p.syntaxError()
			}
			expr = ContinueLoop{}
			p.next()
			if p.curr.Type != EOL {
				return nil, p.unexpectedToken()
			}
			p.next()
		case Ident:
			expr, err = p.parseExpr(bindLowest)
			if p.curr.Type != EOL {
				return nil, p.unexpectedToken()
			}
			p.next()
		case EOL, Comment:
			p.next()
			continue
		default:
			return nil, p.unexpectedToken()
		}
		if err != nil {
			return nil, err
		}
		b.expr = append(b.expr, expr)
	}
	if p.curr.Type != EndObj {
		return nil, p.unexpectedToken()
	}
	p.next()
	return b, nil
}

func (p *Parser) parseIf() (Expr, error) {
	p.next()
	if p.curr.Type != BegGrp {
		return nil, p.unexpectedToken()
	}
	p.next()
	var (
		ter Ternary
		err error
	)
	if ter.cdt, err = p.parseExpr(bindLowest); err != nil {
		return nil, err
	}
	if p.curr.Type != EndGrp {
		return nil, p.unexpectedToken()
	}
	p.next()
	if ter.csq, err = p.parseBody(); err != nil {
		return nil, err
	}
	if p.curr.Type == Else {
		p.next()
		if p.curr.Type == If {
			ter.alt, err = p.parseIf()
		} else {
			ter.alt, err = p.parseBody()
		}
		if err != nil {
			return nil, err
		}
	} else {
		ter.alt = Block{}
	}
	return ter, nil
}

func (p *Parser) parseWhile() (Expr, error) {
	p.enterLoop()
	defer p.leaveLoop()

	p.next()
	if p.curr.Type != BegGrp {
		return nil, p.unexpectedToken()
	}
	p.next()
	var (
		while WhileLoop
		err   error
	)
	if while.cdt, err = p.parseExpr(bindLowest); err != nil {
		return nil, err
	}
	if p.curr.Type != EndGrp {
		return nil, p.unexpectedToken()
	}
	p.next()
	if while.csq, err = p.parseBody(); err != nil {
		return nil, err
	}
	if p.curr.Type == Else {
		p.next()
		if while.alt, err = p.parseBody(); err != nil {
			return nil, err
		}
	}
	return while, nil
}

func (p *Parser) parseForeach() (Expr, error) {
	p.enterLoop()
	defer p.leaveLoop()
	return nil, nil
}

func (p *Parser) parseFor() (Expr, error) {
	p.enterLoop()
	defer p.leaveLoop()
	return nil, nil
}

func (p *Parser) parseLet() (Expr, error) {
	p.next()
	if p.curr.Type != Ident {
		return nil, p.unexpectedToken()
	}
	let := Assignment{
		ident: p.curr,
	}
	p.next()
	if p.curr.Type != Assign {
		return nil, p.unexpectedToken()
	}
	p.next()
	expr, err := p.parseExpr(bindLowest)
	if err != nil {
		return nil, err
	}
	let.expr = expr
	if p.curr.Type != EOL {
		return nil, p.unexpectedToken()
	}
	p.next()
	return let, nil
}

func (p *Parser) parseReturn() (Expr, error) {
	p.next()
	expr, err := p.parseExpr(bindLowest)
	if err != nil {
		return nil, err
	}
	if p.curr.Type != EOL {
		return nil, p.unexpectedToken()
	}
	p.next()
	return Return{expr: expr}, nil
}

func (p *Parser) parseMacro(obj *Object) error {
	macro := p.curr
	p.next()
	if p.curr.Type != BegGrp {
		return p.unexpectedToken()
	}
	p.next()
	args, err := p.parseMapArgs()
	if err != nil {
		return err
	}
	switch p.curr.Type {
	case Comment:
		p.next()
	case EOL:
		p.next()
	default:
		return p.unexpectedToken()
	}
	call, ok := p.macros[macro.Input]
	if !ok {
		return fmt.Errorf("%s: %w macro", macro.Input, ErrUndefined)
	}
	node, err := call(args)
	if err != nil || node == nil {
		return err
	}
	return obj.merge(node)
}

func (p *Parser) parseExpr(bind int) (Expr, error) {
	prefix, ok := p.prefix[p.curr.Type]
	if !ok {
		return nil, p.unexpectedToken()
	}
	left, err := prefix()
	if err != nil {
		return nil, err
	}
	for !p.curr.exprDone() && bind < p.bindCurrent() {
		infix, ok := p.infix[p.curr.Type]
		if !ok {
			return nil, p.unexpectedToken()
		}
		if left, err = infix(left); err != nil {
			return nil, err
		}
	}
	return left, nil
}

func (p *Parser) parseArray() (Expr, error) {
	p.next()
	var arr Array
	for !p.done() && p.curr.Type != EndArr {
		var (
			expr Expr
			err  error
		)
		if p.curr.Type == BegArr {
			expr, err = p.parseArray()
		} else {
			expr, err = p.parseExpr(bindLowest)
		}
		if err != nil {
			return nil, err
		}
		arr.expr = append(arr.expr, expr)
		switch p.curr.Type {
		case Comma:
			p.next()
		case EndArr:
		case EOL:
			if p.peek.Type != EndArr {
				return nil, p.unexpectedToken()
			}
			p.next()
		default:
			return nil, p.unexpectedToken()
		}
	}
	if p.curr.Type != EndArr {
		return nil, p.unexpectedToken()
	}
	p.next()
	return arr, nil
}

func (p *Parser) parseUnary() (Expr, error) {
	var (
		expr Unary
		err  error
	)
	switch p.curr.Type {
	case Not, Bnot, Sub, Add:
		expr.op = p.curr.Type
		p.next()
	default:
		return nil, p.unexpectedToken()
	}
	expr.right, err = p.parseExpr(bindLowest)
	return expr, err
}

func (p *Parser) parseLiteral() (Expr, error) {
	if !p.curr.IsLiteral() {
		return nil, p.unexpectedToken()
	}
	if p.inFunction() && p.curr.Type == Ident {
		curr := p.curr
		curr.Type = EnvVar
		p.next()
		return makeVariable(curr), nil
	}
	expr := makeLiteral(p.curr)
	p.next()
	if p.curr.Type == Ident {
		switch p.curr.Input {
		case "ms":
			expr.mul = millis
		case "s", "sec":
		case "min":
			expr.mul = secPerMin
		case "h", "hour":
			expr.mul = secPerHour
		case "d", "day":
			expr.mul = secPerDay
		case "b", "B":
		case "k":
			expr.mul = iecKilo
		case "K":
			expr.mul = siKilo
		case "m":
			expr.mul = iecMega
		case "M":
			expr.mul = siMega
		case "g":
			expr.mul = iecGiga
		case "G":
			expr.mul = siGiga
		case "t":
			expr.mul = iecTera
		case "T":
			expr.mul = siTera
		default:
			return nil, p.unexpectedToken()
		}
		p.next()
	}
	return expr, nil
}

func (p *Parser) parseVariable() (Expr, error) {
	if !p.curr.IsVariable() {
		return nil, p.unexpectedToken()
	}
	expr := makeVariable(p.curr)
	p.next()
	return expr, nil
}

func (p *Parser) parseInfix(left Expr) (Expr, error) {
	var (
		err  error
		bind = p.bindCurrent()
		expr = Binary{
			left: left,
			op:   p.curr.Type,
		}
	)
	p.next()
	if expr.right, err = p.parseExpr(bind); err != nil {
		return nil, err
	}
	return expr, nil
}

func (p *Parser) parseAssignment(left Expr) (Expr, error) {
	lit, ok := left.(Variable)
	if !ok {
		return nil, fmt.Errorf("")
	}
	p.next()
	expr, err := p.parseExpr(bindLowest)
	if err != nil {
		return nil, err
	}
	a := Assignment{
		ident: lit.tok,
		expr:  expr,
	}
	return a, nil
}

func (p *Parser) parseIndex(left Expr) (Expr, error) {
	p.next()
	ix := Index{
		arr: left,
	}
	ptr, err := p.parseExpr(bindLowest)
	if err != nil {
		return nil, err
	}
	ix.ptr = ptr
	if p.curr.Type != EndArr {
		return nil, p.unexpectedToken()
	}
	p.next()
	return ix, nil
}

func (p *Parser) parseGroup() (Expr, error) {
	p.next()
	expr, err := p.parseExpr(bindLowest)
	if err != nil {
		return nil, err
	}
	if p.curr.Type != EndGrp {
		return nil, p.unexpectedToken()
	}
	p.next()
	return expr, nil
}

func (p *Parser) parseTernary(left Expr) (Expr, error) {
	t := Ternary{
		cdt: left,
	}

	p.next()
	var err error
	if t.csq, err = p.parseExpr(bindCdt); err != nil {
		return nil, err
	}
	if p.curr.Type != Assign {
		return nil, p.unexpectedToken()
	}
	p.next()
	if t.alt, err = p.parseExpr(bindLowest); err != nil {
		return nil, err
	}
	return t, nil
}

func (p *Parser) parseCall(left Expr) (Expr, error) {
	p.next()
	name, ok := left.(Literal)
	if !ok || name.tok.Type != Ident {
		return nil, p.syntaxError()
	}
	call := Call{
		name: name.tok,
	}
	args, err := p.parseArgs()
	if err != nil {
		return nil, err
	}
	call.args = args
	return call, nil
}

func (p *Parser) parseArgs() ([]Expr, error) {
	var args []Expr
	for !p.done() && p.curr.Type != EndGrp {
		expr, err := p.parseExpr(bindLowest)
		if err != nil {
			return nil, err
		}
		args = append(args, expr)
		switch p.curr.Type {
		case EndGrp:
		case Comma:
			if p.peek.Type == EndGrp {
				return nil, p.syntaxError()
			}
			p.next()
		default:
			return nil, p.unexpectedToken()
		}
	}
	if p.curr.Type != EndGrp {
		return nil, p.unexpectedToken()
	}
	p.next()
	return args, nil
}

func (p *Parser) parseMapArgs() (map[string]Expr, error) {
	args := make(map[string]Expr)
	for !p.done() && p.curr.Type != EndGrp {
		if p.curr.Type != Ident {
			return nil, p.unexpectedToken()
		}
		key := p.curr
		p.next()
		if p.curr.Type != Assign {
			return nil, p.unexpectedToken()
		}
		p.next()
		expr, err := p.parseExpr(bindLowest)
		if err != nil {
			return nil, err
		}
		if _, ok := args[key.Input]; ok {
			return nil, fmt.Errorf("%s: duplicate argument", key.Input)
		}
		args[key.Input] = expr
		switch p.curr.Type {
		case EndGrp:
		case Comma:
			if p.peek.Type == EndGrp {
				return nil, p.syntaxError()
			}
			if p.peek.Type == EOL {
				p.next()
			}
			p.next()
		default:
			return nil, p.unexpectedToken()
		}
	}
	if p.curr.Type != EndGrp {
		return nil, p.unexpectedToken()
	}
	p.next()
	return args, nil
}

func (p *Parser) bindCurrent() int {
	return powers[p.curr.Type]
}

func (p *Parser) bindPeek() int {
	return powers[p.peek.Type]
}

func (p *Parser) enterLoop() {
	p.loop++
}

func (p *Parser) leaveLoop() {
	p.loop--
}

func (p *Parser) inLoop() bool {
	return p.loop > 0
}

func (p *Parser) enterFunction() {
	p.userfn++
}

func (p *Parser) leaveFunction() {
	p.userfn--
}

func (p *Parser) inFunction() bool {
	return p.userfn > 0
}

func (p *Parser) unexpectedToken() error {
	return fmt.Errorf("parser error: %s %w: %s", p.curr.Position, ErrUnexpected, p.curr)
}

func (p *Parser) syntaxError() error {
	return fmt.Errorf("%w at %s", ErrSyntax, p.curr.Position)
}

func (p *Parser) done() bool {
	return p.curr.Type == EOF
}

func (p *Parser) next() {
	p.curr = p.peek
	p.peek = p.scan.Scan()
}
