package fig

import (
	"bytes"
	"io"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	kwTrue     = "true"
	kwFalse    = "false"
	kwOn       = "on"
	kwOff      = "off"
	kwYes      = "yes"
	kwNo       = "no"
	kwInf      = "inf"
	kwNan      = "nan"
	kwIf       = "if"
	kwElse     = "else"
	kwFor      = "for"
	kwWhile    = "while"
	kwLet      = "let"
	kwReturn   = "return"
	kwForeach  = "foreach"
	kwIn       = "in"
	kwBreak    = "break"
	kwContinue = "continue"
)

var keywords = []string{
	kwIf,
	kwElse,
	kwFor,
	kwWhile,
	kwLet,
	kwReturn,
	kwForeach,
	kwIn,
	kwBreak,
	kwContinue,
}

func init() {
	sort.Strings(keywords)
}

func isKeyword(str string) bool {
	i := sort.SearchStrings(keywords, str)
	return i < len(keywords) && keywords[i] == str
}

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
		s.skipBlank()
		return tok
	}
	return tok
}

func (s *Scanner) scanHeredoc(tok *Token) {
}

func (s *Scanner) scanIdent(tok *Token) {
}

func (s *Scanner) scanMacro(tok *Token) {
}

func (s *Scanner) scanNumber(tok *Token) {
}

func (s *Scanner) scanFraction(tok *Token) {
}

func (s *Scanner) scanExponent(tok *Token) {
}

func (s *Scanner) scanDate(tok *Token) {
}

func (s *Scanner) scanTimeOffset(tok *Token) {
}

func (s *Scanner) scanTime(tok *Token) {
}

func (s *Scanner) scanHexa(tok *Token) {
}

func (s *Scanner) scanOctal(tok *Token) {
}

func (s *Scanner) scanBin(tok *Token) {
}

func (s *Scanner) scanIntegerWithBase(accept func(b rune) bool) string {
	return ""
}

func (s *Scanner) scanString(tok *Token) {
}

func (s *Scanner) scanDelimiter(tok *Token) {
}

func (s *Scanner) scanComment(tok *Token, long bool) {
}

func (s *Scanner) scanCommentMultiline(tok *Token) {
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
	return b == lsquare || b == rsquare || b == lcurly || b == rcurly || b == comma || b == semicolon
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
