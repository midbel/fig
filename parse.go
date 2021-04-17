package fig

import (
	"errors"
	"fmt"
	"io"
	"strings"
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
	nodes []Node

	Note
}

func (o Object) String() string {
	return fmt.Sprintf("table(%s)", o.name.Input)
}

func (o Object) Has(str string) bool {
	return false
}

func (o Object) Get(str string) (Expr, error) {
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
		if _, err := p.parse(); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) parse() (Node, error) {
	var cs []string
	for p.curr.Type == Comment {
		cs = append(cs, p.curr.Input)
		p.next()
	}
	var (
		paths []Token
		err   error
	)
	if p.curr.Type == Macro {
		return p.parseMacro()
	}
	for !p.done() {
		if p.curr.Type == Assign || p.curr.Type == BegObj {
			break
		}
		if p.curr.Type != Ident && p.curr.Type != String && p.curr.Type != Integer {
			return nil, p.unexpectedToken()
		}
		paths = append(paths, p.curr)
		p.next()
	}
	switch p.curr.Type {
	case Assign:
		if len(paths) != 1 {
			return nil, p.syntaxError()
		}
		var opt Option

		opt.name = paths[0]
		opt.pre = append(opt.pre, cs...)
		if opt.expr, err = p.parseValue(); err != nil {
			return nil, err
		}
		if p.curr.Type == Comment {
			opt.post = p.curr.Input
			p.next()
		}
		fmt.Println(opt)
		return opt, nil
	case BegObj:
		if len(paths) < 1 {
			return nil, p.syntaxError()
		}
		err = p.parseObject()
		return nil, err
	default:
		return nil, p.unexpectedToken()
	}
}

func (p *Parser) parseMacro() (Node, error) {
	p.next()
	if p.curr.Type != BegGrp {
		return nil, p.unexpectedToken()
	}
	for !p.done() && p.curr.Type != EndGrp {
		p.next()
	}
	if p.curr.Type != EndGrp {
		return nil, p.unexpectedToken()
	}
	p.next()
	switch p.curr.Type {
	case Comment:
		p.next()
	case EOL:
		p.next()
	default:
		return nil, p.unexpectedToken()
	}
	return nil, nil
}

func (p *Parser) parseObject() error {
	p.next()
	for !p.done() {
		if _, err := p.parse(); err != nil {
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

func (p *Parser) parseValue() (Expr, error) {
	p.next()
	if p.curr.Type == BegArr {
		return p.parseArray()
	}
	if p.curr.Type == EOL {
		return nil, p.syntaxError()
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
