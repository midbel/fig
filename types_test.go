package fig

import (
	"errors"
	"testing"
	"time"
)

type ValueTestCase struct {
	left  Value
	right Value
	want  Value
	err   error
}

func TestValuePower(t *testing.T) {
	data := []ValueTestCase{
		{
			left:  makeInt(2),
			right: makeInt(2),
			want:  makeInt(4),
		},
		{
			left:  makeInt(2),
			right: makeDouble(3.0),
			want:  makeDouble(8.0),
		},
		{
			left:  makeInt(30),
			right: makeMoment(time.Date(2021, 5, 11, 19, 5, 41, 0, time.UTC)),
			err:   ErrIncompatible,
		},
		{
			left:  makeInt(2),
			right: makeSlice([]Value{makeInt(2), makeDouble(4.0)}),
			want:  makeSlice([]Value{makeInt(4), makeDouble(16.0)}),
		},
		{
			left:  makeDouble(2),
			right: makeSlice([]Value{makeInt(2), makeDouble(4.0)}),
			want:  makeSlice([]Value{makeDouble(4), makeDouble(16.0)}),
		},
		{
			left:  makeSlice([]Value{makeInt(2), makeDouble(4.0)}),
			right: makeInt(2),
			want:  makeSlice([]Value{makeInt(4), makeDouble(16.0)}),
		},
		{
			left:  makeSlice([]Value{makeInt(2), makeDouble(4.0)}),
			right: makeDouble(2),
			want:  makeSlice([]Value{makeDouble(4), makeDouble(16.0)}),
		},
		{
			left:  makeSlice([]Value{makeInt(2), makeDouble(4.0)}),
			right: makeSlice([]Value{makeInt(2), makeDouble(2.0)}),
			want:  makeSlice([]Value{makeInt(4), makeDouble(16.0)}),
		},
		{
			left:  makeInt(2),
			right: makeText("hello"),
			err:   ErrIncompatible,
		},
		{
			left:  makeInt(1),
			right: makeBool(false),
			err:   ErrIncompatible,
		},
	}
	for _, d := range data {
		got, err := d.left.power(d.right)
		checkValueTestCase(t, d, got, err)
	}
}

func TestValueDivide(t *testing.T) {
	data := []ValueTestCase{
		{
			left:  makeInt(10),
			right: makeInt(2),
			want:  makeInt(5),
		},
		{
			left:  makeInt(10),
			right: makeInt(0),
			err:   ErrZeroDiv,
		},
		{
			left:  makeInt(2),
			right: makeDouble(2.0),
			want:  makeDouble(1.0),
		},
		{
			left:  makeDouble(2),
			right: makeDouble(0),
			err: ErrIncompatible,
		},
		{
			left:  makeInt(30),
			right: makeMoment(time.Date(2021, 5, 11, 19, 5, 41, 0, time.UTC)),
			err:   ErrIncompatible,
		},
		{
			left:  makeInt(10),
			right: makeSlice([]Value{makeInt(2), makeDouble(2.0)}),
			want:  makeSlice([]Value{makeInt(5), makeDouble(5.0)}),
		},
		{
			left:  makeDouble(10),
			right: makeSlice([]Value{makeInt(2), makeDouble(2.0)}),
			want:  makeSlice([]Value{makeDouble(5), makeDouble(5.0)}),
		},
		{
			left:  makeSlice([]Value{makeInt(50), makeDouble(200.0)}),
			right: makeInt(10),
			want:  makeSlice([]Value{makeInt(5), makeDouble(20.0)}),
		},
		{
			left:  makeSlice([]Value{makeInt(50), makeDouble(200.0)}),
			right: makeSlice([]Value{makeInt(5), makeDouble(20.0)}),
			want:  makeSlice([]Value{makeInt(10), makeDouble(10.0)}),
		},
		{
			left:  makeSlice([]Value{makeInt(50), makeDouble(200.0)}),
			right: makeSlice([]Value{makeInt(0), makeDouble(20.0)}),
			err:   ErrZeroDiv,
		},
		{
			left:  makeSlice([]Value{makeInt(50), makeDouble(200.0)}),
			right: makeSlice([]Value{makeDouble(20.0)}),
			err:   ErrIncompatible,
		},
		{
			left:  makeInt(2),
			right: makeText("hello"),
			err:   ErrIncompatible,
		},
		{
			left:  makeInt(0),
			right: makeBool(false),
			err:   ErrIncompatible,
		},
	}
	for _, d := range data {
		got, err := d.left.divide(d.right)
		checkValueTestCase(t, d, got, err)
	}
}

func TestValueModulo(t *testing.T) {
	data := []ValueTestCase{
		{
			left:  makeInt(10),
			right: makeInt(2),
			want:  makeInt(0),
		},
		{
			left:  makeDouble(10),
			right: makeDouble(2),
			want:  makeDouble(0),
		},
		{
			left:  makeInt(10),
			right: makeInt(3),
			want:  makeInt(1),
		},
		{
			left:  makeInt(10),
			right: makeInt(0),
			err:   ErrZeroDiv,
		},
		{
			left:  makeDouble(10),
			right: makeInt(0),
			err:   ErrZeroDiv,
		},
		{
			left:  makeInt(2),
			right: makeDouble(2.0),
			want:  makeDouble(0.0),
		},
		{
			left:  makeDouble(2.0),
			right: makeDouble(2.0),
			want:  makeDouble(0.0),
		},
		{
			left:  makeInt(30),
			right: makeMoment(time.Date(2021, 5, 11, 19, 5, 41, 0, time.UTC)),
			err:   ErrIncompatible,
		},
		{
			left:  makeDouble(30),
			right: makeMoment(time.Date(2021, 5, 11, 19, 5, 41, 0, time.UTC)),
			err:   ErrIncompatible,
		},
		{
			left:  makeInt(10),
			right: makeSlice([]Value{makeInt(2), makeInt(3), makeDouble(3.0)}),
			want:  makeSlice([]Value{makeInt(0), makeInt(1), makeDouble(1.0)}),
		},
		{
			left:  makeSlice([]Value{makeInt(10), makeInt(3), makeDouble(3.0)}),
			right: makeInt(2),
			want:  makeSlice([]Value{makeInt(0), makeInt(1), makeDouble(1.0)}),
		},
		{
			left:  makeDouble(10),
			right: makeSlice([]Value{makeInt(2), makeDouble(3), makeDouble(3.0)}),
			want:  makeSlice([]Value{makeDouble(0), makeDouble(1), makeDouble(1.0)}),
		},
		{
			left:  makeSlice([]Value{makeInt(10), makeDouble(3), makeDouble(3.0)}),
			right: makeDouble(2),
			want:  makeSlice([]Value{makeDouble(0), makeDouble(1), makeDouble(1.0)}),
		},
		{
			left:  makeSlice([]Value{makeInt(10), makeDouble(3), makeDouble(3.0)}),
			right: makeSlice([]Value{makeInt(5), makeInt(2), makeDouble(2)}),
			want:  makeSlice([]Value{makeInt(0), makeDouble(1), makeDouble(1.0)}),
		},
		{
			left:  makeSlice([]Value{makeInt(10), makeDouble(3), makeDouble(3.0)}),
			right: makeSlice([]Value{makeInt(0), makeInt(2), makeDouble(2)}),
			err:   ErrZeroDiv,
		},
		{
			left:  makeInt(2),
			right: makeText("hello"),
			err:   ErrIncompatible,
		},
		{
			left:  makeDouble(2),
			right: makeText("hello"),
			err:   ErrIncompatible,
		},
		{
			left:  makeInt(0),
			right: makeBool(false),
			err:   ErrIncompatible,
		},
	}
	for _, d := range data {
		got, err := d.left.modulo(d.right)
		checkValueTestCase(t, d, got, err)
	}
}

func TestValueMultiply(t *testing.T) {
	data := []ValueTestCase{
		{
			left:  makeInt(10),
			right: makeInt(100),
			want:  makeInt(1000),
		},
		{
			left:  makeInt(2),
			right: makeDouble(2.5),
			want:  makeDouble(5.0),
		},
		{
			left:  makeDouble(2.5),
			right: makeInt(2),
			want:  makeDouble(5.0),
		},
		{
			left:  makeDouble(2),
			right: makeDouble(2.5),
			want:  makeDouble(5.0),
		},
		{
			left:  makeInt(30),
			right: makeMoment(time.Date(2021, 5, 11, 19, 5, 41, 0, time.UTC)),
			err:   ErrIncompatible,
		},
		{
			left:  makeDouble(30),
			right: makeMoment(time.Date(2021, 5, 11, 19, 5, 41, 0, time.UTC)),
			err:   ErrIncompatible,
		},
		{
			left:  makeInt(5),
			right: makeSlice([]Value{makeInt(5), makeDouble(200.5)}),
			want:  makeSlice([]Value{makeInt(25), makeDouble(1002.5)}),
		},
		{
			left:  makeDouble(5),
			right: makeSlice([]Value{makeInt(5), makeDouble(200.5)}),
			want:  makeSlice([]Value{makeDouble(25), makeDouble(1002.5)}),
		},
		{
			left:  makeSlice([]Value{makeInt(5), makeDouble(200.5)}),
			right: makeInt(5),
			want:  makeSlice([]Value{makeInt(25), makeDouble(1002.5)}),
		},
		{
			left:  makeSlice([]Value{makeInt(5), makeDouble(200.5)}),
			right: makeDouble(5),
			want:  makeSlice([]Value{makeDouble(25), makeDouble(1002.5)}),
		},
		{
			left:  makeSlice([]Value{makeInt(5), makeDouble(200.5)}),
			right: makeSlice([]Value{makeInt(5), makeDouble(5)}),
			want:  makeSlice([]Value{makeInt(25), makeDouble(1002.5)}),
		},
		{
			left:  makeInt(2),
			right: makeText("hello"),
			want:  makeText("hellohello"),
		},
		{
			left:  makeDouble(2),
			right: makeText("hello"),
			err:   ErrIncompatible,
		},
		{
			left:  makeInt(0),
			right: makeBool(false),
			err:   ErrIncompatible,
		},
	}
	for _, d := range data {
		got, err := d.left.multiply(d.right)
		checkValueTestCase(t, d, got, err)
	}
}

func TestValueSubtract(t *testing.T) {
	data := []ValueTestCase{
		{
			left:  makeInt(100),
			right: makeInt(100),
			want:  makeInt(0),
		},
		{
			left:  makeDouble(100.0),
			right: makeDouble(100.0),
			want:  makeDouble(0),
		},
		{
			left:  makeInt(100),
			right: makeDouble(99.5),
			want:  makeDouble(0.5),
		},
		{
			left:  makeDouble(100.5),
			right: makeInt(100),
			want:  makeDouble(0.5),
		},
		{
			left:  makeInt(30),
			right: makeMoment(time.Date(2021, 5, 11, 19, 5, 41, 0, time.UTC)),
			want:  makeMoment(time.Date(2021, 5, 11, 19, 5, 11, 0, time.UTC)),
		},
		{
			left:  makeDouble(30),
			right: makeMoment(time.Date(2021, 5, 11, 19, 5, 41, 0, time.UTC)),
			err:   ErrIncompatible,
		},
		{
			left:  makeInt(100),
			right: makeSlice([]Value{makeInt(100), makeDouble(200.5)}),
			want:  makeSlice([]Value{makeInt(0), makeDouble(-100.5)}),
		},
		{
			left:  makeDouble(100.0),
			right: makeSlice([]Value{makeInt(100), makeDouble(200.5)}),
			want:  makeSlice([]Value{makeDouble(0), makeDouble(-100.5)}),
		},
		{
			left:  makeSlice([]Value{makeInt(100), makeDouble(200.5)}),
			right: makeDouble(100.0),
			want:  makeSlice([]Value{makeDouble(0), makeDouble(100.5)}),
		},
		{
			left:  makeSlice([]Value{makeInt(100), makeDouble(200.5)}),
			right: makeInt(100),
			want:  makeSlice([]Value{makeInt(0), makeDouble(100.5)}),
		},
		{
			left:  makeSlice([]Value{makeInt(100), makeDouble(200.5)}),
			right: makeSlice([]Value{makeInt(100), makeDouble(100.5)}),
			want:  makeSlice([]Value{makeInt(0), makeDouble(100.0)}),
		},
		{
			left:  makeInt(0),
			right: makeText(""),
			err:   ErrIncompatible,
		},
		{
			left:  makeInt(0),
			right: makeBool(false),
			err:   ErrIncompatible,
		},
	}
	for _, d := range data {
		got, err := d.left.subtract(d.right)
		checkValueTestCase(t, d, got, err)
	}
}

func TestValueAdd(t *testing.T) {
	data := []ValueTestCase{
		{
			left:  makeInt(100),
			right: makeInt(100),
			want:  makeInt(200),
		},
		{
			left:  makeDouble(100.5),
			right: makeDouble(100.5),
			want:  makeDouble(201.0),
		},
		{
			left:  makeInt(100),
			right: makeDouble(100.5),
			want:  makeDouble(200.5),
		},
		{
			left:  makeDouble(100.5),
			right: makeInt(100),
			want:  makeDouble(200.5),
		},
		{
			left:  makeInt(100),
			right: makeText("+200"),
			want:  makeText("100+200"),
		},
		{
			left:  makeInt(30),
			right: makeMoment(time.Date(2021, 5, 11, 19, 5, 11, 0, time.UTC)),
			want:  makeMoment(time.Date(2021, 5, 11, 19, 5, 41, 0, time.UTC)),
		},
		{
			left:  makeDouble(30),
			right: makeMoment(time.Date(2021, 5, 11, 19, 5, 11, 0, time.UTC)),
			err:   ErrIncompatible,
		},
		{
			left:  makeInt(100),
			right: makeSlice([]Value{makeInt(100), makeDouble(200.5)}),
			want:  makeSlice([]Value{makeInt(200), makeDouble(300.5)}),
		},
		{
			left:  makeDouble(100.0),
			right: makeSlice([]Value{makeInt(100), makeDouble(200.5)}),
			want:  makeSlice([]Value{makeDouble(200), makeDouble(300.5)}),
		},
		{
			left:  makeSlice([]Value{makeInt(10), makeDouble(10.5)}),
			right: makeSlice([]Value{makeInt(10), makeDouble(10.5)}),
			want:  makeSlice([]Value{makeInt(20), makeDouble(21.0)}),
		},
		{
			left:  makeSlice([]Value{makeInt(10), makeDouble(10.5)}),
			right: makeInt(10),
			want:  makeSlice([]Value{makeInt(20), makeDouble(20.5)}),
		},
		{
			left:  makeSlice([]Value{makeInt(10), makeDouble(10.5)}),
			right: makeDouble(10),
			want:  makeSlice([]Value{makeDouble(20), makeDouble(20.5)}),
		},
		{
			left:  makeSlice([]Value{makeInt(10), makeDouble(10.5)}),
			right: makeSlice([]Value{makeInt(10)}),
			err:   ErrIncompatible,
		},
		{
			left:  makeInt(0),
			right: makeBool(false),
			err:   ErrIncompatible,
		},
	}
	for _, d := range data {
		got, err := d.left.add(d.right)
		checkValueTestCase(t, d, got, err)
	}
}

func checkValueTestCase(t *testing.T, tvc ValueTestCase, got Value, err error) {
	if tvc.err != nil {
		if err == nil {
			t.Errorf("values: expected error %s but operation succeed", tvc.err)
		} else if !errors.Is(err, tvc.err) {
			t.Errorf("values: errors mismatched! want %v, got %v", tvc.err, err)
		}
		return
	}
	if err != nil {
		t.Errorf("values: unexpected error %s", err)
		return
	}
	testResultValue(t, tvc.want, got)
}

func TestBool(t *testing.T) {
	t.Run("and", testBoolAnd)
	t.Run("or", testBoolOr)
	t.Run("not", testBoolNot)
	t.Run("cmp", testBoolCmp)
	t.Run("arithmetic", testBoolArithmetic)
}

func testBoolOr(t *testing.T) {
	var (
		tru  = makeBool(true)
		fals = makeBool(false)
	)
	data := []struct {
		left  Value
		right Value
		want  bool
		err   error
	}{
		{
			left:  tru,
			right: tru,
			want:  true,
		},
		{
			left:  tru,
			right: fals,
			want:  true,
		},
		{
			left:  fals,
			right: tru,
			want:  true,
		},
		{
			left:  fals,
			right: fals,
			want:  false,
		},
	}
	for _, d := range data {
		got, err := d.left.or(d.right)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			continue
		}
		testBoolResult(t, d.want, got)
	}
}

func testBoolAnd(t *testing.T) {
	var (
		tru  = makeBool(true)
		fals = makeBool(false)
	)
	data := []struct {
		left  Value
		right Value
		want  bool
	}{
		{
			left:  tru,
			right: tru,
			want:  true,
		},
		{
			left:  tru,
			right: fals,
			want:  false,
		},
		{
			left:  fals,
			right: tru,
			want:  false,
		},
		{
			left:  fals,
			right: fals,
			want:  false,
		},
	}
	for _, d := range data {
		got, err := d.left.and(d.right)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			continue
		}
		testBoolResult(t, d.want, got)
	}
}

func testBoolNot(t *testing.T) {
	var (
		tru  = makeBool(true)
		fals = makeBool(false)
		got  Value
		err  error
	)
	got, err = tru.not()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}
	testBoolResult(t, false, got)
	got, err = fals.not()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}
	testBoolResult(t, true, got)
}

func testBoolCmp(t *testing.T) {
	var (
		tru  = makeBool(true)
		fals = makeBool(false)
	)
	data := []struct {
		left  Value
		right Value
		want  int
	}{
		{
			left:  tru,
			right: tru,
			want:  0,
		},
		{
			left:  tru,
			right: fals,
			want:  1,
		},
		{
			left:  fals,
			right: tru,
			want:  -1,
		},
		{
			left:  fals,
			right: fals,
			want:  0,
		},
	}
	for _, d := range data {
		got, err := d.left.compare(d.right)
		if err != nil {
			t.Errorf("cmp: unexpected error: %s", err)
			continue
		}
		if d.want != got {
			t.Errorf("results mismatched! want %d, got %d", d.want, got)
		}
	}
}

func testBoolArithmetic(t *testing.T) {
	var (
		tru  = makeBool(true)
		fals = makeBool(false)
		err  error
		errs []error
	)
	_, err = tru.add(fals)
	errs = append(errs, err)
	_, err = tru.subtract(fals)
	errs = append(errs, err)
	_, err = tru.multiply(fals)
	errs = append(errs, err)
	_, err = tru.divide(fals)
	errs = append(errs, err)
	_, err = tru.power(fals)
	errs = append(errs, err)
	_, err = tru.modulo(fals)
	errs = append(errs, err)
	_, err = tru.reverse()
	errs = append(errs, err)
	_, err = tru.leftshift(fals)
	errs = append(errs, err)
	_, err = tru.rightshift(fals)
	errs = append(errs, err)
	_, err = tru.binand(fals)
	errs = append(errs, err)
	_, err = tru.binor(fals)
	errs = append(errs, err)
	_, err = tru.binxor(fals)
	errs = append(errs, err)
	_, err = tru.binnot()
	errs = append(errs, err)
	_, err = tru.at(makeInt(0))
	errs = append(errs, err)
	for _, e := range errs {
		if !errors.Is(e, ErrUnsupported) {
			t.Errorf("expected %v, got %v", ErrUnsupported, e)
		}
	}
}

func testMomentResult(t *testing.T, want time.Time, got Value) {
	t.Helper()
	g, ok := got.(Moment)
	if !ok {
		t.Errorf("moment: unexpected type! want %T, got %T", g, got)
		return
	}
	if !want.Equal(g.inner) {
		t.Errorf("moment: mismatched! want %s, got %s", want, g.inner)
	}
}

func testResultValue(t *testing.T, want, got Value) {
	switch w := want.(type) {
	case Int:
		testIntResult(t, w.inner, got)
	case Double:
		testDoubleResult(t, w.inner, got)
	case Text:
		testTextResult(t, w.inner, got)
	case Moment:
		testMomentResult(t, w.inner, got)
	case Slice:
		testSliceResult(t, w.inner, got)
	default:
		t.Errorf("unexpected type: %T", w)
	}
}

func testSliceResult(t *testing.T, want []Value, got Value) {
	t.Helper()
	g, ok := got.(Slice)
	if !ok {
		t.Errorf("slice: unexpected type! want %T, got %T", g, got)
		return
	}
	if len(want) != len(g.inner) {
		t.Errorf("slice: length mismatched! want %d, got %d", len(want), len(g.inner))
	}
	for i := range want {
		testResultValue(t, want[i], g.inner[i])
	}
}

func testTextResult(t *testing.T, want string, got Value) {
	t.Helper()
	g, ok := got.(Text)
	if !ok {
		t.Errorf("text: unexpected type! want %T, got %T", g, got)
		return
	}
	if g.inner != want {
		t.Errorf("text: mismatched! want %s, got %s", want, g.inner)
	}
}

func testDoubleResult(t *testing.T, want float64, got Value) {
	t.Helper()
	g, ok := got.(Double)
	if !ok {
		t.Errorf("doubles: unexpected type! want %T, got %T", g, got)
		return
	}
	if g.inner != want {
		t.Errorf("doubles: mismatched! want %f, got %f", want, g.inner)
	}
}

func testIntResult(t *testing.T, want int64, got Value) {
	t.Helper()
	g, ok := got.(Int)
	if !ok {
		t.Errorf("integers: unexpected type! want %T, got %T", g, got)
		return
	}
	if g.inner != want {
		t.Errorf("integers: mismatched! want %d, got %d", want, g.inner)
	}
}

func testBoolResult(t *testing.T, want bool, got Value) {
	t.Helper()
	g, ok := got.(Bool)
	if !ok {
		t.Errorf("booleans: unexpected type! want %T, got %T", g, got)
		return
	}
	if g.inner != want {
		t.Errorf("booleans: mismatched! want %t, got %t", want, g.inner)
	}
}
