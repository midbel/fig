package fig

import (
	"fmt"
)

type Position struct {
	Line int
	Col  int
}

func (p Position) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Col)
}

const (
	EOF rune = -(iota + 1)
	Ident
	Comment
	Macro
	Boolean
	Heredoc
	String
	Integer
	Float
	LocalVar
	EnvVar
	BegArr
	EndArr
	BegObj
	EndObj
	BegGrp
	EndGrp
	Comma
	Assign
	EOL
	Invalid
)

var types = map[rune]string{
	EOF:      "eof",
	Ident:    "ident",
	Comment:  "comment",
	Macro:    "macro",
	Heredoc:  "heredoc",
	String:   "string",
	Integer:  "integer",
	Float:    "float",
	Boolean:  "boolean",
	BegArr:   "beg-arr",
	EndArr:   "end-arr",
	BegObj:   "beg-obj",
	EndObj:   "end-obj",
	BegGrp:   "beg-grp",
	EndGrp:   "end-grp",
	Comma:    "comma",
	Assign:   "assignment",
	EOL:      "eol",
	Invalid:  "invalid",
	LocalVar: "local-var",
	EnvVar:   "env-var",
}

type Token struct {
	Literal     string
	Type        rune
	Interpolate bool
	Position
}

func makeToken(str string, kind rune) Token {
	return Token{
		Literal: str,
		Type:    kind,
	}
}

func (t Token) Equal(other Token) bool {
	return t.Literal == other.Literal && t.Type == other.Type
}

func (t Token) isComment() bool {
	return t.Type == Comment
}

func (t Token) isIdent() bool {
	return t.Type == Ident || t.Type == String || t.Type == Integer
}

func (t Token) isLiteral() bool {
	switch t.Type {
	case Integer, Float, String, Boolean, Heredoc, Ident:
		return true
	default:
		return false
	}
}

func (t Token) isNumber() bool {
	return t.Type == Integer || t.Type == Float
}

func (t Token) isValue() bool {
	return t.isLiteral() || t.isVariable()
}

func (t Token) isVariable() bool {
	return t.Type == LocalVar || t.Type == EnvVar
}

func (t Token) isEOL() bool {
	return t.Type == EOL
}

func (t Token) isZero() bool {
	return t.Literal == "" && t.Type == 0
}

func (t Token) String() string {
	prefix, ok := types[t.Type]
	if !ok {
		return "<unknown>"
	}
	if t.Literal == "" {
		return fmt.Sprintf("<%s>", prefix)
	}
	return fmt.Sprintf("<%s(%s)>", prefix, t.Literal)
}
