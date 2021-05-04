package fig

import (
	"strings"
	"testing"
)

const sample = `
bool  = true
str1  = literal
str2  = "quoted"
int   = 100
float = 1.2

expr {
  int = 100 + 1
  neg = -100
}

compare {
	lt  = 100 < 1_000
	gt  = 100 > 1_000
	eq  = $lt == $gt
	ne  = $lt != $gt
	not = !$eq
}

unit {
  si  = 1K
  iec = 1k
}

variables {
	local {
		var  = 100
		int  = $int * 5 # will get the int defined at top level of document
		expr = ($int / $var) # will get the int defined in the current object
	}
	global {
		int = @int
	}

	heredoc = <<DOC
	the quick brown fox
	jumps over
	the lazy dog
	DOC
}

variables mix {
	expr = $int - @int
}

functions {
	sqrt  = sqrt(4)
	upper = upper($str1)
	lower = lower($upper)
	max   = max(1, 9, 4, 7)
	min   = min(1, 1, 0, -1)
	replace = $lower != $upper ? replace("foobar", "oo", "OO") : ""
}
`

func TestDocument(t *testing.T) {
	doc, err := ParseDocument(strings.NewReader(sample))
	if err != nil {
		t.Errorf("unexpected error parsing document: %s", err)
		return
	}
	doc.SetInt("int", 5)

	checkString(t, doc)
	checkFloat(t, doc)
	checkBool(t, doc)
	checkInt(t, doc)
}

func checkBool(t *testing.T, doc *Document) {
	t.Helper()
	data := []struct {
		Key  []string
		Want bool
	}{
		{Key: []string{"bool"}, Want: true},
		{Key: []string{"compare", "lt"}, Want: true},
		{Key: []string{"compare", "gt"}, Want: false},
		{Key: []string{"compare", "eq"}, Want: false},
		{Key: []string{"compare", "ne"}, Want: true},
		{Key: []string{"compare", "not"}, Want: true},
	}
	for _, d := range data {
		got, err := doc.Bool(d.Key...)
		if err != nil {
			t.Errorf("fail to find %s: %s", d.Key, err)
			continue
		}
		if d.Want != got {
			key := strings.Join(d.Key, "/")
			t.Errorf("%s: bool mismatched! want %t, got %t", key, d.Want, got)
		}
	}
}

func checkString(t *testing.T, doc *Document) {
	t.Helper()
	data := []struct {
		Key  []string
		Want string
	}{
		{Key: []string{"str1"}, Want: "literal"},
		{Key: []string{"str2"}, Want: "quoted"},
		{Key: []string{"functions", "upper"}, Want: "LITERAL"},
		{Key: []string{"functions", "lower"}, Want: "literal"},
		{Key: []string{"functions", "replace"}, Want: "fOObar"},
		{Key: []string{"variables", "heredoc"}, Want: "the quick brown fox\njumps over\nthe lazy dog"},
	}
	for _, d := range data {
		got, err := doc.Text(d.Key...)
		if err != nil {
			t.Errorf("fail to find %s: %s", d.Key, err)
			continue
		}
		if d.Want != strings.ReplaceAll(got, "\r\n", "\n") {
			key := strings.Join(d.Key, "/")
			t.Errorf("%s: strings mismatched! want %s, got %s", key, d.Want, got)
		}
	}
}

func checkInt(t *testing.T, doc *Document) {
	t.Helper()
	data := []struct {
		Key  []string
		Want int64
	}{
		{Key: []string{"int"}, Want: 100},
		{Key: []string{"expr", "int"}, Want: 101},
		{Key: []string{"expr", "neg"}, Want: -100},
		{Key: []string{"unit", "si"}, Want: 1000},
		{Key: []string{"unit", "iec"}, Want: 1024},
		{Key: []string{"variables", "local", "int"}, Want: 500},
		{Key: []string{"variables", "local", "expr"}, Want: 5},
		{Key: []string{"functions", "max"}, Want: 9},
		{Key: []string{"functions", "min"}, Want: -1},
	}
	for _, d := range data {
		got, err := doc.Int(d.Key...)
		if err != nil {
			t.Errorf("fail to find %s: %s", d.Key, err)
			continue
		}
		if d.Want != got {
			key := strings.Join(d.Key, "/")
			t.Errorf("%s: integers mismatched! want %d, got %d", key, d.Want, got)
		}
	}
}

func checkFloat(t *testing.T, doc *Document) {
	t.Helper()
	data := []struct {
		Key  []string
		Want float64
	}{
		{Key: []string{"float"}, Want: 1.2},
		{Key: []string{"functions", "sqrt"}, Want: 2},
	}
	for _, d := range data {
		got, err := doc.Float(d.Key...)
		if err != nil {
			t.Errorf("fail to find %s: %s", d.Key, err)
			continue
		}
		if d.Want != got {
			key := strings.Join(d.Key, "/")
			t.Errorf("%s: floats mismatched! want %f, got %f", key, d.Want, got)
		}
	}
}
