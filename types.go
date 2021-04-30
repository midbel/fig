package fig

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"time"
)

var (
	ErrIncompatible = errors.New("incompatible types")
	ErrUnsupported  = errors.New("unsupported operation")
	ErrZeroDiv      = errors.New("division by zero")
)

const (
	scoreBool int = iota
	scoreText
	scoreInt
	scoreDouble
	scoreTime
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

func (b Bool) add(_ Value) (Value, error)        { return nil, ErrUnsupported }
func (b Bool) subtract(_ Value) (Value, error)   { return nil, ErrUnsupported }
func (b Bool) multiply(_ Value) (Value, error)   { return nil, ErrUnsupported }
func (b Bool) divide(_ Value) (Value, error)     { return nil, ErrUnsupported }
func (b Bool) modulo(_ Value) (Value, error)     { return nil, ErrUnsupported }
func (b Bool) power(_ Value) (Value, error)      { return nil, ErrUnsupported }
func (b Bool) reverse() (Value, error)           { return nil, ErrUnsupported }
func (b Bool) leftshift(_ Value) (Value, error)  { return nil, ErrUnsupported }
func (b Bool) rightshift(_ Value) (Value, error) { return nil, ErrUnsupported }
func (b Bool) binand(_ Value) (Value, error)     { return nil, ErrUnsupported }
func (b Bool) binor(_ Value) (Value, error)      { return nil, ErrUnsupported }
func (b Bool) binnot() (Value, error)            { return nil, ErrUnsupported }
func (b Bool) binxor(_ Value) (Value, error)     { return nil, ErrUnsupported }
func (b Bool) toInt() (Value, error)             { return nil, ErrIncompatible }
func (b Bool) toDouble() (Value, error)          { return nil, ErrIncompatible }
func (b Bool) toText() (Value, error)            { return nil, ErrIncompatible }
func (b Bool) toMoment() (Value, error)          { return nil, ErrIncompatible }

type Int struct {
	inner int64
}

func makeInt(val int64) Value {
	return Int{inner: val}
}

func (i Int) add(other Value) (Value, error) {
	x, err := toInt(other)
	if err != nil {
		return nil, err
	}
	return makeInt(i.inner + x), nil
}

func (i Int) subtract(other Value) (Value, error) {
	x, err := toInt(other)
	if err != nil {
		return nil, err
	}
	return makeInt(i.inner - x), nil
}

func (i Int) multiply(other Value) (Value, error) {
	x, err := toInt(other)
	if err != nil {
		return nil, err
	}
	return makeInt(i.inner * x), nil
}

func (i Int) divide(other Value) (Value, error) {
	x, err := toInt(other)
	if err != nil {
		return nil, err
	}
	if x == 0 {
		return nil, ErrZeroDiv
	}
	return makeInt(i.inner / x), nil
}

func (i Int) modulo(other Value) (Value, error) {
	x, err := toInt(other)
	if err != nil {
		return nil, err
	}
	if x == 0 {
		return nil, ErrZeroDiv
	}
	return makeInt(i.inner % x), nil
}

func (i Int) power(other Value) (Value, error) {
	x, err := toInt(other)
	if err != nil {
		return nil, err
	}
	r := math.Pow(float64(i.inner), float64(x))
	return makeInt(int64(r)), nil
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
	x, err := toInt(other)
	if err != nil {
		return nil, err
	}
	return makeInt(i.inner << x), nil
}

func (i Int) rightshift(other Value) (Value, error) {
	x, err := toInt(other)
	if err != nil {
		return nil, err
	}
	return makeInt(i.inner >> x), nil
}

func (i Int) binand(other Value) (Value, error) {
	x, err := toInt(other)
	if err != nil {
		return nil, err
	}
	return makeInt(i.inner & x), nil
}

func (i Int) binor(other Value) (Value, error) {
	x, err := toInt(other)
	if err != nil {
		return nil, err
	}
	return makeInt(i.inner | x), nil
}

func (i Int) binxor(other Value) (Value, error) {
	x, err := toInt(other)
	if err != nil {
		return nil, err
	}
	return makeInt(i.inner ^ x), nil
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
	return nil, ErrIncompatible
}

type Double struct {
	inner float64
}

func makeDouble(val float64) Value {
	return Double{inner: val}
}

func (d Double) add(other Value) (Value, error) {
	x, err := toFloat(other)
	if err != nil {
		return nil, err
	}
	return makeDouble(d.inner + x), nil
}

func (d Double) subtract(other Value) (Value, error) {
	x, err := toFloat(other)
	if err != nil {
		return nil, err
	}
	return makeDouble(d.inner - x), nil
}

func (d Double) multiply(other Value) (Value, error) {
	x, err := toFloat(other)
	if err != nil {
		return nil, err
	}
	return makeDouble(d.inner * x), nil
}

func (d Double) divide(other Value) (Value, error) {
	x, err := toFloat(other)
	if err != nil {
		return nil, err
	}
	if x == 0 {
		return nil, ErrZeroDiv
	}
	return makeDouble(d.inner / x), nil
}

func (d Double) modulo(other Value) (Value, error) {
	x, err := toFloat(other)
	if err != nil {
		return nil, err
	}
	if x == 0 {
		return nil, ErrZeroDiv
	}
	return makeDouble(math.Mod(d.inner, x)), nil
}

func (d Double) power(other Value) (Value, error) {
	x, err := toFloat(other)
	if err != nil {
		return nil, err
	}
	return makeDouble(math.Pow(d.inner, x)), nil
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

func (d Double) leftshift(other Value) (Value, error)  { return nil, ErrUnsupported }
func (d Double) rightshift(other Value) (Value, error) { return nil, ErrUnsupported }
func (d Double) binand(other Value) (Value, error)     { return nil, ErrUnsupported }
func (d Double) binor(other Value) (Value, error)      { return nil, ErrUnsupported }
func (d Double) binxor(other Value) (Value, error)     { return nil, ErrUnsupported }
func (d Double) binnot() (Value, error)                { return nil, ErrUnsupported }
func (d Double) toText() (Value, error)                { return nil, ErrIncompatible }
func (d Double) toMoment() (Value, error)              { return nil, ErrIncompatible }

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

func (t Text) subtract(_ Value) (Value, error)   { return nil, ErrUnsupported }
func (t Text) multiply(_ Value) (Value, error)   { return nil, ErrUnsupported }
func (t Text) divide(_ Value) (Value, error)     { return nil, ErrUnsupported }
func (t Text) modulo(_ Value) (Value, error)     { return nil, ErrUnsupported }
func (t Text) power(_ Value) (Value, error)      { return nil, ErrUnsupported }
func (t Text) reverse() (Value, error)           { return nil, ErrUnsupported }
func (t Text) not() (Value, error)               { return nil, ErrUnsupported }
func (t Text) leftshift(_ Value) (Value, error)  { return nil, ErrUnsupported }
func (t Text) rightshift(_ Value) (Value, error) { return nil, ErrUnsupported }
func (t Text) binand(_ Value) (Value, error)     { return nil, ErrUnsupported }
func (t Text) binor(_ Value) (Value, error)      { return nil, ErrUnsupported }
func (t Text) binxor(_ Value) (Value, error)     { return nil, ErrUnsupported }
func (t Text) binnot() (Value, error)            { return nil, ErrUnsupported }
func (t Text) toInt() (Value, error)             { return nil, ErrIncompatible }
func (t Text) toDouble() (Value, error)          { return nil, ErrIncompatible }
func (t Text) toMoment() (Value, error)          { return nil, ErrIncompatible }

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

func (m Moment) multiply(_ Value) (Value, error)   { return nil, ErrUnsupported }
func (m Moment) divide(_ Value) (Value, error)     { return nil, ErrUnsupported }
func (m Moment) modulo(_ Value) (Value, error)     { return nil, ErrUnsupported }
func (m Moment) power(_ Value) (Value, error)      { return nil, ErrUnsupported }
func (m Moment) reverse() (Value, error)           { return nil, ErrUnsupported }
func (m Moment) leftshift(_ Value) (Value, error)  { return nil, ErrUnsupported }
func (m Moment) rightshift(_ Value) (Value, error) { return nil, ErrUnsupported }
func (m Moment) binand(_ Value) (Value, error)     { return nil, ErrUnsupported }
func (m Moment) binor(_ Value) (Value, error)      { return nil, ErrUnsupported }
func (m Moment) binnot() (Value, error)            { return nil, ErrUnsupported }
func (m Moment) binxor(_ Value) (Value, error)     { return nil, ErrUnsupported }

func and(left, right Value) Value {
	return makeBool(left.isTrue() && right.isTrue())
}

func or(left, right Value) Value {
	return makeBool(left.isTrue() && right.isTrue())
}

func not(left Value) Value {
	return makeBool(!left.isTrue())
}

// int <op> int => int
// float <op> float => float
// int <op> float => float
// bool <op> bool => bool
// * <op> bool => incompatible
// * <op> text => text
// int <op> moment => moment
// float <op> moment => moment

func add(left, right Value) (Value, error) {
	var err error
	left, right, err = promote(left, right)
	if err != nil {
		return nil, err
	}
	return left.add(right)
}

func subtract(left, right Value) (Value, error) {
	var err error
	left, right, err = promote(left, right)
	if err != nil {
		return nil, err
	}
	return left.subtract(right)
}

func multiply(left, right Value) (Value, error) {
	var err error
	left, right, err = promote(left, right)
	if err != nil {
		return nil, err
	}
	return left.multiply(right)
}

func divide(left, right Value) (Value, error) {
	var err error
	left, right, err = promote(left, right)
	if err != nil {
		return nil, err
	}
	return left.divide(right)
}

func modulo(left, right Value) (Value, error) {
	var err error
	left, right, err = promote(left, right)
	if err != nil {
		return nil, err
	}
	return left.modulo(right)
}

func power(left, right Value) (Value, error) {
	return left.power(right)
}

func reverse(left Value) (Value, error) {
	return left.reverse()
}

func compare(left, right Value) (int, error) {
	var err error
	left, right, err = promote(left, right)
	if err != nil {
		return 0, err
	}
	return left.compare(right)
}

func leftshift(left, right Value) (Value, error) {
	return left.leftshift(right)
}

func rightshift(left, right Value) (Value, error) {
	return left.rightshift(right)
}

func binand(left, right Value) (Value, error) {
	return left.binand(right)
}

func binor(left, right Value) (Value, error) {
	return left.binor(right)
}

func binxor(left, right Value) (Value, error) {
	return left.binxor(right)
}

func binnot(left Value) (Value, error) {
	return left.binnot()
}

func promote(left, right Value) (Value, Value, error) {
	var err error
	if left.score() < right.score() {
		left, err = promoteValue(left, right)
	} else if left.score() > right.score() {
		right, err = promoteValue(right, left)
	}
	return left, right, err
}

func promoteValue(left, right Value) (Value, error) {
	var err error
	switch right.(type) {
	case Int:
		left, err = left.toInt()
	case Double:
		left, err = left.toDouble()
	case Bool:
		left, err = left.toBool()
	case Moment:
		left, err = left.toMoment()
	case Text:
		left, err = left.toText()
	default:
		err = ErrIncompatible
	}
	return left, err
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
