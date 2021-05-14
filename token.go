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
	EOF = -(iota + 1)
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
	Input string
	Type  rune
	Position
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
	case Comma, Comment, EOL, EOF, EndArr:
		return true
	default:
		return false
	}
}

func (t Token) String() string {
	var prefix string
	switch t.Type {
	case EOF:
		return "<eof>"
	case EOL:
		return "<eol>"
	case Ident:
		prefix = "ident"
	case Macro:
		prefix = "macro"
	case Comment:
		prefix = "comment"
	case Heredoc:
		prefix = "heredoc"
	case String:
		prefix = "string"
	case Integer:
		prefix = "integer"
	case Float:
		prefix = "float"
	case Date:
		prefix = "date"
	case DateTime:
		prefix = "datetime"
	case Time:
		prefix = "time"
	case Boolean:
		prefix = "boolean"
	case BegArr:
		return "<beg-arr>"
	case EndArr:
		return "<end-arr>"
	case BegGrp:
		return "<beg-grp>"
	case EndGrp:
		return "<end-grp>"
	case BegObj:
		return "<beg-obj>"
	case EndObj:
		return "<end-obj>"
	case Assign:
		return "<assign>"
	case Comma:
		return "<comma>"
	case Question:
		return "<question>"
	case Invalid:
		prefix = "invalid"
	case Not:
		return "<not>"
	case Equal:
		return "<eq>"
	case NotEqual:
		return "<ne>"
	case Gt:
		return "<gt>"
	case Lt:
		return "<lt>"
	case Ge:
		return "<ge>"
	case Le:
		return "<le>"
	case Add:
		return "<add>"
	case AddAssign:
		return "<add-assign>"
	case Sub:
		return "<subtract>"
	case SubAssign:
		return "<subtract-assign>"
	case Div:
		return "<divide>"
	case DivAssign:
		return "<divide-assign>"
	case Mul:
		return "<multiply>"
	case MulAssign:
		return "<multiply-assign>"
	case Mod:
		return "<modulo>"
	case ModAssign:
		return "<modulo-assign>"
	case Pow:
		return "<power>"
	case And:
		return "<and>"
	case Or:
		return "<or>"
	case Band:
		return "<bin-and>"
	case BandAssign:
		return "<bin-and-assign>"
	case Bor:
		return "<bin-or>"
	case BorAssign:
		return "<bin-or-assign>"
	case Bnot:
		return "<bin-not>"
	case Lshift:
		return "<left-shift>"
	case LshiftAssign:
		return "<left-shift-assign>"
	case Rshift:
		return "<right-shift>"
	case RshiftAssign:
		return "<right-shift-assign>"
	case LocalVar:
		prefix = "local"
	case EnvVar:
		prefix = "env"
	case Keyword:
		prefix = "keyword"
	case Ret:
		return "<return>"
	case Let:
		return "<let>"
	case If:
		return "<if>"
	case For, While, Foreach:
		prefix = "loop"
	case Break:
		return "<break>"
	case Continue:
		return "<continue>"
	case Increment:
		return "<increment>"
	case Decrement:
		return "<decrement>"
	default:
		prefix = "unknown"
	}
	return fmt.Sprintf("<%s(%s)>", prefix, t.Input)
}
