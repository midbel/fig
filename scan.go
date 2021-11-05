package fig

import (
	"bytes"
	"io"
	"strings"
	"unicode/utf8"
)

const (
	zero       = 0
	cr         = '\r'
	nl         = '\n'
	tab        = '\t'
	space      = ' '
	squote     = '\''
	dquote     = '"'
	lcurly     = '{'
	rcurly     = '}'
	lsquare    = '['
	rsquare    = ']'
	lparen     = '('
	rparen     = ')'
	langle     = '<'
	rangle     = '>'
	bang       = '!'
	dot        = '.'
	underscore = '_'
	comma      = ','
	dollar     = '$'
	arobase    = '@'
	plus       = '+'
	minus      = '-'
	slash      = '/'
	star       = '*'
	percent    = '%'
	equal      = '='
	colon      = ':'
	pound      = '#'
	backslash  = '\\'
	ampersand  = '&'
	pipe       = '|'
	tilde      = '~'
	question   = '?'
	semicolon  = ';'
)

var escapes = map[rune]rune{
	'n':       nl,
	'r':       cr,
	dquote:    dquote,
	backslash: backslash,
}

type Scanner struct {
	input []byte
	curr  int
	next  int
	char  rune

	str bytes.Buffer

	line   int
	column int
	seen   int
}

func Scan(r io.Reader) (*Scanner, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	s := Scanner{
		input:  bytes.ReplaceAll(buf, []byte{cr, nl}, []byte{nl}),
		line:   1,
		column: 0,
	}
	s.read()
	return &s, nil
}

func (s *Scanner) Scan() Token {
	s.reset()
	s.skipBlank()
	var tok Token
	tok.Position = Position{
		Line: s.line,
		Col:  s.column,
	}
	if s.char == 0 || s.char == utf8.RuneError {
		tok.Type = EOF
		return tok
	}
	if k := s.peek(); s.char == pound || s.char == slash && k == star {
		s.scanComment(&tok, s.char == slash && k == star)
		return tok
	} else if s.char == langle && k == s.char {
		s.scanHeredoc(&tok)
		return tok
	}
	switch {
	case isLetter(s.char):
		s.scanIdent(&tok)
	case isVariable(s.char):
		s.scanVariable(&tok)
	case isDigit(s.char) || isSign(s.char):
		s.scanNumber(&tok)
	case isQuote(s.char):
		s.scanString(&tok)
	case isDelim(s.char):
		s.scanDelimiter(&tok)
	case isMacro(s.char):
		tok.Type = Macro
		s.read()
	case isAssign(s.char):
		tok.Type = Assign
		s.read()
		s.skipBlank()
	case isNL(s.char):
		tok.Type = EOL
		s.skipNL()
	default:
		tok.Type = Invalid
	}
	return tok
}

func (s *Scanner) scanHeredoc(tok *Token) {
	s.read()
	s.read()
	if !isUpper(s.char) {
		tok.Type = Invalid
		return
	}
	var (
		tmp bytes.Buffer
		pfx string
	)
	for isUpper(s.char) {
		s.str.WriteRune(s.char)
		s.read()
	}
	pfx = s.str.String()
	s.str.Reset()
	if !isNL(s.char) {
		tok.Type = Invalid
		return
	}
	s.skipNL()
	for !s.isDone() {
		for !isNL(s.char) && !s.isDone() {
			tmp.WriteRune(s.char)
			s.read()
		}
		if tmp.String() == pfx {
			break
		}
		for isNL(s.char) {
			tmp.WriteRune(s.char)
			s.read()
		}
		io.Copy(&s.str, &tmp)
		tmp.Reset()
	}
	tok.Type = Heredoc
	tok.Literal = strings.TrimSpace(s.str.String())
}

func (s *Scanner) scanVariable(tok *Token) {
	tok.Type = LocalVar
	if s.char == arobase {
		tok.Type = EnvVar
	}
	s.read()
	if !isLetter(s.char) {
		tok.Type = Invalid
		return
	}
	for isIdent(s.char) {
		s.str.WriteRune(s.char)
		s.read()
	}
	tok.Literal = s.str.String()
}

func (s *Scanner) scanIdent(tok *Token) {
	for isIdent(s.char) {
		s.str.WriteRune(s.char)
		s.read()
	}
	tok.Literal = s.str.String()
	tok.Type = Ident
	switch tok.Literal {
	case "true", "false", "yes", "no", "on", "off":
		tok.Type = Boolean
	case "null":
	default:
	}
}

func (s *Scanner) scanMacro(tok *Token) {
}

func (s *Scanner) scanString(tok *Token) {
	quote := s.char
	s.read()
	for !s.isDone() {
		s.str.WriteRune(s.char)
		s.read()
		if s.char == quote {
			s.read()
			break
		}
	}
	tok.Literal = s.str.String()
	tok.Type = String
	if s.isDone() {
		tok.Type = Invalid
	}
}

func (s *Scanner) scanNumber(tok *Token) {
	signed := isSign(s.char)
	if s.char == '0' {
		var ok bool
		switch k := s.peek(); k {
		case 'x':
			s.scanHexa(tok)
		case 'o':
			s.scanOctal(tok)
		case 'b':
			s.scanBin(tok)
		default:
			ok = true
		}
		if signed {
			tok.Type = Invalid
		}
		if !ok {
			return
		}
	}
	if signed {
		s.str.WriteRune(s.char)
		s.read()
	}
	tok.Type = Integer
	if ok := s.scanDigit(isDigit); !ok {
		tok.Type = Invalid
	}
	tok.Literal = s.str.String()
	if tok.Type == Invalid {
		return
	}
	switch s.char {
	case dot:
		s.scanFraction(tok)
	case colon:
	case minus:
	case 'e', 'E':
		s.scanExponent(tok)
	default:
	}
}

func (s *Scanner) scanFraction(tok *Token) {
	s.str.WriteRune(s.char)
	s.read()
	tok.Type = Float
	if ok := s.scanDigit(isDigit); !ok {
		tok.Type = Invalid
	}
	tok.Literal = s.str.String()
	if tok.Type == Invalid {
		return
	}
	if s.char == 'e' || s.char == 'E' {
		s.scanExponent(tok)
	}
	tok.Literal = s.str.String()
}

func (s *Scanner) scanExponent(tok *Token) {
	s.str.WriteRune(s.char)
	s.read()
	if isSign(s.char) {
		s.str.WriteRune(s.char)
		s.read()
	}
	tok.Type = Float
	if ok := s.scanDigit(isDigit); !ok {
		tok.Type = Invalid
	}
	tok.Literal = s.str.String()
}

func (s *Scanner) scanHexa(tok *Token) {
	s.read()
	s.read()
	s.str.WriteRune('0')
	s.str.WriteRune('x')
	tok.Type = Integer
	if ok := s.scanDigit(isHex); !ok {
		tok.Type = Invalid
	}
	tok.Literal = s.str.String()
}

func (s *Scanner) scanOctal(tok *Token) {
	s.read()
	s.read()
	s.str.WriteRune('0')
	s.str.WriteRune('o')
	tok.Type = Integer
	if ok := s.scanDigit(isOctal); !ok {
		tok.Type = Invalid
	}
	tok.Literal = s.str.String()
}

func (s *Scanner) scanBin(tok *Token) {
	s.read()
	s.read()
	s.str.WriteRune('0')
	s.str.WriteRune('b')

	tok.Type = Integer
	if ok := s.scanDigit(isBin); !ok {
		tok.Type = Invalid
	}
	tok.Literal = s.str.String()
}

func (s *Scanner) scanDigit(accept func(rune) bool) bool {
	for accept(s.char) {
		s.str.WriteRune(s.char)
		s.read()
		if s.char == underscore {
			s.read()
			if !accept(s.char) {
				return false
			}
		}
	}
	return true
}

func (s *Scanner) scanDelimiter(tok *Token) {
	switch s.char {
	case lsquare:
		tok.Type = BegArr
	case rsquare:
		tok.Type = EndArr
	case lcurly:
		tok.Type = BegObj
	case rcurly:
		tok.Type = EndObj
	case lparen:
		tok.Type = BegGrp
	case rparen:
		tok.Type = EndGrp
	case comma:
		tok.Type = Comma
	case semicolon:
		tok.Type = EOL
	}
	s.read()
	switch tok.Type {
	case Comma, EOL, BegArr, BegObj:
		s.skipBlank()
		s.skipNL()
	default:
	}
}

func (s *Scanner) scanDate(tok *Token) {
}

func (s *Scanner) scanTimeOffset(tok *Token) {
}

func (s *Scanner) scanTime(tok *Token) {
}

func (s *Scanner) scanComment(tok *Token, multi bool) {
	if multi {
		s.scanCommentMultiline(tok)
	}
	s.read()
	s.skipBlank()
	for !isNL(s.char) {
		s.str.WriteRune(s.char)
		s.read()
	}
	s.skipNL()
	tok.Literal = s.str.String()
	tok.Type = Comment
}

func (s *Scanner) scanCommentMultiline(tok *Token) {
}

func (s *Scanner) isDone() bool {
	return s.char == zero || s.char == utf8.RuneError
}

func (s *Scanner) peek() rune {
	r, _ := utf8.DecodeRune(s.input[s.next:])
	return r
}

func (s *Scanner) prev() rune {
	if s.curr == 0 {
		return zero
	}
	r, _ := utf8.DecodeLastRune(s.input[:s.curr])
	return r
}

func (s *Scanner) read() {
	if s.curr >= len(s.input) {
		s.char = 0
		return
	}
	r, n := utf8.DecodeRune(s.input[s.next:])
	if r == utf8.RuneError {
		s.char = 0
		s.next = len(s.input)
	}
	last := s.char
	s.char, s.curr, s.next = r, s.next, s.next+n

	if last == nl {
		s.line++
		s.seen, s.column = s.column, 1
	} else {
		s.column++
	}
}

func (s *Scanner) skipBlank() {
	s.skip(isBlank)
}

func (s *Scanner) skipNL() {
	s.skip(isNL)
}

func (s *Scanner) skip(fn func(rune) bool) {
	for fn(s.char) {
		s.read()
	}
}

func (s *Scanner) reset() {
	s.str.Reset()
}

func isNL(b rune) bool {
	return b == nl || b == cr
}

func isEOF(b rune) bool {
	return b == zero || b == utf8.RuneError
}

func isBlank(b rune) bool {
	return b == space || b == tab
}

func isIdent(b rune) bool {
	return isLetter(b) || isDigit(b) || b == underscore || b == minus
}

func isDigit(b rune) bool {
	return b >= '0' && b <= '9'
}

func isSign(b rune) bool {
	return b == plus || b == minus
}

func isQuote(b rune) bool {
	return b == dquote || b == squote
}

func isLetter(b rune) bool {
	return isLower(b) || isUpper(b)
}

func isLower(b rune) bool {
	return b >= 'a' && b <= 'z'
}

func isUpper(b rune) bool {
	return b >= 'A' && b <= 'Z'
}

func isDelim(b rune) bool {
	return b == lsquare || b == rsquare ||
		b == lcurly || b == rcurly ||
		b == lparen || b == rparen ||
		b == comma || b == semicolon
}

func isAssign(b rune) bool {
	return b == colon || b == equal
}

func isHex(b rune) bool {
	return isDigit(b) || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'Z')
}

func isBin(b rune) bool {
	return b == '0' || b == '1'
}

func isOctal(b rune) bool {
	return b >= '0' && b <= '7'
}

func isMacro(b rune) bool {
	return b == dot
}

func isVariable(b rune) bool {
	return b == dollar || b == arobase
}
