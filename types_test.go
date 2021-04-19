package fig

import (
	"errors"
	"math"
	"testing"
	"time"
)

type ValueCase struct {
	Left  Value
	Right Value
	Want  Value
	Error error
}

func TestValue(t *testing.T) {
	t.Run("add", testValueAdd)
	t.Run("subtract", testValueSubtract)
}

func testValueSubtract(t *testing.T) {
	data := []ValueCase{
		{
			Left:  makeInt(1),
			Right: makeInt(1),
			Want:  makeInt(0),
		},
		{
			Left:  makeInt(2),
			Right: makeDouble(1.4),
			Want:  makeInt(1),
		},
		{
			Left:  makeDouble(1.4),
			Right: makeInt(1),
			Want:  makeDouble(0.4),
		},
		{
			Left:  makeDouble(1.4),
			Right: makeMoment(time.Now()),
			Error: ErrIncompatible,
		},
		{
			Left:  makeInt(1),
			Right: makeText("foobar"),
			Error: ErrIncompatible,
		},
		{
			Left:  makeDouble(2.1),
			Right: makeDouble(2.1),
			Want:  makeDouble(0),
		},
		{
			Left:  makeDouble(2.1),
			Right: makeBool(false),
			Error: ErrIncompatible,
		},
	}
	for _, d := range data {
		got, err := d.Left.subtract(d.Right)
		checkResult(t, d, got, err)
	}
}

func testValueAdd(t *testing.T) {
	data := []ValueCase{
		{
			Left:  makeInt(1),
			Right: makeInt(1),
			Want:  makeInt(2),
		},
		{
			Left:  makeInt(1),
			Right: makeDouble(1.4),
			Want:  makeInt(2),
		},
		{
			Left:  makeDouble(1.4),
			Right: makeInt(1),
			Want:  makeDouble(2.4),
		},
		{
			Left:  makeDouble(1.4),
			Right: makeMoment(time.Now()),
			Error: ErrIncompatible,
		},
		{
			Left:  makeInt(1),
			Right: makeText("foobar"),
			Error: ErrIncompatible,
		},
		{
			Left:   makeText("foobar"),
			Right:  makeInt(1),
			Error: ErrIncompatible,
		},
		{
			Left:  makeDouble(2.1),
			Right: makeDouble(2.1),
			Want:  makeDouble(4.2),
		},
		{
			Left:  makeDouble(2.1),
			Right: makeBool(false),
			Error: ErrIncompatible,
		},
		{
			Left:  makeText("foo"),
			Right: makeText("bar"),
			Want:  makeText("foobar"),
		},
	}
	for _, d := range data {
		got, err := d.Left.add(d.Right)
		checkResult(t, d, got, err)
	}
}

func checkResult(t *testing.T, d ValueCase, got Value, err error) {
	t.Helper()
	if err != nil && d.Error == nil {
		t.Errorf("unexpected error! %s", err)
		return
	}
	if d.Error != nil {
		if !errors.Is(err, d.Error) {
			t.Errorf("errors mismatched! want %s, got %s", d.Error, err)
		}
		return
	}
	if !checkValue(d.Want, got) {
		t.Errorf("values mismatched! want %v, got %v", d.Want, got)
	}
}

func checkValue(want, got Value) bool {
	switch want := want.(type) {
	case Int:
		got, ok := got.(Int)
		if ok {
			ok = got.inner == want.inner
		}
		return ok
	case Double:
		got, ok := got.(Double)
		if ok {
			ok = math.Abs(got.inner - want.inner) < 0.00001
		}
		return ok
	case Bool:
		got, ok := got.(Bool)
		if ok {
			ok = got.inner == want.inner
		}
		return ok
	case Text:
		got, ok := got.(Text)
		if ok {
			ok = got.inner == want.inner
		}
		return ok
	case Moment:
		got, ok := got.(Moment)
		if ok {
			ok = got.inner.Equal(want.inner)
		}
		return ok
	default:
		return false
	}
}
