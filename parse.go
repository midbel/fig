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

type Parser struct {
	scan *Scanner
	curr Token
	peek Token

	macros map[string]macroFunc
}

func NewParser(r io.Reader) (*Parser, error) {
	sc, err := Scan(r)
	if err != nil {
		return nil, err
	}

	var p Parser
	p.scan = sc
	p.macros = map[string]macroFunc{
		"include": Include,
	}
	p.next()
	p.next()
	return &p, nil

}

func Parse(r io.Reader) (Node, error) {
	p, err := NewParser(r)
	if err != nil {
		return nil, err
	}
	return p.Parse()
}

func (p *Parser) Parse() (Node, error) {
	for p.curr.isEOL() {
		p.next()
	}
	obj := createObject("root")
	if p.curr.Type == BegObj {
		return obj, p.parseObject(obj)
	}
	for !p.done() {
		if err := p.parse(obj); err != nil {
			return nil, err
		}
	}
	return obj, nil
}

func (p *Parser) parse(obj *object) error {
	if p.curr.isComment() {
		p.parseComment()
		return nil
	}
	var ident Token
	switch {
	case p.curr.Type == Macro:
		return p.parseMacro(obj)
	case p.curr.isIdent():
		ident = p.curr
	default:
		return p.unexpectedToken()
	}
	p.next()
	var (
		err error
		n   Node
	)
	switch {
	case p.curr.isIdent():
		nest, err1 := obj.getObject(ident.Literal, false)
		if err1 != nil {
			return err1
		}
		for !p.done() {
			if p.curr.Type == BegObj {
				break
			}
			if !p.curr.isIdent() {
				return p.unexpectedToken()
			}
			nest, err1 = nest.getObject(p.curr.Literal, p.peek.Type == BegObj)
			if err1 != nil {
				return err1
			}
			p.next()
		}
		if p.curr.Type != BegObj {
			return p.unexpectedToken()
		}
		err = p.parseObject(nest)
	case p.curr.Type == BegObj:
		nest, err1 := obj.getObject(ident.Literal, true)
		if err1 != nil {
			return err1
		}
		err = p.parseObject(nest)
	case p.curr.Type == Assign:
		p.next()
		n, err = p.parseValue()
		if err == nil {
			n = createOption(ident.Literal, n)
			err = obj.set(n)
		}
	default:
		err = p.unexpectedToken()
	}
	if err != nil {
		return err
	}
	return p.parseEOL()
}

func (p *Parser) parseEOL() error {
	switch p.curr.Type {
	case EOL:
		p.next()
		if p.curr.isComment() {
			p.parseComment()
		}
	case Comment:
		p.parseComment()
	case EOF:
	default:
		return p.unexpectedToken()
	}
	return nil
}

func (p *Parser) parseValue() (Node, error) {
	var (
		n   Node
		err error
	)
	switch {
	case p.curr.Type == BegArr:
		n, err = p.parseArray()
	case p.curr.isVariable():
		n = createLiteral(p.curr)
		p.next()
	case p.curr.isLiteral():
		i := createLiteral(p.curr)
		p.next()
		if p.curr.isIdent() {
			i.Mul = p.curr
			p.next()
		}
		n = i
	case p.curr.isEOL():
	case p.curr.isComment():
	default:
		return nil, p.unexpectedToken()
	}
	return n, err
}

func (p *Parser) parseObject(obj *object) error {
	p.next()
	for !p.done() {
		if p.curr.Type == EndObj {
			break
		}
		if err := p.parse(obj); err != nil {
			return err
		}
	}
	if p.curr.Type != EndObj {
		return p.unexpectedToken()
	}
	p.next()
	return nil
}

func (p *Parser) parseArray() (Node, error) {
	var (
		arr = createArray()
		n   Node
	)
	p.next()
	for !p.done() {
		if p.curr.Type == EndArr {
			break
		}
		if p.curr.isComment() {
			p.parseComment()
		}
		var err error
		switch {
		case p.curr.isLiteral():
			n, err = p.parseValue()
		case p.curr.Type == BegArr:
			n, err = p.parseArray()
		default:
			return nil, p.unexpectedToken()
		}
		if err != nil {
			return nil, err
		}
		arr.Nodes = append(arr.Nodes, n)
		switch p.curr.Type {
		case Comment:
			p.parseComment()
			if p.curr.Type != EndArr {
				return nil, p.unexpectedToken()
			}
		case Comma:
			p.next()
			if p.curr.isComment() {
				p.parseComment()
			}
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

func (p *Parser) parseMacro(obj *object) error {
	p.next()
	if p.curr.Type != Ident {
		return p.unexpectedToken()
	}
	ident := p.curr
	p.next()
	if p.curr.Type != BegGrp {
		return p.unexpectedToken()
	}
	args, kwargs, err := p.parseArgs()
	if err != nil {
		return err
	}
	err = p.parseEOL()
	if err != nil {
		return err
	}
	macro, ok := p.macros[ident.Literal]
	if !ok {
		return fmt.Errorf("%s: undefined macro", ident.Literal)
	}
	return macro(obj, args, kwargs)
}

func (p *Parser) parseArgs() ([]Argument, map[string]Argument, error) {
	p.next()
	var (
		named  bool
		args   []Argument
		kwargs = make(map[string]Argument)
	)
	for !p.done() {
		if p.curr.Type == EndGrp {
			break
		}
		if !named && p.peek.Type == Assign {
			named = true
		}
		if !named {
			if !p.curr.isLiteral() {
				return nil, nil, p.unexpectedToken()
			}
			args = append(args, createLiteral(p.curr))
		} else {
			if p.curr.Type != Ident {
				return nil, nil, p.unexpectedToken()
			}
			if _, ok := kwargs[p.curr.Literal]; ok {
				return nil, nil, p.unexpectedToken()
			}
			ident := p.curr
			p.next()
			if p.curr.Type != Assign {
				return nil, nil, p.unexpectedToken()
			}
			p.next()
			if !p.curr.isLiteral() {
				return nil, nil, p.unexpectedToken()
			}
			kwargs[ident.Literal] = createLiteral(p.curr)
		}
		p.next()
		switch p.curr.Type {
		case Comma:
			p.next()
		case EndGrp:
		default:
			return nil, nil, p.unexpectedToken()
		}
	}
	if p.curr.Type != EndGrp {
		return nil, nil, p.unexpectedToken()
	}
	p.next()
	return args, kwargs, nil
}

func (p *Parser) parseComment() {
	for p.curr.isComment() {
		p.next()
	}
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
