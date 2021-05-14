package fig

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"time"
)

var timePattern = []string{
	"2006-01-02T15:04:05",
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05.000Z",
	"2006-01-02T15:04:05.000000Z",
	"2006-01-02T15:04:05.000000000Z",
	"2006-01-02T15:04:05-07:00",
	"2006-01-02T15:04:05.000-07:00",
	"2006-01-02T15:04:05.000000-07:00",
	"2006-01-02T15:04:05.000000000-07:00",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04:05Z",
	"2006-01-02 15:04:05.000Z",
	"2006-01-02 15:04:05.000000Z",
	"2006-01-02 15:04:05.000000000Z",
	"2006-01-02 15:04:05-07:00",
	"2006-01-02 15:04:05.000-07:00",
	"2006-01-02 15:04:05.000000-07:00",
	"2006-01-02 15:04:05.000000000-07:00",
	"2006-01-02",
	"15:04:05",
	"15:04:05.000",
	"15:04:05.000000",
	"15:04:05.000000000",
}

var (
	ErrIncompatible = errors.New("incompatible types")
	ErrUnsupported  = errors.New("unsupported operation")
	ErrZeroDiv      = errors.New("division by zero")
	ErrIndex        = errors.New("index out of range")
)

const (
	scoreLowest int = iota
	scoreBool
	scoreText
	scoreInt
	scoreDouble
	scoreTime
	scoreSlice
)

const epsilon = 1e-9

type Value interface {
	add(Value) (Value, error)
	subtract(Value) (Value, error)
	multiply(Value) (Value, error)
	divide(Value) (Value, error)
	modulo(Value) (Value, error)
	power(Value) (Value, error)
	reverse() (Value, error)

	at(Value) (Value, error)

	not() (Value, error)
	and(Value) (Value, error)
	or(Value) (Value, error)

	compare(Value) (int, error)

	leftshift(Value) (Value, error)
	rightshift(Value) (Value, error)
	binand(Value) (Value, error)
	binor(Value) (Value, error)
	binnot() (Value, error)
	binxor(Value) (Value, error)

	isTrue() bool

	score() int
	toInt() (Value, error)
	toDouble() (Value, error)
	toBool() (Value, error)
	toText() (Value, error)
	toMoment() (Value, error)
}

type Bool struct {
	inner bool
}

func makeBool(b bool) Value {
	return Bool{inner: b}
}

func (b Bool) not() (Value, error) {
	return not(b), nil
}

func (b Bool) and(other Value) (Value, error) {
	return and(b, other), nil
}

func (b Bool) or(other Value) (Value, error) {
	return or(b, other), nil
}

func (b Bool) compare(other Value) (int, error) {
	x, ok := other.(Bool)
	if !ok {
		return -1, ErrIncompatible
	}
	if b.inner == x.inner {
		return 0, nil
	}
	if b.inner {
		return 1, nil
	}
	return -1, nil
}

func (b Bool) isTrue() bool {
	return b.inner
}

func (_ Bool) score() int {
	return scoreBool
}

func (b Bool) toBool() (Value, error) {
	return b, nil
}

func (_ Bool) at(_ Value) (Value, error)         { return nil, ErrUnsupported }
func (_ Bool) add(_ Value) (Value, error)        { return nil, ErrUnsupported }
func (_ Bool) subtract(_ Value) (Value, error)   { return nil, ErrUnsupported }
func (_ Bool) multiply(_ Value) (Value, error)   { return nil, ErrUnsupported }
func (_ Bool) divide(_ Value) (Value, error)     { return nil, ErrUnsupported }
func (_ Bool) modulo(_ Value) (Value, error)     { return nil, ErrUnsupported }
func (_ Bool) power(_ Value) (Value, error)      { return nil, ErrUnsupported }
func (_ Bool) reverse() (Value, error)           { return nil, ErrUnsupported }
func (_ Bool) leftshift(_ Value) (Value, error)  { return nil, ErrUnsupported }
func (_ Bool) rightshift(_ Value) (Value, error) { return nil, ErrUnsupported }
func (_ Bool) binand(_ Value) (Value, error)     { return nil, ErrUnsupported }
func (_ Bool) binor(_ Value) (Value, error)      { return nil, ErrUnsupported }
func (_ Bool) binnot() (Value, error)            { return nil, ErrUnsupported }
func (_ Bool) binxor(_ Value) (Value, error)     { return nil, ErrUnsupported }
func (_ Bool) toInt() (Value, error)             { return nil, ErrIncompatible }
func (_ Bool) toDouble() (Value, error)          { return nil, ErrIncompatible }
func (_ Bool) toText() (Value, error)            { return nil, ErrIncompatible }
func (_ Bool) toMoment() (Value, error)          { return nil, ErrIncompatible }

type Int struct {
	inner int64
}

func makeInt(val int64) Value {
	return Int{inner: val}
}

func (i Int) add(other Value) (Value, error) {
	switch s := other.score(); s {
	case scoreInt:
		x, _ := toInt(other)
		return makeInt(i.inner + x), nil
	case scoreDouble:
		d, _ := i.toDouble()
		return d.add(other)
	case scoreText:
		t, _ := i.toText()
		return t.add(other)
	case scoreTime:
		m, ok := other.(Moment)
		if !ok {
			return nil, ErrIncompatible
		}
		return m.addDuration(time.Duration(i.inner) * time.Second)
	case scoreSlice:
		s, ok := other.(Slice)
		if !ok {
			return nil, ErrIncompatible
		}
		return s.apply(func(curr Value) (Value, error) {
			return i.add(curr)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (i Int) subtract(other Value) (Value, error) {
	switch s := other.score(); s {
	case scoreInt:
		x, _ := toInt(other)
		return makeInt(i.inner - x), nil
	case scoreDouble:
		d, _ := i.toDouble()
		return d.subtract(other)
	case scoreTime:
		m, ok := other.(Moment)
		if !ok {
			return nil, ErrIncompatible
		}
		return m.addDuration(time.Duration(-i.inner) * time.Second)
	case scoreSlice:
		s, ok := other.(Slice)
		if !ok {
			return nil, ErrIncompatible
		}
		return s.apply(func(curr Value) (Value, error) {
			return i.subtract(curr)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (i Int) multiply(other Value) (Value, error) {
	switch s := other.score(); s {
	case scoreInt:
		x, _ := toInt(other)
		return makeInt(i.inner * x), nil
	case scoreDouble:
		d, err := i.toDouble()
		if err != nil {
			return nil, err
		}
		return d.multiply(other)
	case scoreText:
		t, ok := other.(Text)
		if !ok {
			return nil, ErrIncompatible
		}
		return t.multiply(i)
	case scoreSlice:
		s, ok := other.(Slice)
		if !ok {
			return nil, ErrIncompatible
		}
		return s.apply(func(curr Value) (Value, error) {
			return i.multiply(curr)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (i Int) divide(other Value) (Value, error) {
	switch s := other.score(); s {
	case scoreInt:
		x, _ := toInt(other)
		if x == 0 {
			return nil, ErrZeroDiv
		}
		return makeInt(i.inner / x), nil
	case scoreDouble:
		d, _ := i.toDouble()
		return d.divide(other)
	case scoreSlice:
		s, ok := other.(Slice)
		if !ok {
			return nil, ErrIncompatible
		}
		return s.apply(func(curr Value) (Value, error) {
			return i.divide(curr)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (i Int) modulo(other Value) (Value, error) {
	switch s := other.score(); s {
	case scoreInt:
		x, _ := toInt(other)
		if x == 0 {
			return nil, ErrZeroDiv
		}
		return makeInt(i.inner % x), nil
	case scoreDouble:
		d, _ := i.toDouble()
		return d.modulo(other)
	case scoreSlice:
		s, ok := other.(Slice)
		if !ok {
			return nil, ErrIncompatible
		}
		return s.apply(func(curr Value) (Value, error) {
			return i.modulo(curr)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (i Int) power(other Value) (Value, error) {
	switch s := other.score(); s {
	case scoreInt:
		x, _ := toInt(other)
		r := math.Pow(float64(i.inner), float64(x))
		return makeInt(int64(r)), nil
	case scoreDouble:
		d, _ := i.toDouble()
		return d.power(other)
	case scoreSlice:
		s, ok := other.(Slice)
		if !ok {
			return nil, ErrIncompatible
		}
		return s.apply(func(curr Value) (Value, error) {
			return i.power(curr)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (i Int) reverse() (Value, error) {
	return makeInt(-i.inner), nil
}

func (i Int) not() (Value, error) {
	return not(i), nil
}

func (i Int) and(other Value) (Value, error) {
	return and(i, other), nil
}

func (i Int) or(other Value) (Value, error) {
	return or(i, other), nil
}

func (i Int) compare(other Value) (int, error) {
	x, err := toInt(other)
	if err != nil {
		return -1, err
	}
	if i.inner == x {
		return 0, nil
	}
	if i.inner > x {
		return 1, nil
	}
	return -1, nil
}

func (i Int) leftshift(other Value) (Value, error) {
	switch s := other.score(); s {
	case scoreInt:
		x, _ := toInt(other)
		return makeInt(i.inner << x), nil
	case scoreSlice:
		s, ok := other.(Slice)
		if !ok {
			return nil, ErrIncompatible
		}
		return s.apply(func(curr Value) (Value, error) {
			return i.leftshift(curr)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (i Int) rightshift(other Value) (Value, error) {
	switch s := other.score(); s {
	case scoreInt:
		x, _ := toInt(other)
		return makeInt(i.inner >> x), nil
	case scoreSlice:
		s, ok := other.(Slice)
		if !ok {
			return nil, ErrIncompatible
		}
		return s.apply(func(curr Value) (Value, error) {
			return i.rightshift(curr)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (i Int) binand(other Value) (Value, error) {
	switch s := other.score(); s {
	case scoreInt:
		x, _ := toInt(other)
		return makeInt(i.inner & x), nil
	case scoreSlice:
		s, ok := other.(Slice)
		if !ok {
			return nil, ErrIncompatible
		}
		return s.apply(func(curr Value) (Value, error) {
			return i.binand(curr)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (i Int) binor(other Value) (Value, error) {
	switch s := other.score(); s {
	case scoreInt:
		x, _ := toInt(other)
		return makeInt(i.inner | x), nil
	case scoreSlice:
		s, ok := other.(Slice)
		if !ok {
			return nil, ErrIncompatible
		}
		return s.apply(func(curr Value) (Value, error) {
			return i.binor(curr)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (i Int) binxor(other Value) (Value, error) {
	switch s := other.score(); s {
	case scoreInt:
		x, _ := toInt(other)
		return makeInt(i.inner ^ x), nil
	case scoreSlice:
		s, ok := other.(Slice)
		if !ok {
			return nil, ErrIncompatible
		}
		return s.apply(func(curr Value) (Value, error) {
			return i.leftshift(curr)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (i Int) binnot() (Value, error) {
	return makeInt(^i.inner), nil
}

func (i Int) isTrue() bool {
	return i.inner != 0
}

func (_ Int) score() int {
	return scoreInt
}

func (i Int) toInt() (Value, error) {
	return i, nil
}

func (i Int) toDouble() (Value, error) {
	return makeDouble(float64(i.inner)), nil
}

func (i Int) toBool() (Value, error) {
	return makeBool(i.isTrue()), nil
}

func (i Int) toText() (Value, error) {
	return makeText(strconv.FormatInt(i.inner, 10)), nil
}

func (i Int) toMoment() (Value, error) {
	return makeMoment(time.Unix(i.inner, 0)), nil
}

func (_ Int) at(_ Value) (Value, error) { return nil, ErrUnsupported }

type Double struct {
	inner float64
}

func makeDouble(val float64) Value {
	return Double{inner: val}
}

func (d Double) add(other Value) (Value, error) {
	switch s := other.score(); s {
	case scoreDouble:
		x, _ := toFloat(other)
		return makeDouble(d.inner + x), nil
	case scoreInt:
		i, _ := other.toDouble()
		return d.add(i)
	case scoreSlice:
		s, ok := other.(Slice)
		if !ok {
			return nil, ErrIncompatible
		}
		return s.apply(func(curr Value) (Value, error) {
			return d.add(curr)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (d Double) subtract(other Value) (Value, error) {
	switch s := other.score(); s {
	case scoreDouble:
		x, _ := toFloat(other)
		return makeDouble(d.inner - x), nil
	case scoreInt:
		i, _ := other.toDouble()
		return d.subtract(i)
	case scoreSlice:
		s, ok := other.(Slice)
		if !ok {
			return nil, ErrIncompatible
		}
		return s.apply(func(curr Value) (Value, error) {
			return d.subtract(curr)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (d Double) multiply(other Value) (Value, error) {
	switch s := other.score(); s {
	case scoreDouble:
		x, _ := toFloat(other)
		return makeDouble(d.inner * x), nil
	case scoreInt:
		i, _ := other.toDouble()
		return d.multiply(i)
	case scoreSlice:
		s, ok := other.(Slice)
		if !ok {
			return nil, ErrIncompatible
		}
		return s.apply(func(curr Value) (Value, error) {
			return d.multiply(curr)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (d Double) divide(other Value) (Value, error) {
	switch s := other.score(); s {
	case scoreDouble:
		x, _ := toFloat(other)
		if x == 0 {
			return nil, ErrZeroDiv
		}
		return makeDouble(d.inner / x), nil
	case scoreInt:
		i, _ := other.toDouble()
		return d.divide(i)
	case scoreSlice:
		s, ok := other.(Slice)
		if !ok {
			return nil, ErrIncompatible
		}
		return s.apply(func(curr Value) (Value, error) {
			return d.divide(curr)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (d Double) modulo(other Value) (Value, error) {
	switch s := other.score(); s {
	case scoreDouble:
		x, _ := toFloat(other)
		if x == 0 {
			return nil, ErrZeroDiv
		}
		return makeDouble(math.Mod(d.inner, x)), nil
	case scoreInt:
		i, _ := other.toDouble()
		return d.modulo(i)
	case scoreSlice:
		s, ok := other.(Slice)
		if !ok {
			return nil, ErrIncompatible
		}
		return s.apply(func(curr Value) (Value, error) {
			return d.modulo(curr)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (d Double) power(other Value) (Value, error) {
	switch s := other.score(); s {
	case scoreDouble:
		x, _ := toFloat(other)
		return makeDouble(math.Pow(d.inner, x)), nil
	case scoreInt:
		i, _ := other.toDouble()
		return d.power(i)
	case scoreSlice:
		s, ok := other.(Slice)
		if !ok {
			return nil, ErrIncompatible
		}
		return s.apply(func(curr Value) (Value, error) {
			return d.power(curr)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (d Double) reverse() (Value, error) {
	return makeDouble(-d.inner), nil
}

func (d Double) not() (Value, error) {
	return not(d), nil
}

func (d Double) and(other Value) (Value, error) {
	return and(d, other), nil
}

func (d Double) or(other Value) (Value, error) {
	return or(d, other), nil
}

func (d Double) compare(other Value) (int, error) {
	x, err := toFloat(other)
	if err != nil {
		return -1, err
	}
	if math.Abs(d.inner-x) < epsilon {
		return 0, nil
	}
	if d.inner > x {
		return 1, nil
	}
	return -1, nil
}

func (d Double) isTrue() bool {
	return d.inner != 0
}

func (_ Double) score() int {
	return scoreDouble
}

func (d Double) toInt() (Value, error) {
	return makeInt(int64(d.inner)), nil
}

func (d Double) toDouble() (Value, error) {
	return d, nil
}

func (d Double) toBool() (Value, error) {
	return makeBool(d.isTrue()), nil
}

func (_ Double) leftshift(other Value) (Value, error)  { return nil, ErrUnsupported }
func (_ Double) rightshift(other Value) (Value, error) { return nil, ErrUnsupported }
func (_ Double) binand(other Value) (Value, error)     { return nil, ErrUnsupported }
func (_ Double) binor(other Value) (Value, error)      { return nil, ErrUnsupported }
func (_ Double) binxor(other Value) (Value, error)     { return nil, ErrUnsupported }
func (_ Double) binnot() (Value, error)                { return nil, ErrUnsupported }
func (_ Double) toText() (Value, error)                { return nil, ErrIncompatible }
func (_ Double) toMoment() (Value, error)              { return nil, ErrIncompatible }
func (_ Double) at(_ Value) (Value, error)             { return nil, ErrUnsupported }

type Text struct {
	inner string
}

func makeText(str string) Value {
	return Text{inner: str}
}

func (t Text) add(other Value) (Value, error) {
	x, ok := other.(Text)
	if !ok {
		return nil, ErrIncompatible
	}
	return makeText(t.inner + x.inner), nil
}

func (t Text) multiply(other Value) (Value, error) {
	x, ok := other.(Int)
	if !ok {
		return nil, ErrIncompatible
	}
	return makeText(strings.Repeat(t.inner, int(x.inner))), nil
}

func (t Text) and(other Value) (Value, error) {
	return and(t, other), nil
}

func (t Text) or(other Value) (Value, error) {
	return or(t, other), nil
}

func (t Text) compare(other Value) (int, error) {
	x, ok := other.(Text)
	if !ok {
		return -1, ErrIncompatible
	}
	return strings.Compare(t.inner, x.inner), nil
}

func (t Text) isTrue() bool {
	return t.inner != ""
}

func (_ Text) score() int {
	return scoreText
}

func (t Text) toBool() (Value, error) {
	return makeBool(t.isTrue()), nil
}

func (t Text) toText() (Value, error) {
	return t, nil
}

func (t Text) toInt() (Value, error) {
	i, err := strconv.ParseInt(t.inner, 0, 64)
	if err != nil {
		return nil, err
	}
	return makeInt(i), nil
}

func (t Text) toDouble() (Value, error) {
	i, err := strconv.ParseFloat(t.inner, 64)
	if err != nil {
		return nil, err
	}
	return makeDouble(i), nil
}

func (t Text) toMoment() (Value, error) {
	var (
		when time.Time
		err  error
	)
	for _, pattern := range timePattern {
		when, err = time.Parse(pattern, t.inner)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	return makeMoment(when), nil
}

func (_ Text) at(_ Value) (Value, error)         { return nil, ErrUnsupported }
func (_ Text) subtract(_ Value) (Value, error)   { return nil, ErrUnsupported }
func (_ Text) divide(_ Value) (Value, error)     { return nil, ErrUnsupported }
func (_ Text) modulo(_ Value) (Value, error)     { return nil, ErrUnsupported }
func (_ Text) power(_ Value) (Value, error)      { return nil, ErrUnsupported }
func (_ Text) reverse() (Value, error)           { return nil, ErrUnsupported }
func (_ Text) not() (Value, error)               { return nil, ErrUnsupported }
func (_ Text) leftshift(_ Value) (Value, error)  { return nil, ErrUnsupported }
func (_ Text) rightshift(_ Value) (Value, error) { return nil, ErrUnsupported }
func (_ Text) binand(_ Value) (Value, error)     { return nil, ErrUnsupported }
func (_ Text) binor(_ Value) (Value, error)      { return nil, ErrUnsupported }
func (_ Text) binxor(_ Value) (Value, error)     { return nil, ErrUnsupported }
func (_ Text) binnot() (Value, error)            { return nil, ErrUnsupported }

type Moment struct {
	inner time.Time
}

func makeMoment(mmt time.Time) Value {
	return Moment{
		inner: mmt,
	}
}

func (m Moment) add(other Value) (Value, error) {
	x, err := toInt(other)
	if err != nil {
		return nil, err
	}
	y := m.adjust().(Moment)
	when := y.inner.Add(time.Duration(x) * time.Second)
	return makeMoment(when), nil
}

func (m Moment) addDuration(d time.Duration) (Value, error) {
	w := m.inner.Add(d)
	return makeMoment(w), nil
}

func (m Moment) subtract(other Value) (Value, error) {
	x, err := toInt(other)
	if err != nil {
		return nil, err
	}
	y := m.adjust().(Moment)
	when := y.inner.Add(time.Duration(-x) * time.Second)
	return makeMoment(when), nil
}

func (m Moment) compare(other Value) (int, error) {
	x, ok := other.(Moment)
	if !ok {
		return -1, ErrIncompatible
	}
	y := m.adjust().(Moment)
	if y.inner.Equal(x.inner) {
		return 0, nil
	}
	if y.inner.After(x.inner) {
		return 1, nil
	}
	return -1, nil
}

func (m Moment) and(other Value) (Value, error) {
	return and(m, other), nil
}

func (m Moment) or(other Value) (Value, error) {
	return or(m, other), nil
}

func (m Moment) not() (Value, error) {
	return not(m), nil
}

func (m Moment) isTrue() bool {
	return !m.inner.IsZero()
}

func (_ Moment) score() int {
	return scoreTime
}

func (m Moment) adjust() Value {
	if m.inner.Year() > 0 {
		return m
	}
	n := time.Now()
	n = m.inner.AddDate(n.Year(), int(n.Month()), n.Day()+1)
	return makeMoment(n)
}

func (m Moment) toInt() (Value, error) {
	return makeInt(m.inner.Unix()), nil
}

func (m Moment) toDouble() (Value, error) {
	return makeDouble(float64(m.inner.Unix())), nil
}

func (m Moment) toBool() (Value, error) {
	return makeBool(m.isTrue()), nil
}

func (m Moment) toText() (Value, error) {
	return makeText(m.inner.Format(time.RFC3339)), nil
}

func (m Moment) toMoment() (Value, error) {
	return m, nil
}

func (_ Moment) at(_ Value) (Value, error)         { return nil, ErrUnsupported }
func (_ Moment) multiply(_ Value) (Value, error)   { return nil, ErrUnsupported }
func (_ Moment) divide(_ Value) (Value, error)     { return nil, ErrUnsupported }
func (_ Moment) modulo(_ Value) (Value, error)     { return nil, ErrUnsupported }
func (_ Moment) power(_ Value) (Value, error)      { return nil, ErrUnsupported }
func (_ Moment) reverse() (Value, error)           { return nil, ErrUnsupported }
func (_ Moment) leftshift(_ Value) (Value, error)  { return nil, ErrUnsupported }
func (_ Moment) rightshift(_ Value) (Value, error) { return nil, ErrUnsupported }
func (_ Moment) binand(_ Value) (Value, error)     { return nil, ErrUnsupported }
func (_ Moment) binor(_ Value) (Value, error)      { return nil, ErrUnsupported }
func (_ Moment) binnot() (Value, error)            { return nil, ErrUnsupported }
func (_ Moment) binxor(_ Value) (Value, error)     { return nil, ErrUnsupported }

type Slice struct {
	inner []Value
}

func makeSlice(vs []Value) Value {
	return Slice{inner: vs}
}

func (s Slice) add(other Value) (Value, error) {
	switch sc := other.score(); sc {
	case scoreInt, scoreDouble:
		return s.apply(func(curr Value) (Value, error) {
			return curr.add(other)
		})
	case scoreSlice:
		return s.combine(other, func(fst, snd Value) (Value, error) {
			return fst.add(snd)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (s Slice) subtract(other Value) (Value, error) {
	switch sc := other.score(); sc {
	case scoreInt, scoreDouble:
		return s.apply(func(curr Value) (Value, error) {
			return curr.subtract(other)
		})
	case scoreSlice:
		return s.combine(other, func(fst, snd Value) (Value, error) {
			return fst.subtract(snd)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (s Slice) multiply(other Value) (Value, error) {
	switch sc := other.score(); sc {
	case scoreInt, scoreDouble:
		return s.apply(func(curr Value) (Value, error) {
			return curr.multiply(other)
		})
	case scoreSlice:
		return s.combine(other, func(fst, snd Value) (Value, error) {
			return fst.multiply(snd)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (s Slice) divide(other Value) (Value, error) {
	switch sc := other.score(); sc {
	case scoreInt, scoreDouble:
		return s.apply(func(curr Value) (Value, error) {
			return curr.divide(other)
		})
	case scoreSlice:
		return s.combine(other, func(fst, snd Value) (Value, error) {
			return fst.divide(snd)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (s Slice) modulo(other Value) (Value, error) {
	switch sc := other.score(); sc {
	case scoreInt, scoreDouble:
		return s.apply(func(curr Value) (Value, error) {
			return curr.modulo(other)
		})
	case scoreSlice:
		return s.combine(other, func(fst, snd Value) (Value, error) {
			return fst.modulo(snd)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (s Slice) power(other Value) (Value, error) {
	switch sc := other.score(); sc {
	case scoreInt, scoreDouble:
		return s.apply(func(curr Value) (Value, error) {
			return curr.power(other)
		})
	case scoreSlice:
		return s.combine(other, func(fst, snd Value) (Value, error) {
			return fst.power(snd)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (s Slice) isTrue() bool {
	return len(s.inner) > 0
}

func (s Slice) score() int {
	return scoreSlice
}

func (s Slice) and(other Value) (Value, error) {
	return and(s, other), nil
}

func (s Slice) or(other Value) (Value, error) {
	return or(s, other), nil
}

func (s Slice) compare(other Value) (int, error) {
	x, ok := other.(Slice)
	if !ok {
		return -1, ErrIncompatible
	}
	if len(s.inner) == len(x.inner) {
		return 0, nil
	}
	if len(s.inner) > len(x.inner) {
		return 1, nil
	}
	return -1, nil
}

func (s Slice) leftshift(other Value) (Value, error) {
	switch sc := other.score(); sc {
	case scoreInt, scoreDouble:
		return s.apply(func(curr Value) (Value, error) {
			return curr.leftshift(other)
		})
	case scoreSlice:
		return s.combine(other, func(fst, snd Value) (Value, error) {
			return fst.leftshift(snd)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (s Slice) rightshift(other Value) (Value, error) {
	switch sc := other.score(); sc {
	case scoreInt, scoreDouble:
		return s.apply(func(curr Value) (Value, error) {
			return curr.rightshift(other)
		})
	case scoreSlice:
		return s.combine(other, func(fst, snd Value) (Value, error) {
			return fst.rightshift(snd)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (s Slice) binand(other Value) (Value, error) {
	switch sc := other.score(); sc {
	case scoreInt, scoreDouble:
		return s.apply(func(curr Value) (Value, error) {
			return curr.binand(other)
		})
	case scoreSlice:
		return s.combine(other, func(fst, snd Value) (Value, error) {
			return fst.binand(snd)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (s Slice) binor(other Value) (Value, error) {
	switch sc := other.score(); sc {
	case scoreInt, scoreDouble:
		return s.apply(func(curr Value) (Value, error) {
			return curr.binor(other)
		})
	case scoreSlice:
		return s.combine(other, func(fst, snd Value) (Value, error) {
			return fst.binor(snd)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (s Slice) binnot() (Value, error) {
	return s.apply(func(curr Value) (Value, error) {
		return curr.binnot()
	})
}

func (s Slice) binxor(other Value) (Value, error) {
	switch sc := other.score(); sc {
	case scoreInt, scoreDouble:
		return s.apply(func(curr Value) (Value, error) {
			return curr.binxor(other)
		})
	case scoreSlice:
		return s.combine(other, func(fst, snd Value) (Value, error) {
			return fst.binxor(snd)
		})
	default:
		return nil, ErrIncompatible
	}
}

func (s Slice) at(ix Value) (Value, error) {
	y, err := toInt(ix)
	if err != nil {
		return nil, ErrIncompatible
	}
	x := int(y)
	if x < 0 {
		x = len(s.inner) + x
	}
	if x < 0 || x >= len(s.inner) {
		return nil, ErrIndex
	}
	return s.inner[x], nil
}

func (s Slice) combine(other Value, fn func(fst, snd Value) (Value, error)) (Value, error) {
	recv, ok := other.(Slice)
	if !ok {
		return nil, ErrIncompatible
	}
	if len(s.inner) != len(recv.inner) {
		return nil, ErrIncompatible
	}
	vs := make([]Value, len(s.inner))
	for i := range s.inner {
		x, err := fn(s.inner[i], recv.inner[i])
		if err != nil {
			return nil, err
		}
		vs[i] = x
	}
	return makeSlice(vs), nil
}

func (s Slice) apply(fn func(curr Value) (Value, error)) (Value, error) {
	vs := make([]Value, len(s.inner))
	for i := range s.inner {
		x, err := fn(s.inner[i])
		if err != nil {
			return nil, err
		}
		vs[i] = x
	}
	return makeSlice(vs), nil
}

func (_ Slice) reverse() (Value, error)  { return nil, ErrUnsupported }
func (_ Slice) not() (Value, error)      { return nil, ErrUnsupported }
func (_ Slice) toInt() (Value, error)    { return nil, ErrUnsupported }
func (_ Slice) toDouble() (Value, error) { return nil, ErrUnsupported }
func (_ Slice) toBool() (Value, error)   { return nil, ErrUnsupported }
func (_ Slice) toText() (Value, error)   { return nil, ErrUnsupported }
func (_ Slice) toMoment() (Value, error) { return nil, ErrUnsupported }

func and(left, right Value) Value {
	return makeBool(left.isTrue() && right.isTrue())
}

func or(left, right Value) Value {
	if left.isTrue() {
		return left
	}
	if right.isTrue() {
		return right
	}
	return makeBool(false)
}

func not(left Value) Value {
	return makeBool(!left.isTrue())
}

func toInt(v Value) (int64, error) {
	switch v := v.(type) {
	case Int:
		return v.inner, nil
	case Double:
		return int64(v.inner), nil
	default:
		return 0, ErrIncompatible
	}
}

func toFloat(v Value) (float64, error) {
	switch v := v.(type) {
	case Int:
		return float64(v.inner), nil
	case Double:
		return v.inner, nil
	default:
		return 0, ErrIncompatible
	}
}

func toText(v Value) (string, error) {
	switch v := v.(type) {
	case Int:
		return strconv.FormatInt(v.inner, 10), nil
	case Double:
		return strconv.FormatFloat(v.inner, 'f', -1, 64), nil
	case Bool:
		return strconv.FormatBool(v.inner), nil
	case Moment:
		return v.inner.Format(time.RFC3339), nil
	case Text:
		return v.inner, nil
	default:
		return "", ErrIncompatible
	}
}

func toTime(v Value) (time.Time, error) {
	switch v := v.(type) {
	case Moment:
		return v.inner, nil
	default:
		return time.Time{}, ErrIncompatible
	}
}
