package fig

import (
  "testing"
)

func TestInt(t *testing.T) {
  t.Run("basic", testBasicInt)
}

func testBasicInt(t *testing.T) {
  var (
    left = makeInt(4)
    right = makeInt(2)
    res Value
  )
  res, _ = left.add(right)
  checkInt(t, res, 6)
  res, _ = left.subtract(right)
  checkInt(t, res, 2)
  res, _ = left.multiply(right)
  checkInt(t, res, 8)
  res, _ = left.divide(right)
  checkInt(t, res, 2)
  res, _ = left.modulo(right)
  checkInt(t, res, 0)
  res, _ = left.power(right)
  checkInt(t, res, 16)
  res, _ = left.reverse()
  checkInt(t, res, -4)
  res, _ = left.leftshift(right)
  checkInt(t, res, 16)
  res, _ = left.rightshift(right)
  checkInt(t, res, 1)
  res, _ = left.binand(right)
  checkInt(t, res, 0)
  res, _ = left.binor(right)
  checkInt(t, res, 6)
}

func TestBool(t *testing.T) {
  t.SkipNow()
}

func checkInt(t *testing.T, got Value, want int64) {
  t.Helper()
  i, ok := got.(Int)
  if !ok {
    t.Errorf("unexpected result! %T", got)
    return
  }
  if i.inner != want {
    t.Errorf("result mismatched! want %d, got %d", want, i.inner)
  }
}
