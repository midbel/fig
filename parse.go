package fig

import (
  "errors"
  "fmt"
  "io"
)

var (
	ErrUnexpected = errors.New("unexpected token")
	ErrSyntax     = errors.New("syntax error")
)

const (
	kilo = 1024
	mega = kilo * kilo
	giga = mega * kilo
	tera = giga * kilo
)

const (
	bindLowest = iota
	bindAssign
	bindRel
	bindCmp
	bindBin
	bindShift
	bindAdd
	bindMul
	bindPow
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
	Assign:   bindAssign,
}

type Expr interface {
	Eval() error
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

func (i Literal) String() string {
	return fmt.Sprintf("literal(%s)", i.tok.Input)
}

func (i Literal) Eval() error {
	return nil
}

type Variable struct {
	tok Token
}

func (v Variable) String() string {
	return fmt.Sprintf("variable(%s)", v.tok.Input)
}

func (v Variable) Eval() error {
	return nil
}

type Node interface{}

type Note struct {
	pre  []string
	post string
}

type Table struct {
	name  Token
	nodes []Node

	Note
}

type Option struct {
	name Token
	expr Expr

	Note
}

type Parser struct {
	scan *Scanner
	curr Token
	peek Token

	infix  map[rune]func(Expr) (Expr, error)
	prefix map[rune]func() (Expr, error)
}

func Parse(r io.Reader) error {
	sc, err := Scan(r)
	if err != nil {
		return err
	}

	var p Parser
	p.scan = sc
	p.prefix = map[rune]func() (Expr, error){
		Add:      p.parseUnary,
		Sub:      p.parseUnary,
		Not:      p.parseUnary,
		Bnot:     p.parseUnary,
		Ident:    p.parseLiteral,
		Integer:  p.parseLiteral,
		Float:    p.parseLiteral,
		String:   p.parseLiteral,
		Date:     p.parseLiteral,
		Time:     p.parseLiteral,
		DateTime: p.parseLiteral,
		Boolean:  p.parseLiteral,
		Null:     p.parseLiteral,
		LocalVar: p.parseVariable,
		EnvVar:   p.parseVariable,
		BegGrp:   p.parseGroup,
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
		BegGrp:   p.parseCall,
	}
	p.next()
	p.next()

	return p.Parse()
}

func (p *Parser) Parse() error {
	for !p.done() {
		fmt.Println("current", p.curr)
		if err := p.parse(); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) parse() error {
	for p.curr.Type == Comment {
		p.next()
	}
	var paths []Token
	for !p.done() {
		if p.curr.Type == Assign || p.curr.Type == BegObj {
			break
		}
		paths = append(paths, p.curr)
		p.next()
	}
	var err error
	if p.curr.Type == Assign {
		if len(paths) != 1 {
			return p.syntaxError()
		}
		var expr Expr
		expr, err = p.parseValue()
		fmt.Println(expr, err)
		if p.curr.Type == Comment {
			p.next()
		}
	} else if p.curr.Type == BegObj {
		if len(paths) < 1 {
			return p.syntaxError()
		}
		err = p.parseObject()
	} else {
		err = p.unexpectedToken()
	}
	return err
}

func (p *Parser) parseObject() error {
	p.next()
	for !p.done() {
		if err := p.parse(); err != nil {
			return err
		}
		if p.curr.Type == EndObj {
			break
		}
	}
	if p.curr.Type != EndObj {
		return p.unexpectedToken()
	}
	p.next()
	if p.curr.Type == Comment {
		p.next()
	}
	return nil
}

func (p *Parser) parseArray() (Expr, error) {
	p.next()
	for !p.done() {
		if p.curr.Type == EndArr {
			break
		}
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
		_ = expr
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
	return nil, nil
}

func (p *Parser) parseValue() (Expr, error) {
	p.next()
	if p.curr.Type == BegArr {
		return p.parseArray()
	}
	expr, err := p.parseExpr(bindLowest)
	if err != nil {
		err = fmt.Errorf("parsing expression: %w", err)
	}
	if p.curr.Type == EOL {
		p.next()
	}
	return expr, err
}

func (p *Parser) exprDone() bool {
	switch p.curr.Type {
	case Comma, Comment, EOL, EOF, EndArr:
		return true
	default:
		return false
	}
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
	for !p.exprDone() && bind < p.bindCurrent() {
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
	var expr Expr
	switch p.curr.Type {
	case Ident:
	case Integer:
	case Float:
	case String:
	case Date:
	case Time:
	case DateTime:
	case Boolean:
	case Null:
	default:
		return nil, p.unexpectedToken()
	}
	expr = Literal{tok: p.curr}
	p.next()
	if p.curr.Type == Ident {
		p.next()
	}
	return expr, nil
}

func (p *Parser) parseVariable() (Expr, error) {
	var expr Expr
	switch p.curr.Type {
	case LocalVar:
	case EnvVar:
	default:
		return nil, p.unexpectedToken()
	}
	expr = Variable{tok: p.curr}
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

func (p *Parser) parseCall(left Expr) (Expr, error) {
	return nil, nil
}

func (p *Parser) bindCurrent() int {
	return powers[p.curr.Type]
}

func (p *Parser) bindPeek() int {
	return powers[p.peek.Type]
}

func (p *Parser) unexpectedToken() error {
	return fmt.Errorf("%s %w: %s", p.curr.Position, ErrUnexpected, p.curr)
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
