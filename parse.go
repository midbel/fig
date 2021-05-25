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
	Lshift:       bindShift,
	Rshift:       bindShift,
	Band:         bindBin,
	Bor:          bindBin,
	And:          bindRel,
	Or:           bindRel,
	Lt:           bindCmp,
	Le:           bindCmp,
	Gt:           bindCmp,
	Ge:           bindCmp,
	Equal:        bindCmp,
	NotEqual:     bindCmp,
	Add:          bindAdd,
	Sub:          bindAdd,
	Mul:          bindMul,
	Div:          bindMul,
	Mod:          bindMul,
	Pow:          bindPow,
	BegGrp:       bindCall,
	BegArr:       bindIndex,
	Assign:       bindAssign,
	AddAssign:    bindAssign,
	SubAssign:    bindAssign,
	MulAssign:    bindAssign,
	DivAssign:    bindAssign,
	ModAssign:    bindAssign,
	LshiftAssign: bindAssign,
	RshiftAssign: bindAssign,
	BandAssign:   bindAssign,
	BorAssign:    bindAssign,
	Question:     bindCdt,
}

type Parser struct {
	scan *Scanner
	curr Token
	peek Token

	infix    map[rune]func(Expr) (Expr, error)
	prefix   map[rune]func() (Expr, error)
	keywords map[rune]func() (Expr, error)
	macros   map[string]func(map[string]Expr) (Node, error)

	loop  int
	block int
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
	p.keywords = map[rune]func() (Expr, error){
		Let:      p.parseLet,
		Ret:      p.parseReturn,
		If:       p.parseIf,
		For:      p.parseFor,
		While:    p.parseWhile,
		Foreach:  p.parseForeach,
		Break:    p.parseBreak,
		Continue: p.parseContinue,
	}
	p.prefix = map[rune]func() (Expr, error){
		Add:      p.parseUnary,
		Sub:      p.parseUnary,
		Not:      p.parseUnary,
		Bnot:     p.parseUnary,
		Ident:    p.parseIdentifier,
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
		Add:          p.parseInfix,
		Sub:          p.parseInfix,
		Div:          p.parseInfix,
		Mul:          p.parseInfix,
		Mod:          p.parseInfix,
		Pow:          p.parseInfix,
		And:          p.parseInfix,
		Or:           p.parseInfix,
		Gt:           p.parseInfix,
		Lt:           p.parseInfix,
		Ge:           p.parseInfix,
		Le:           p.parseInfix,
		Equal:        p.parseInfix,
		NotEqual:     p.parseInfix,
		Lshift:       p.parseInfix,
		Rshift:       p.parseInfix,
		Band:         p.parseInfix,
		Bor:          p.parseInfix,
		Bnot:         p.parseInfix,
		BegGrp:       p.parseCall,
		BegArr:       p.parseIndex,
		Question:     p.parseTernary,
		Assign:       p.parseAssignment,
		AddAssign:    p.parseAssignment,
		SubAssign:    p.parseAssignment,
		MulAssign:    p.parseAssignment,
		DivAssign:    p.parseAssignment,
		ModAssign:    p.parseAssignment,
		BandAssign:   p.parseAssignment,
		BorAssign:    p.parseAssignment,
		LshiftAssign: p.parseAssignment,
		RshiftAssign: p.parseAssignment,
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
	var (
		err error
		tok Token
	)
	for i := 0; !p.done() && (p.curr.Type != BegObj && p.curr.Type != BegArr); i++ {
		if !p.curr.IsIdent() {
			return p.unexpectedToken()
		}
		switch p.peek.Type {
		case BegObj:
			obj, err = obj.insert(p.curr)
		case BegArr:
			tok = p.curr
		default:
			obj, err = obj.get(p.curr)
		}
		if err != nil {
			return err
		}
		p.next()
	}
	switch p.curr.Type {
	case BegObj:
		err = p.parseObject(obj)
	case BegArr:
		err = p.parseList(tok, obj)
	default:
		err = p.unexpectedToken()
	}
	return err
}

func (p *Parser) parseList(tok Token, obj *Object) error {
	p.next()
	if p.curr.Type != BegObj {
		return p.unexpectedToken()
	}
	for !p.done() && p.curr.Type != EndArr {
		o, err := obj.insert(tok)
		if err != nil {
			return err
		}
		if err = p.parseObject(o); err != nil {
			return err
		}
		p.skip(EOL)
	}
	if p.curr.Type != EndArr {
		return p.unexpectedToken()
	}
	p.next()
	p.skip(EOL)
	return nil
}

func (p *Parser) parseObject(obj *Object) error {
	p.next()
	for !p.done() && p.curr.Type != EndObj {
		if p.curr.Type == EOL {
			p.next()
			continue
		}
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
	expr, err := p.parseExpression()
	if err != nil {
		return nil, fmt.Errorf("parsing expression: %w", err)
	}
	if p.curr.Type == EOL {
		p.next()
	}
	return expr, err
}

func (p *Parser) parseFunction() (Func, error) {
	var (
		fn  Func
		err error
	)
	fn.name = p.curr
	p.next()
	if fn.args, err = p.parseArguments(); err != nil {
		return fn, err
	}
	if fn.body, err = p.parseBody(); err != nil {
		return fn, err
	}
	return fn, nil
}

func (p *Parser) parseArguments() ([]Argument, error) {
	var (
		args   []Argument
		err    error
		onlykw bool
	)
	p.next()
	for i := 0; !p.done() && p.curr.Type != EndGrp; i++ {
		if p.curr.Type != Ident {
			return nil, p.unexpectedToken()
		}
		a := Argument{
			name: p.curr,
			pos:  i,
		}
		p.next()
		if onlykw && p.curr.Type != Assign {
			return nil, p.syntaxError()
		}
		if p.curr.Type == Assign {
			p.next()
			if a.expr, err = p.parseExpr(bindLowest); err != nil {
				return nil, err
			}
			onlykw = true
		}
		switch p.curr.Type {
		case Comma:
			if p.peek.Type != Ident {
				return nil, p.syntaxError()
			}
			p.next()
		case EndGrp:
		case EOL:
			if p.peek.Type != EndGrp {
				return nil, p.syntaxError()
			}
			p.next()
		default:
			return nil, p.unexpectedToken()
		}
		args = append(args, a)
	}
	if p.curr.Type != EndGrp {
		return nil, p.unexpectedToken()
	}
	p.next()
	return args, nil
}

func (p *Parser) parseExpression() (Expr, error) {
	parse, ok := p.keywords[p.curr.Type]
	if ok {
		e, err := parse()
		return e, err
	}
	switch p.curr.Type {
	case EOL, Comment:
		p.next()
	case BegObj:
		return p.parseBody()
	default:
		return p.parseExpr(bindLowest)
	}
	return nil, nil
}

func (p *Parser) parseBreak() (Expr, error) {
	if !p.inLoop() {
		return nil, p.syntaxError()
	}
	p.next()
	if p.curr.Type != EOL {
		return nil, p.unexpectedToken()
	}
	p.next()
	return BreakLoop{}, nil
}

func (p *Parser) parseContinue() (Expr, error) {
	if !p.inLoop() {
		return nil, p.syntaxError()
	}
	p.next()
	if p.curr.Type != EOL {
		return nil, p.unexpectedToken()
	}
	p.next()
	return ContinueLoop{}, nil
}

func (p *Parser) parseBody() (Expr, error) {
	if p.curr.Type != BegObj {
		return nil, p.unexpectedToken()
	}
	p.enterBlock()
	defer p.leaveBlock()
	p.next()
	var b Block
	for p.curr.Type != EndObj {
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if expr == nil {
			continue
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
	}
	return ter, nil
}

func (p *Parser) parseWhile() (Expr, error) {
	p.enterLoop()
	defer p.leaveLoop()

	p.next()

	var (
		while WhileLoop
		err   error
	)

	switch p.curr.Type {
	case BegObj:
	case BegGrp:
		p.next()
		if while.cdt, err = p.parseExpression(); err != nil {
			return nil, err
		}
		if p.curr.Type != EndGrp {
			return nil, p.unexpectedToken()
		}
		p.next()
	default:
		return nil, p.unexpectedToken()
	}
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
	p.next()
	if p.curr.Type != BegGrp {
		return nil, p.unexpectedToken()
	}
	p.next()
	if p.curr.Type != Ident {
		return nil, p.unexpectedToken()
	}

	p.enterLoop()
	defer p.leaveLoop()

	var (
		foreach ForeachLoop
		err     error
	)
	foreach.ident = p.curr
	p.next()
	if p.curr.Type == Comma {
		p.next()
		if p.curr.Type != Ident {
			return nil, p.unexpectedToken()
		}
		foreach.loop = foreach.ident
		foreach.ident = p.curr
		p.next()
	}
	if p.curr.Type != Keyword && p.curr.Input != kwIn {
		return nil, p.unexpectedToken()
	}
	p.next()
	if foreach.expr, err = p.parseExpression(); err != nil {
		return nil, err
	}
	if p.curr.Type != EndGrp {
		return nil, p.unexpectedToken()
	}
	p.next()
	if foreach.body, err = p.parseBody(); err != nil {
		return nil, err
	}
	if p.curr.Type == Else {
		p.next()
		if foreach.alt, err = p.parseBody(); err != nil {
			return nil, err
		}
	}
	return foreach, nil
}

func (p *Parser) parseFor() (Expr, error) {
	p.next()

	p.enterLoop()
	defer p.leaveLoop()

	p.enterBlock()
	defer p.leaveBlock()

	var (
		forloop ForLoop
		err     error
	)

	switch p.curr.Type {
	case BegObj:
	case BegGrp:
		p.next()
		if p.curr.Type != Semicolon {
			if forloop.init, err = p.parseExpression(); err != nil {
				return nil, err
			}
		} else {
			p.next()
		}

		if p.curr.Type != Semicolon {
			if forloop.cdt, err = p.parseExpr(bindLowest); err != nil {
				return nil, err
			}
			if p.curr.Type != Semicolon {
				return nil, p.unexpectedToken()
			}
			p.next()
		} else {
			p.next()
		}

		if p.curr.Type != EndGrp {
			if forloop.next, err = p.parseExpr(bindLowest); err != nil {
				return nil, err
			}
			if p.curr.Type != EndGrp {
				return nil, p.unexpectedToken()
			}
		}
		p.next()
	default:
		return nil, p.unexpectedToken()
	}

	if forloop.body, err = p.parseBody(); err != nil {
		return nil, err
	}
	if p.curr.Type == Else {
		p.next()
		if forloop.alt, err = p.parseBody(); err != nil {
			return nil, err
		}
	}
	return forloop, nil
}

func (p *Parser) parseLet() (Expr, error) {
	p.next()
	if !p.inBlock() && !p.inLoop() {
		return nil, p.syntaxError()
	}
	if p.curr.Type != Ident {
		return nil, p.unexpectedToken()
	}
	let := Assignment{
		ident: p.curr,
		let:   true,
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
	if p.curr.Type != EOL && p.curr.Type != Semicolon {
		return nil, p.unexpectedToken()
	}
	p.next()
	return let, nil
}

func (p *Parser) parseReturn() (Expr, error) {
	if !p.inBlock() {
		return nil, p.syntaxError()
	}
	p.next()
	expr, err := p.parseExpr(bindLowest)
	if err != nil {
		return nil, err
	}
	if p.curr.Type != EOL && p.curr.Type != Comment {
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

func (p *Parser) parseIdentifier() (Expr, error) {
	if p.inBlock() {
		expr := makeIdentifier(p.curr)
		p.next()
		return expr, nil
	}
	expr := makeLiteral(p.curr)
	p.next()
	return expr, nil
}

func (p *Parser) parseLiteral() (Expr, error) {
	if !p.curr.IsLiteral() {
		return nil, p.unexpectedToken()
	}
	if p.curr.Interpolate {
		defer p.next()
		return parseTemplate(p.curr.Input)
	}
	expr := makeLiteral(p.curr)
	if mul, ok := multipliers[p.peek.Input]; ok {
		expr.mul = mul
		p.next()
	}
	p.next()
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
	lit, ok := left.(Identifier)
	if !ok {
		return nil, p.syntaxError()
	}
	op, ok := assignments[p.curr.Type]
	if !ok {
		return nil, p.unexpectedToken()
	}
	p.next()
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if op != Assign {
		expr = Binary{
			left:  left,
			right: expr,
			op:    op,
		}
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

	var (
		call Call
		err  error
	)
	switch left := left.(type) {
	case Identifier:
		if !p.inBlock() {
			return nil, p.syntaxError()
		}
		call.name = left.tok
	case Literal:
		if left.tok.Type != Ident {
			return nil, p.syntaxError()
		}
		call.name = left.tok
	default:
		return nil, p.syntaxError()
	}
	call.args, err = p.parseArgs()
	return call, err
}

func (p *Parser) parseArgs() ([]Argument, error) {
	var (
		args   []Argument
		err    error
		onlykw bool
		seen   = make(map[string]struct{})
	)
	for i := 0; !p.done() && p.curr.Type != EndGrp; i++ {
		a := Argument{
			pos: i,
		}
		if p.curr.Type == Ident && p.peek.Type == Assign {
			if _, ok := seen[p.curr.Input]; ok {
				return nil, fmt.Errorf("%s: duplicate argument", p.curr.Input)
			}
			seen[p.curr.Input] = struct{}{}
			a.name = p.curr
			onlykw = true
			p.next()
			p.next()
		}
		if onlykw && a.name.isZero() {
			return nil, p.syntaxError()
		}
		if a.expr, err = p.parseExpr(bindLowest); err != nil {
			return nil, err
		}
		args = append(args, a)
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

func (p *Parser) enterLoop() {
	p.loop++
}

func (p *Parser) leaveLoop() {
	p.loop--
}

func (p *Parser) inLoop() bool {
	return p.loop > 0
}

func (p *Parser) enterBlock() {
	p.block++
}

func (p *Parser) leaveBlock() {
	p.block--
}

func (p *Parser) inBlock() bool {
	return p.block > 0
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

func (p *Parser) skip(kind rune) {
	for p.curr.Type == kind {
		p.next()
	}
}

func (p *Parser) next() {
	p.curr = p.peek
	p.peek = p.scan.Scan()
}
