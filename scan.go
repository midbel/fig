package fig

import (
	"bytes"
	"io"
	"unicode/utf8"
)

const (
	kwTrue  = "true"
	kwFalse = "false"
	kwOn    = "on"
	kwOff   = "off"
	kwYes   = "yes"
	kwNo    = "no"
	kwNull  = "null"
	kwInf   = "inf"
	kwNan   = "nan"
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
		s.scanComment(&tok)
		return tok
	}
	switch {
	case isNL(s.char):
		tok.Type = EOL
		s.skipNewline()
	case isLetter(s.char):
		s.scanIdent(&tok)
	case isQuote(s.char):
		s.scanString(&tok)
	case isDelim(s.char):
		s.scanDelimiter(&tok)
	case isDigit(s.char):
		s.scanNumber(&tok)
	case isOperator(s.char):
		s.scanOperator(&tok)
	case isMacro(s.char):
		s.scanMacro(&tok)
	case isVariable(s.char):
		s.scanVariable(&tok)
	default:
		tok.Type = Invalid
	}
	if tok.Type == Heredoc {
		s.scanHeredoc(&tok)
	}
	s.skipBlank()
	return tok
}

func (s *Scanner) scanHeredoc(tok *Token) {
	for isUpper(s.char) {
		s.str.WriteRune(s.char)
		s.read()
	}
	if !isNL(s.char) {
		tok.Type = Invalid
		return
	}
	var (
		label = s.str.String()
		prev  string
		tmp   bytes.Buffer
	)
	s.str.Reset()
	for !isEOF(s.char) {
		s.read()
		for !isNL(s.char) && !isEOF(s.char) {
			tmp.WriteRune(s.char)
			s.read()
		}
		line := tmp.String()
		if line == label {
			break
		}
		tmp.WriteRune(nl)
		io.Copy(&s.str, &tmp)
		prev = line
	}
	tok.Type = String
	tok.Input = s.str.String()
	if len(prev) == 0 {
		tok.Type = Invalid
	}
}

func (s *Scanner) scanVariable(tok *Token) {
	var kind rune
	switch s.char {
	case dollar:
		kind = LocalVar
	case arobase:
		kind = EnvVar
	}
	s.read()
	if !isLetter(s.char) {
		tok.Type = Invalid
		return
	}
	s.scanIdent(tok)
	tok.Type = kind

}

func (s *Scanner) scanIdent(tok *Token) {
	for isIdent(s.char) {
		s.str.WriteRune(s.char)
		s.read()
	}
	tok.Input = s.str.String()
	switch tok.Input {
	case kwTrue, kwFalse, kwOn, kwOff, kwYes, kwNo:
		tok.Type = Boolean
	case kwNull:
		tok.Type = Null
	case kwInf, kwNan:
		tok.Type = Float
	default:
		tok.Type = Ident
	}
}

func (s *Scanner) scanMacro(tok *Token) {
	s.read()
	for isLetter(s.char) || s.char == underscore {
		s.str.WriteRune(s.char)
		s.read()
	}
	tok.Type = Macro
	tok.Input = s.str.String()
}

func (s *Scanner) scanNumber(tok *Token) {
	var zeros int
	if s.char == '0' {
		peek := s.peek()
		switch peek {
		case 'x':
			s.scanHexa(tok)
		case 'o':
			s.scanOctal(tok)
		case 'b':
			s.scanBin(tok)
		}
		if peek == 'x' || peek == 'o' || peek == 'b' {
			return
		}
		s.str.WriteRune(s.char)
		s.read()
		for s.char == '0' {
			s.str.WriteRune(s.char)
			s.read()
			zeros++
		}
	}
	for isDigit(s.char) {
		s.str.WriteRune(s.char)
		s.read()
		if s.char == underscore {
			s.read()
			if !isDigit(s.char) {
				tok.Type = Invalid
				return
			}
		}
	}
	switch peek := s.peek(); {
	case (s.char == 'e' || s.char == 'E') && isDigit(peek):
		s.scanExponent(tok)
	case s.char == dot && isDigit(peek):
		s.scanFraction(tok)
	case s.char == colon && isDigit(peek):
		s.scanTime(tok)
	case s.char == minus && isDigit(peek):
		s.scanDate(tok)
	default:
		tok.Type = Integer
		tok.Input = s.str.String()
	}
	if (tok.Type == Integer || tok.Type == Float) && zeros > 0 {
		tok.Type = Invalid
	}
}

func (s *Scanner) scanFraction(tok *Token) {
	s.read()
	s.str.WriteRune(dot)
	for isDigit(s.char) {
		s.str.WriteRune(s.char)
		s.read()
		if s.char == underscore {
			s.read()
			if !isDigit(s.char) {
				tok.Type = Invalid
			}
		}
	}
	tok.Type = Float
	tok.Input = s.str.String()
	if s.char == 'e' || s.char == 'E' {
		s.scanExponent(tok)
	}
}

func (s *Scanner) scanExponent(tok *Token) {
	s.str.WriteRune(s.char)
	s.read()
	if isSign(s.char) {
		s.str.WriteRune(s.char)
		s.read()
	}
	for isDigit(s.char) {
		s.str.WriteRune(s.char)
		s.read()
		if s.char == underscore {
			s.read()
			if !isDigit(s.char) {
				tok.Type = Invalid
				return
			}
		}
	}
	tok.Type = Float
	tok.Input = s.str.String()
}

func (s *Scanner) scanDate(tok *Token) {
	scan := func() bool {
		for i := 0; i < 2; i++ {
			if !isDigit(s.char) {
				return false
			}
			s.str.WriteRune(s.char)
			s.read()
		}
		return true
	}
	s.str.WriteRune(s.char)
	s.read()
	for i := 0; i < 2; i++ {
		if !scan() || (i == 0 && s.char != minus) {
			tok.Type = Invalid
			tok.Input = s.str.String()
			return
		}
		if i == 0 {
			s.str.WriteRune(s.char)
			s.read()
		}
	}
	tok.Type = Date
	tok.Input = s.str.String()
	if s.char == 'T' || s.char == space {
		s.scanTime(tok)
		if tok.Type != Time {
			tok.Input = s.str.String()
			return
		}
		s.scanTimeOffset(tok)
		if tok.Type == Invalid {
			return
		}
		tok.Type = DateTime
		tok.Input = s.str.String()
	}
}

func (s *Scanner) scanTimeOffset(tok *Token) {
	if s.char == 'Z' {
		s.str.WriteRune(s.char)
		s.read()
		return
	}
	if !isSign(s.char) {
		return
	}
	scan := func() bool {
		for i := 0; i < 2; i++ {
			if !isDigit(s.char) {
				return false
			}
			s.str.WriteRune(s.char)
			s.read()
		}
		return true
	}
	s.str.WriteRune(s.char)
	s.read()
	for i := 0; i < 2; i++ {
		if !scan() || (i == 0 && s.char != colon) {
			tok.Type = Invalid
			tok.Input = s.str.String()
			return
		}
		if i == 0 {
			s.str.WriteRune(s.char)
			s.read()
		}
	}
}

func (s *Scanner) scanTime(tok *Token) {
	scan := func() bool {
		for i := 0; i < 2; i++ {
			if !isDigit(s.char) {
				return false
			}
			s.str.WriteRune(s.char)
			s.read()
		}
		return true
	}
	count := 2
	if s.char == 'T' || s.char == ' ' {
		count++
	}
	s.str.WriteRune(s.char)
	s.read()
	for i := 0; i < count; i++ {
		if !scan() && i < count-1 && s.char != colon {
			tok.Type = Invalid
			tok.Input = s.str.String()
			return
		}
		if i < count-1 {
			s.str.WriteRune(s.char)
			s.read()
		}
	}
	if s.char == dot {
		s.str.WriteRune(s.char)
		s.read()
		for i := 0; isDigit(s.char) && i < 9; i++ {
			s.str.WriteRune(s.char)
			s.read()
		}
	}
	tok.Type = Time
	tok.Input = s.str.String()
}

func (s *Scanner) scanHexa(tok *Token) {
	s.read()
	s.read()
	s.str.WriteRune('0')
	s.str.WriteRune('x')
	tok.Type = Integer
	tok.Input = s.scanIntegerWithBase(isHex)
	if tok.Input == "" {
		tok.Type = Invalid
	}
}

func (s *Scanner) scanOctal(tok *Token) {
	s.read()
	s.read()
	s.str.WriteRune('0')
	s.str.WriteRune('o')
	tok.Type = Integer
	tok.Input = s.scanIntegerWithBase(isOctal)
	if tok.Input == "" {
		tok.Type = Invalid
	}
}

func (s *Scanner) scanBin(tok *Token) {
	s.read()
	s.read()
	s.str.WriteRune('0')
	s.str.WriteRune('b')
	tok.Type = Integer
	tok.Input = s.scanIntegerWithBase(isBin)
	if tok.Input == "" {
		tok.Type = Invalid
	}
}

func (s *Scanner) scanIntegerWithBase(accept func(b rune) bool) string {
	for accept(s.char) {
		s.str.WriteRune(s.char)
		s.read()
		if s.char == underscore {
			s.read()
			if !isDigit(s.char) {
				return ""
			}
		}
	}
	return s.str.String()
}

func (s *Scanner) scanString(tok *Token) {
	quote := s.char
	s.read()
	for s.char != quote {
		if quote == dquote && s.char == backslash {
			s.read()
			s.char = escapes[s.char]
		}
		s.str.WriteRune(s.char)
		s.read()
	}
	tok.Type = String
	tok.Input = s.str.String()
	s.read()
}

func (s *Scanner) scanOperator(tok *Token) {
	prev := s.prev()
	switch s.char {
	case colon:
		tok.Type = Assign
	case equal:
		tok.Type = Assign
		if peek := s.peek(); peek == equal {
			tok.Type = Equal
			s.read()
		}
	case bang:
		tok.Type = Not
		if peek := s.peek(); peek == equal {
			tok.Type = NotEqual
			s.read()
		}
	case plus:
		tok.Type = Add
	case minus:
		tok.Type = Sub
	case slash:
		tok.Type = Div
	case star:
		tok.Type = Mul
		if peek := s.peek(); peek == star {
			tok.Type = Pow
			s.read()
		}
	case percent:
		tok.Type = Mod
	case langle:
		tok.Type = Lt
		if peek := s.peek(); peek == equal {
			tok.Type = Le
			s.read()
		} else if peek == langle {
			tok.Type = Lshift
			s.read()
			if isUpper(s.peek()) {
				tok.Type = Heredoc
			}
		}
	case rangle:
		tok.Type = Gt
		if peek := s.peek(); peek == equal {
			tok.Type = Ge
			s.read()
		} else if s.peek(); peek == rangle {
			tok.Type = Rshift
			s.read()
		}
	case lparen:
		tok.Type = BegGrp
	case rparen:
		tok.Type = EndGrp
	case ampersand:
		tok.Type = Band
		if peek := s.peek(); peek == ampersand {
			tok.Type = And
			s.read()
		}
	case pipe:
		tok.Type = Bor
		if peek := s.peek(); peek == pipe {
			tok.Type = Or
			s.read()
		}
	case tilde:
		tok.Type = Bnot
	}
	if !isGroup(s.char) && tok.Type != Assign && tok.Type != Heredoc {
		var (
			next   = s.peek()
			before = isGroup(prev) || isBlank(prev)
			after  = isGroup(next) || isBlank(next)
		)
		if !before && !after {
			tok.Type = Invalid
		}
	}
	s.read()
}

func (s *Scanner) scanDelimiter(tok *Token) {
	var kind rune
	switch s.char {
	case lsquare:
		kind = BegArr
	case rsquare:
		kind = EndArr
	case lcurly:
		kind = BegObj
	case rcurly:
		kind = EndObj
	case comma:
		kind = Comma
	default:
	}
	tok.Type = kind
	s.read()
	s.skipBlank()
	s.skipNewline()
}

func (s *Scanner) scanComment(tok *Token) {
	s.read()
	if isBlank(s.char) {
		s.skipBlank()
	}

	for !isNL(s.char) && !isEOF(s.char) {
		s.str.WriteRune(s.char)
		s.read()
	}
	s.skipNewline()
	tok.Type = Comment
	tok.Input = s.str.String()
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

func (s *Scanner) unread() {
	if s.next <= 0 || s.char == zero {
		return
	}
	if s.char == nl {
		s.line--
		s.column = s.seen
	} else {
		s.column--
	}
	s.next = s.curr
	s.curr -= utf8.RuneLen(s.char)
	s.char, _ = utf8.DecodeRune(s.input[s.curr:])
}

func (s *Scanner) skipBlank() {
	s.skip(isBlank)
}

func (s *Scanner) skipNewline() {
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
	return b == nl
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
	return b == lsquare || b == rsquare || b == lcurly || b == rcurly || b == comma
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

func isOperator(b rune) bool {
	return isSign(b) || b == tilde || b == pipe || b == ampersand ||
		b == colon || b == equal || b == bang || b == slash ||
		b == star || b == percent || b == langle || b == rangle ||
		isGroup(b)
}

func isVariable(b rune) bool {
	return b == dollar || b == arobase
}

func isGroup(b rune) bool {
	return b == rparen || b == lparen
}
