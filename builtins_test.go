package fig

import (
	"errors"
	"reflect"
	"testing"
)

func TestBuiltins(t *testing.T) {
	t.Run("typeof", testTypeof)
	t.Run("length", testLength)
	t.Run("first", testFirst)
	t.Run("last", testLast)
	t.Run("all", testAll)
	t.Run("any", testAny)
	t.Run("sequence", testSequence)
	t.Run("avg", testAvg)
}

func testAvg(t *testing.T) {
	var (
		args = makeSlice([]Value{makeInt(10), makeInt(10), makeDouble(10), makeDouble(5), makeDouble(5)})
		env  = EmptyEnv()
		want = 8.0
	)
	env.Define("args", args)
	v, err := avg(env)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}
	got, err := toFloat(v)
	if err != nil {
		t.Errorf("fail to get double from value: %s", err)
		return
	}
	if got != want {
		t.Errorf("results mismatched! want %f, got %f", want, got)
	}
}

func testSequence(t *testing.T) {
	data := []struct {
		First Value
		Last  Value
		Step  Value
		Want  Value
	}{
		{
			First: makeInt(0),
			Last:  makeInt(3),
			Step:  makeInt(1),
			Want:  makeSlice([]Value{makeInt(0), makeInt(1), makeInt(2)}),
		},
		{
			First: makeInt(0),
			Last:  makeInt(6),
			Step:  makeInt(2),
			Want:  makeSlice([]Value{makeInt(0), makeInt(2), makeInt(4)}),
		},
		{
			First: makeInt(8),
			Last:  makeInt(5),
			Step:  makeInt(1),
			Want:  makeSlice([]Value{makeInt(8), makeInt(7), makeInt(6)}),
		},
	}
	for _, d := range data {
		e := EmptyEnv()
		e.Define("first", d.First)
		e.Define("last", d.Last)
		if d.Step != nil {
			e.Define("step", d.Step)
		}
		got, err := sequence(e)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			continue
		}
		if !reflect.DeepEqual(d.Want, got) {
			t.Errorf("results mismatched! want %v, got %v", d.Want, got)
		}
	}
}

func testAll(t *testing.T) {
	data := []struct {
		Value
		Want bool
	}{
		{
			Value: makeSlice([]Value{}),
			Want:  false,
		},
		{
			Value: makeSlice([]Value{makeBool(false)}),
			Want:  false,
		},
		{
			Value: makeSlice([]Value{makeBool(true), makeText("true")}),
			Want:  true,
		},
	}
	for _, d := range data {
		e := EmptyEnv()
		e.Define("args", d.Value)
		v, err := all(e)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			continue
		}
		got := v.isTrue()
		if d.Want != got {
			t.Errorf("results mismatched! want %t, got %t", d.Want, got)
		}
	}
}

func testAny(t *testing.T) {
	data := []struct {
		Value
		Want bool
	}{
		{
			Value: makeSlice([]Value{}),
			Want:  false,
		},
		{
			Value: makeSlice([]Value{makeBool(false)}),
			Want:  false,
		},
		{
			Value: makeSlice([]Value{makeBool(true)}),
			Want:  true,
		},
		{
			Value: makeSlice([]Value{makeBool(true), makeText("true")}),
			Want:  true,
		},
	}
	for _, d := range data {
		e := EmptyEnv()
		e.Define("args", d.Value)
		v, err := any(e)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			continue
		}
		got := v.isTrue()
		if d.Want != got {
			t.Errorf("results mismatched! want %t, got %t", d.Want, got)
		}
	}
}

func testFirst(t *testing.T) {
	data := []struct {
		Value
		Want Value
	}{
		{
			Value: makeSlice([]Value{makeInt(0), makeInt(1)}),
			Want:  makeInt(0),
		},
		{
			Value: makeSlice([]Value{makeText("hello"), makeText("world")}),
			Want:  makeText("hello"),
		},
	}
	for _, d := range data {
		e := EmptyEnv()
		e.Define("arr", d.Value)
		got, err := first(e)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			continue
		}
		cmp, err := d.Want.compare(got)
		if err != nil {
			t.Errorf("fail to compare values: %s", err)
			continue
		}
		if cmp != 0 {
			t.Errorf("results mismatched! want %d, got %d", d.Want, got)
		}
	}
}

func testLast(t *testing.T) {
	data := []struct {
		Value
		Want Value
	}{
		{
			Value: makeSlice([]Value{makeInt(0), makeInt(1)}),
			Want:  makeInt(1),
		},
		{
			Value: makeSlice([]Value{makeText("hello"), makeText("world")}),
			Want:  makeText("world"),
		},
	}
	for _, d := range data {
		e := EmptyEnv()
		e.Define("arr", d.Value)
		got, err := last(e)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			continue
		}
		cmp, err := d.Want.compare(got)
		if err != nil {
			t.Errorf("fail to compare values: %s", err)
			continue
		}
		if cmp != 0 {
			t.Errorf("results mismatched! want %d, got %d", d.Want, got)
		}
	}
}

func testLength(t *testing.T) {
	data := []struct {
		Value
		Want int64
		Err  error
	}{
		{
			Value: makeInt(0),
			Err:   ErrUnsupported,
		},
		{
			Value: makeText("value"),
			Want:  5,
		},
		{
			Value: makeDouble(0),
			Err:   ErrUnsupported,
		},
		{
			Value: makeBool(false),
			Err:   ErrUnsupported,
		},
		{
			Value: makeSlice([]Value{}),
			Want:  0,
		},
		{
			Value: makeSlice([]Value{makeBool(false)}),
			Want:  1,
		},
	}
	for _, d := range data {
		e := EmptyEnv()
		e.Define("obj", d.Value)
		v, err := length(e)
		if d.Err != nil {
			if err == nil {
				t.Errorf("expected error but function succeed!")
			} else if !errors.Is(err, d.Err) {
				t.Errorf("errors mismatched! want %s, got %s", d.Err, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			continue
		}
		got, err := toInt(v)
		if err != nil {
			t.Errorf("fail to get int from value: %s", err)
			continue
		}
		if d.Want != got {
			t.Errorf("results mismatched! want %d, got %d", d.Want, got)
		}
	}
}

func testTypeof(t *testing.T) {
	data := []struct {
		Value
		Want string
		Err  error
	}{
		{
			Value: makeInt(0),
			Want:  "integer",
		},
		{
			Value: makeText(""),
			Want:  "text",
		},
		{
			Value: makeDouble(0),
			Want:  "double",
		},
		{
			Value: makeBool(false),
			Want:  "boolean",
		},
		{
			Value: makeSlice([]Value{}),
			Want:  "array",
		},
	}
	for _, d := range data {
		e := EmptyEnv()
		e.Define("obj", d.Value)
		v, err := typeof(e)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			continue
		}
		got, err := toText(v)
		if err != nil {
			t.Errorf("fail to get text from value: %s", err)
			continue
		}
		if d.Want != got {
			t.Errorf("results mismatched! want %s, got %s", d.Want, got)
		}
	}
}
