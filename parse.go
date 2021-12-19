package fig

import (
	"errors"
	"fmt"
	"io"
	"strconv"
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

	macros map[string]macrodef
}

func NewParser(r io.Reader) (*Parser, error) {
	sc, err := Scan(r)
	if err != nil {
		return nil, err
	}

	var p Parser
	p.scan = sc
	p.macros = map[string]macrodef{
		"include":  createMacroDef(Include, false),
		"define":   createMacroDef(Define, true),
		"apply":    createMacroDef(Apply, false),
		"extend":   createMacroDef(Extend, true),
		"repeat":   createMacroDef(Repeat, true),
		"readfile": createMacroDef(ReadFile, false),
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
		return p.unexpected()
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
				return p.unexpected()
			}
			nest, err1 = nest.getObject(p.curr.Literal, p.peek.Type == BegObj)
			if err1 != nil {
				return err1
			}
			p.next()
		}
		if p.curr.Type != BegObj {
			return p.unexpected()
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
		err = p.unexpected()
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
		return p.unexpected()
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
		n = createVariable(p.curr)
		p.next()
		n, err = p.parseSlice(n)
	case p.curr.isLiteral():
		if p.curr.Type == Ident && p.peek.Type == BegGrp {
			return p.parseCall()
		}
		i := createLiteral(p.curr)
		p.next()
		if i.Token.isNumber() && p.curr.isIdent() {
			i.Mul = p.curr
			p.next()
		}
		n = i
	case p.curr.isEOL():
	case p.curr.isComment():
	default:
		return nil, p.unexpected()
	}
	return n, err
}

func (p *Parser) parseCall() (Node, error) {
	var (
		c   = createCall(p.curr.Literal)
		err error
	)
	p.next()
	c.Args, c.Kwargs, err = p.parseArgs()
	return c, err
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
		return p.unexpected()
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
		case p.curr.isLiteral() || p.curr.isVariable():
			n, err = p.parseValue()
		case p.curr.Type == BegArr:
			n, err = p.parseArray()
		default:
			return nil, p.unexpected()
		}
		if err != nil {
			return nil, err
		}
		arr.Nodes = append(arr.Nodes, n)
		switch p.curr.Type {
		case Comment:
			p.parseComment()
			if p.curr.Type != EndArr {
				return nil, p.unexpected()
			}
		case Comma:
			p.next()
			if p.curr.isComment() {
				p.parseComment()
			}
		case EndArr:
		case EOL:
			if p.peek.Type != EndArr {
				return nil, p.unexpected()
			}
			p.next()
		default:
			return nil, p.unexpected()
		}
	}
	if p.curr.Type != EndArr {
		return nil, p.unexpected()
	}
	p.next()
	return arr, nil
}

func (p *Parser) parseSlice(node Node) (Node, error) {
	if p.curr.Type != BegArr {
		return node, nil
	}
	p.next()
	var (
		err error
		slc = createSlice(node)
	)
	switch p.curr.Type {
	case Integer:
		slc.from.index, err = strconv.ParseInt(p.curr.Literal, 0, 64)
		if err != nil {
			return nil, err
		}
		slc.from.set = true
		p.next()
	case Slice:
		slc.from.set = true
	default:
		return nil, p.unexpected()
	}
	switch p.curr.Type {
	case EndArr:
		slc.from = slc.to
	case Slice:
		p.next()
	default:
		return nil, p.unexpected()
	}
	switch p.curr.Type {
	case Integer:
		slc.to.index, err = strconv.ParseInt(p.curr.Literal, 0, 64)
		if err != nil {
			return nil, err
		}
		slc.to.set = true
		p.next()
	case EndArr:
	default:
		return nil, p.unexpected()
	}
	if p.curr.Type != EndArr {
		return nil, p.unexpected()
	}
	p.next()
	return slc, nil
}

func (p *Parser) parseMacro(obj *object) error {
	p.next()
	if p.curr.Type != Ident {
		return p.unexpected()
	}
	ident := p.curr
	p.next()
	if p.curr.Type != BegGrp {
		return p.unexpected()
	}
	args, kwargs, err := p.parseArgs()
	if err != nil {
		return err
	}
	def, ok := p.macros[ident.Literal]
	if !ok {
		return fmt.Errorf("%s: undefined macro", ident.Literal)
	}
	var nest Node
	if def.withobject {
		if p.curr.Type != BegObj {
			return p.unexpected()
		}
		// tmp := createObject("")
		tmp := enclosedObject("", obj)
		if err := p.parseObject(tmp); err != nil {
			return err
		}
		nest = tmp
	}
	err = p.parseEOL()
	if err != nil {
		return err
	}
	return def.macroFunc(obj, nest, args, kwargs)
}

func (p *Parser) parseArgs() ([]Node, map[string]Node, error) {
	p.next()
	var (
		named  bool
		args   []Node
		kwargs = make(map[string]Node)
	)
	for !p.done() {
		if p.curr.Type == EndGrp {
			break
		}
		if !named && p.peek.Type == Assign {
			named = true
		}
		if !named {
			n, err := p.parseValue()
			if err != nil {
				return nil, nil, err
			}
			args = append(args, n)
		} else {
			if p.curr.Type != Ident {
				return nil, nil, p.unexpected()
			}
			if _, ok := kwargs[p.curr.Literal]; ok {
				return nil, nil, p.unexpected()
			}
			ident := p.curr
			p.next()
			if p.curr.Type != Assign {
				return nil, nil, p.unexpected()
			}
			p.next()
			n, err := p.parseValue()
			if err != nil {
				return nil, nil, err
			}
			kwargs[ident.Literal] = n
		}
		switch p.curr.Type {
		case Comma:
			p.next()
		case EndGrp:
		default:
			return nil, nil, p.unexpected()
		}
	}
	if p.curr.Type != EndGrp {
		return nil, nil, p.unexpected()
	}
	p.next()
	return args, kwargs, nil
}

func (p *Parser) parseComment() {
	for p.curr.isComment() {
		p.next()
	}
}

func (p *Parser) unexpected() error {
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
