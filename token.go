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
	Question
	EOL
	Not
	Assign
	Add
	AddAssign
	Sub
	SubAssign
	Mul
	MulAssign
	Div
	DivAssign
	Mod
	ModAssign
	Pow
	Gt
	Lt
	Ge
	Le
	Lshift
	LshiftAssign
	Rshift
	RshiftAssign
	Band
	BandAssign
	Bor
	BorAssign
	Bnot
	And
	Or
	Equal
	NotEqual
	BegGrp
	EndGrp
	LocalVar
	EnvVar
	Invalid
	Let
	Ret
	If
	For
	While
	Else
	Foreach
	Break
	Continue
	Increment
	Decrement
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
	Question:     "question",
	EOL:          "eol",
	Not:          "not",
	Assign:       "assign",
	Add:          "add",
	AddAssign:    "add-assign",
	Sub:          "sub",
	SubAssign:    "sub-assign",
	Mul:          "mul",
	MulAssign:    "mul-assign",
	Div:          "div",
	DivAssign:    "div-assign",
	Mod:          "mod",
	ModAssign:    "mod-assign",
	Pow:          "pow",
	Gt:           "gt",
	Lt:           "lt",
	Ge:           "ge",
	Le:           "le",
	Lshift:       "left-shift",
	LshiftAssign: "left-shift-assign",
	Rshift:       "right-shift",
	RshiftAssign: "right-shift-assign",
	Band:         "bin-and",
	BandAssign:   "bin-and-assign",
	Bor:          "bin-or",
	BorAssign:    "bin-or-assign",
	Bnot:         "bin-not",
	And:          "and",
	Or:           "or",
	Equal:        "eq",
	NotEqual:     "ne",
	BegGrp:       "beg-grp",
	EndGrp:       "end-grp",
	LocalVar:     "local-var",
	EnvVar:       "env-bar",
	Invalid:      "invalid",
	Let:          "let",
	Ret:          "return",
	If:           "if",
	For:          "for",
	While:        "while",
	Else:         "else",
	Foreach:      "foreach",
	Break:        "break",
	Continue:     "continue",
	Increment:    "increment",
	Decrement:    "decrement",
}

var assignments = map[rune]rune{
	Assign:       Assign,
	AddAssign:    Add,
	SubAssign:    Sub,
	MulAssign:    Mul,
	DivAssign:    Div,
	ModAssign:    Mod,
	BandAssign:   Band,
	BorAssign:    Bor,
	LshiftAssign: Lshift,
	RshiftAssign: Rshift,
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

func (t Token) IsVariable() bool {
	return t.Type == LocalVar || t.Type == EnvVar
}

func (t Token) isZero() bool {
	return t.Input == "" && t.Type == 0
}

func (t Token) exprDone() bool {
	switch t.Type {
	case Comma, Semicolon, Comment, EOL, EOF, EndArr:
		return true
	default:
		return false
	}
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
