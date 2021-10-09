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
	Keyword
	Ident
	Comment
	Macro
	Heredoc
	String
	Integer
	Float
	Date
	DateTime
	Time
	Boolean
	BegArr
	EndArr
	BegObj
	EndObj
	Comma
	Semicolon
)

var types = map[rune]string{
	EOF:          "eof",
	Keyword:      "keyword",
	Ident:        "ident",
	Comment:      "comment",
	Macro:        "macro",
	Heredoc:      "heredoc",
	String:       "string",
	Integer:      "integer",
	Float:        "float",
	Date:         "date",
	DateTime:     "datetime",
	Time:         "time",
	Boolean:      "boolean",
	BegArr:       "beg-arr",
	EndArr:       "end-arr",
	BegObj:       "beg-obj",
	EndObj:       "end-obj",
	Comma:        "comma",
	Semicolon:    "semicolon",
}

type Token struct {
	Input       string
	Type        rune
	Interpolate bool
	Position
}

func makeToken(str string, kind rune) Token {
	return Token{
		Input: str,
		Type:  kind,
	}
}

func (t Token) Equal(other Token) bool {
	return t.Input == other.Input && t.Type == other.Type
}

func (t Token) IsComment() bool {
	return t.Type == Comment
}

func (t Token) IsIdent() bool {
	return t.Type == Ident || t.Type == String || t.Type == Integer
}

func (t Token) IsLiteral() bool {
	switch t.Type {
	case Integer, Float, String, Date, Time, DateTime, Boolean, Heredoc:
		return true
	default:
		return false
	}
}

func (t Token) isZero() bool {
	return t.Input == "" && t.Type == 0
}

func (t Token) String() string {
	prefix, ok := types[t.Type]
	if !ok {
		return "<unknown>"
	}
	if t.Input == "" {
		return fmt.Sprintf("<%s>", prefix)
	}
	return fmt.Sprintf("<%s(%s)>", prefix, t.Input)
}
