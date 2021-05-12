package fig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDocument(t *testing.T) {
	r, err := os.Open(filepath.Join("testdata", "main.fig"))
	if err != nil {
		t.Fatalf("fail to open sample config file: %s", err)
		return
	}
	defer r.Close()
	doc, err := ParseDocument(r)
	if err != nil {
		t.Errorf("unexpected error parsing document: %s", err)
		return
	}
	doc.DefineInt("int", 5)

	checkString(t, doc)
	checkFloat(t, doc)
	checkBool(t, doc)
	checkInt(t, doc)
	checkTime(t, doc)
}

func checkTime(t *testing.T, doc *Document) {
	t.Helper()
	data := []struct {
		Key  []string
		Want time.Time
	}{
		{Key: []string{"times", "date"}, Want: time.Date(2021, 5, 7, 0, 0, 0, 0, time.UTC)},
		{Key: []string{"times", "dt1"}, Want: time.Date(2021, 5, 7, 19, 6, 47, 0, time.UTC)},
		{Key: []string{"times", "dt2"}, Want: time.Date(2021, 5, 7, 19, 6, 47, 0, time.UTC)},
		{Key: []string{"times", "dt3"}, Want: time.Date(2021, 5, 7, 19, 6, 47, 0, time.UTC)},
		{Key: []string{"times", "dt4"}, Want: time.Date(2021, 5, 7, 19, 6, 47, 123_000_000, time.UTC)},
		{Key: []string{"times", "dt5"}, Want: time.Date(2021, 5, 7, 19, 6, 47, 123_456_000, time.UTC)},
		{Key: []string{"times", "dt6"}, Want: time.Date(2021, 5, 7, 19, 6, 47, 123_456_789, time.UTC)},
		{Key: []string{"times", "time1"}, Want: time.Date(0, 1, 1, 19, 6, 47, 0, time.UTC)},
		{Key: []string{"times", "time2"}, Want: time.Date(0, 1, 1, 19, 6, 47, 123_000_000, time.UTC)},
		{Key: []string{"times", "time3"}, Want: time.Date(0, 1, 1, 19, 6, 47, 123_456_000, time.UTC)},
		{Key: []string{"times", "time4"}, Want: time.Date(0, 1, 1, 19, 6, 47, 123_456_789, time.UTC)},
	}
	for _, d := range data {
		got, err := doc.Time(d.Key...)
		if err != nil {
			key := strings.Join(d.Key, "/")
			t.Errorf("fail to find %s: %s", key, err)
			continue
		}
		if !d.Want.Equal(got) {
			key := strings.Join(d.Key, "/")
			t.Errorf("%s: time mismatched! want %s, got %s", key, d.Want, got)
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
		{Key: []string{"expr", "div"}, Want: 5},
		{Key: []string{"unit", "si"}, Want: 1000},
		{Key: []string{"unit", "iec"}, Want: 1024},
		{Key: []string{"functions", "max"}, Want: 9},
		{Key: []string{"functions", "min"}, Want: -1},
		{Key: []string{"variables", "local", "var"}, Want: 100},
		{Key: []string{"variables", "local", "int"}, Want: 500},
		{Key: []string{"variables", "local", "expr"}, Want: 5},
		{Key: []string{"variables", "mix", "mod"}, Want: 0},
	}
	for _, d := range data {
		got, err := doc.Int(d.Key...)
		if err != nil {
			key := strings.Join(d.Key, "/")
			t.Errorf("fail to find %s: %s", key, err)
			continue
		}
		if d.Want != got {
			key := strings.Join(d.Key, "/")
			t.Errorf("%s: integers mismatched! want %d, got %d", key, d.Want, got)
		}
	}
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
			key := strings.Join(d.Key, "/")
			t.Errorf("fail to find %s: %s", key, err)
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
		{Key: []string{"index"}, Want: "string"},
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
			key := strings.Join(d.Key, "/")
			t.Errorf("fail to find %s: %s", key, err)
			continue
		}
		if d.Want != strings.ReplaceAll(got, "\r\n", "\n") {
			key := strings.Join(d.Key, "/")
			t.Errorf("%s: strings mismatched! want %s, got %s", key, d.Want, got)
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
			key := strings.Join(d.Key, "/")
			t.Errorf("fail to find %s: %s", key, err)
			continue
		}
		if d.Want != got {
			key := strings.Join(d.Key, "/")
			t.Errorf("%s: floats mismatched! want %f, got %f", key, d.Want, got)
		}
	}
}
