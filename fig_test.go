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

unit {
  si  = 1K
  iec = 1k
}
`

func TestDocument(t *testing.T) {
	doc, err := ParseDocument(strings.NewReader(sample))
	if err != nil {
		t.Errorf("unexpected error parsing document: %s", err)
		return
	}
	checkString(t, doc)
	checkInt(t, doc)
	checkFloat(t, doc)
}

func checkInt(t *testing.T, doc *Document) {
	t.Helper()
	data := []struct {
		Key  []string
		Want int64
	}{
		{
			Key:  []string{"int"},
			Want: 100,
		},
		{
			Key:  []string{"expr", "int"},
			Want: 101,
		},
		{
			Key:  []string{"expr", "neg"},
			Want: -100,
		},
		{
			Key:  []string{"unit", "si"},
			Want: 1000,
		},
		{
			Key:  []string{"unit", "iec"},
			Want: 1024,
		},
	}
	for _, d := range data {
		got, err := doc.Int(d.Key...)
		if err != nil {
			t.Errorf("fail to find %s: %s", d.Key, err)
			continue
		}
		if d.Want != got {
			t.Errorf("integers mismatched! want %d, got %d", d.Want, got)
		}
	}
}

func checkString(t *testing.T, doc *Document) {
	t.Helper()
	data := []struct {
		Key  []string
		Want string
	}{
		{
			Key:  []string{"str1"},
			Want: "literal",
		},
		{
			Key:  []string{"str2"},
			Want: "quoted",
		},
	}
	for _, d := range data {
		got, err := doc.Text(d.Key...)
		if err != nil {
			t.Errorf("fail to find %s: %s", d.Key, err)
			continue
		}
		if d.Want != got {
			t.Errorf("strings mismatched! want %s, got %s", d.Want, got)
		}
	}
}

func checkFloat(t *testing.T, doc *Document) {
	t.Helper()
	data := []struct {
		Key  []string
		Want float64
	}{
		{
			Key:  []string{"float"},
			Want: 1.2,
		},
	}
	for _, d := range data {
		got, err := doc.Float(d.Key...)
		if err != nil {
			t.Errorf("fail to find %s: %s", d.Key, err)
			continue
		}
		if d.Want != got {
			t.Errorf("floats mismatched! want %f, got %f", d.Want, got)
		}
	}
}
