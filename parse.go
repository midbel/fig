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
}

func NewParser(r io.Reader) (*Parser, error) {
	sc, err := Scan(r)
	if err != nil {
		return nil, err
	}

	var p Parser
	p.scan = sc
	p.next()
	p.next()

	return &p, nil
}

func Parse(r io.Reader) error {
	p, err := NewParser(r)
	if err != nil {
		return err
	}
	return p.Parse()
}

func (p *Parser) Parse() error {
	if p.curr.Type == BegObj {
		return p.parseObject()
	}
	for !p.done() {
		if err := p.parse(); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) parse() error {
	if p.curr.isComment() {
		p.parseComment()
		return nil
	}
	switch {
	case p.curr.Type == Macro:
		return p.parseMacro()
	case p.curr.isIdent():
	default:
		return p.unexpectedToken()
	}
	p.next()
	var (
		err error
		n   node
	)
	switch {
	case p.curr.isIdent():
		for !p.done() {
			if p.curr.Type == BegObj {
				break
			}
			if !p.curr.isIdent() {
				return p.unexpectedToken()
			}
			p.next()
		}
		if p.curr.Type != BegObj {
			return p.unexpectedToken()
		}
		err = p.parseObject()
	case p.curr.Type == BegObj:
		err = p.parseObject()
	case p.curr.Type == Assign:
		p.next()
		n, err = p.parseValue()
		fmt.Println("parse", n)
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
	default:
		return p.unexpectedToken()
	}
	return nil
}

func (p *Parser) parseValue() (node, error) {
	var (
		n   node
		err error
	)
	switch {
	case p.curr.Type == BegArr:
		n, err = p.parseArray()
	case p.curr.isLiteral() || p.curr.isVariable():
		n = createLiteral(p.curr)
		p.next()
	default:
		return nil, p.unexpectedToken()
	}
	return n, err
}

func (p *Parser) parseObject() error {
	p.next()
	for !p.done() {
		if p.curr.Type == EndObj {
			break
		}
		if err := p.parse(); err != nil {
			return err
		}
	}
	if p.curr.Type != EndObj {
		return p.unexpectedToken()
	}
	p.next()
	return nil
}

func (p *Parser) parseArray() (node, error) {
	var (
		arr array
		n   node
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

func (p *Parser) parseMacro() error {
	p.next()
	if p.curr.Type != Ident {
		return p.unexpectedToken()
	}
	p.next()
	if p.curr.Type != BegGrp {
		return p.unexpectedToken()
	}
	if err := p.parseArgs(); err != nil {
		return err
	}
	return p.parseEOL()
}

func (p *Parser) parseArgs() error {
	p.next()
	var named bool
	for !p.done() {
		if p.curr.Type == EndGrp {
			break
		}
		if !named && p.peek.Type == Assign {
			named = true
		}
		if !named {
			if !p.curr.isLiteral() {
				return p.unexpectedToken()
			}
		} else {
			if p.curr.Type != Ident {
				return p.unexpectedToken()
			}
			p.next()
			if p.curr.Type != Assign {
				return p.unexpectedToken()
			}
			p.next()
			if !p.curr.isLiteral() {
				return p.unexpectedToken()
			}
		}
		p.next()
		switch p.curr.Type {
		case Comma:
			p.next()
		case EndGrp:
		default:
			return p.unexpectedToken()
		}
	}
	if p.curr.Type != EndGrp {
		return p.unexpectedToken()
	}
	p.next()
	return nil
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
