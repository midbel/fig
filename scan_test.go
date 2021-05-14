package fig

import (
	"strings"
	"testing"
)

func TestScan(t *testing.T) {
	const str = `
    key = value
    object {}
    array []
    group ()
    # scan integers
    1_011 0xdead_beef 0b1010 0o123
    # scan float
    1.123 0.2e+3 1e+312
    # scan booleans
    true false
    yes no
    on off
    # scan strings
    "double quoted string"
    'single quoted string'
    # scan variables
    @env
    $local
    # scan dates
    2021-04-18, 2021-04-18T19:16:45.321Z, 2021-04-18 19:16:45.321+02:00
    19:16:45, 19:16:45.123
		# scan macros
		.include
		/* long comment */
		/* very long /*nested long comment*/ and more */
		let return if for while else
		/* invalid comment
  `
	input := strings.ReplaceAll(str, "\r\n", "\n")
	sc, err := Scan(strings.NewReader(strings.TrimSpace(input)))
	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}
	tokens := []Token{
		makeToken("key", Ident),
		makeToken("", Assign),
		makeToken("value", Ident),
		makeToken("", EOL),
		makeToken("object", Ident),
		makeToken("", BegObj),
		makeToken("", EndObj),
		makeToken("array", Ident),
		makeToken("", BegArr),
		makeToken("", EndArr),
		makeToken("group", Ident),
		makeToken("", BegGrp),
		makeToken("", EndGrp),
		makeToken("", EOL),
		makeToken("scan integers", Comment),
		makeToken("1011", Integer),
		makeToken("0xdeadbeef", Integer),
		makeToken("0b1010", Integer),
		makeToken("0o123", Integer),
		makeToken("", EOL),
		makeToken("scan float", Comment),
		makeToken("1.123", Float),
		makeToken("0.2e+3", Float),
		makeToken("1e+312", Float),
		makeToken("", EOL),
		makeToken("scan booleans", Comment),
		makeToken("true", Boolean),
		makeToken("false", Boolean),
		makeToken("", EOL),
		makeToken("yes", Boolean),
		makeToken("no", Boolean),
		makeToken("", EOL),
		makeToken("on", Boolean),
		makeToken("off", Boolean),
		makeToken("", EOL),
		makeToken("scan strings", Comment),
		makeToken("double quoted string", String),
		makeToken("", EOL),
		makeToken("single quoted string", String),
		makeToken("", EOL),
		makeToken("scan variables", Comment),
		makeToken("env", EnvVar),
		makeToken("", EOL),
		makeToken("local", LocalVar),
		makeToken("", EOL),
		makeToken("scan dates", Comment),
		makeToken("2021-04-18", Date),
		makeToken("", Comma),
		makeToken("2021-04-18T19:16:45.321Z", DateTime),
		makeToken("", Comma),
		makeToken("2021-04-18 19:16:45.321+02:00", DateTime),
		makeToken("", EOL),
		makeToken("19:16:45", Time),
		makeToken("", Comma),
		makeToken("19:16:45.123", Time),
		makeToken("", EOL),
		makeToken("scan macros", Comment),
		makeToken("include", Macro),
		makeToken("", EOL),
		makeToken("long comment", Comment),
		makeToken("very long /*nested long comment*/ and more", Comment),
		makeToken("let", Let),
		makeToken("return", Ret),
		makeToken("if", If),
		makeToken("for", For),
		makeToken("while", While),
		makeToken("else", Else),
		makeToken("", EOL),
		makeToken("invalid comment", Invalid),
	}
	for i := 0; ; i++ {
		tok := sc.Scan()
		if tok.Type == EOF {
			break
		}
		if i >= len(tokens) {
			t.Errorf("too many tokens! want %d, got %d", len(tokens), i)
			return
		}
		if tok.Type == Invalid && tokens[i].Type != Invalid {
			t.Errorf("%s: invalid token! expected %s", tok, tokens[i])
			return
		}
		if !tokens[i].Equal(tok) {
			t.Errorf("token mismatched! want %s, got %s", tokens[i], tok)
			return
		}
	}
}

func makeToken(str string, kind rune) Token {
	return Token{
		Input: str,
		Type:  kind,
	}
}
