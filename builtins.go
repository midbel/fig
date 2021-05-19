package fig

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrArgument = errors.New("invalid number of argument(s)")
	ErrMissing  = errors.New("missing argument")
)

// var builtins = map[string]func(...Value) (Value, error){
// 	"typeof":   typeof,
// 	"len":      length,
// 	"first":    first,
// 	"last":     last,
// 	"rand":     rand,
// 	"sqrt":     sqrt,
// 	"abs":      abs,
// 	"max":      max,
// 	"min":      min,
// 	"seq":      sequence,
// 	"all":      all,
// 	"any":      any,
// 	"avg":      avg,
// 	"upper":    upper,
// 	"lower":    lower,
// 	"split":    split,
// 	"join":     join,
// 	"contains": contains,
// 	"substr":   substring,
// 	"trim":     trim,
// 	"replace":  replace,
// 	"dirname":  dirname,
// 	"dir":      dirname,
// 	"basename": basename,
// 	"base":     basename,
// 	"isdir":    isDir,
// 	"isfile":   isFile,
// 	"read":     read,
// }

type Builtin struct {
	name  string
	alias []string
	args  []Argument
	exec  func(Environment) (Value, error)
}

var builtins = map[string]Builtin{
	"typeof": {
		name: "typeof",
		args: []Argument{
			{
				name: makeToken("obj", Ident),
				pos:  0,
			},
		},
		exec: typeof,
	},
	"len": {
		name: "len",
		args: []Argument{
			{
				name: makeToken("obj", Ident),
				pos:  0,
			},
		},
		exec: length,
	},
	"first": {
		name: "first",
		args: []Argument{
			{
				name: makeToken("arr", Ident),
				pos:  0,
			},
		},
		exec: first,
	},
	"last": {
		name: "last",
		args: []Argument{
			{
				name: makeToken("arr", Ident),
				pos:  0,
			},
		},
		exec: last,
	},
	"seq": {
		name: "seq",
		args: []Argument{
			{
				name: makeToken("first", Ident),
				pos:  0,
			},
			{
				name: makeToken("last", Ident),
				pos:  1,
			},
			{
				name: makeToken("step", Ident),
				pos:  2,
				expr: makeLiteral(makeToken("1", Ident)),
			},
		},
		exec: sequence,
	},
	"randn": {
		name: "randn",
		args: []Argument{
			{
				name: makeToken("num", Ident),
				pos:  0,
			},
		},
		exec: randn,
	},
	"abs": {
		name: "abs",
		args: []Argument{
			{
				name: makeToken("num", Ident),
				pos:  0,
			},
		},
		exec: sqrt,
	},
	"sqrt": {
		name: "sqrt",
		args: []Argument{
			{
				name: makeToken("num", Ident),
				pos:  0,
			},
		},
		exec: sqrt,
	},
	"min": {
		name: "min",
		args: []Argument{
			{
				name:     makeToken("args", Ident),
				pos:      0,
				variadic: true,
			},
		},
		exec: min,
	},
	"max": {
		name: "max",
		args: []Argument{
			{
				name:     makeToken("args", Ident),
				pos:      0,
				variadic: true,
			},
		},
		exec: sqrt,
	},
	"all": {
		name: "all",
		args: []Argument{
			{
				name:     makeToken("args", Ident),
				pos:      0,
				variadic: true,
			},
		},
		exec: all,
	},
	"any": {
		name: "any",
		args: []Argument{
			{
				name:     makeToken("args", Ident),
				pos:      0,
				variadic: true,
			},
		},
		exec: any,
	},
	"avg": {
		name: "avg",
		args: []Argument{
			{
				name:     makeToken("args", Ident),
				pos:      0,
				variadic: true,
			},
		},
		exec: avg,
	},
	"upper": {
		name: "upper",
		args: []Argument{
			{
				name: makeToken("str", Ident),
				pos:  0,
			},
		},
		exec: upper,
	},
	"lower": {
		name: "lower",
		args: []Argument{
			{
				name: makeToken("str", Ident),
				pos:  0,
			},
		},
		exec: lower,
	},
	"split": {
		name: "split",
		args: []Argument{
			{
				name: makeToken("str", Ident),
				pos:  0,
			},
			{
				name: makeToken("sep", Ident),
				pos:  1,
				expr: makeLiteral(makeToken(" ", String)),
			},
		},
		exec: split,
	},
	"join": {
		name: "join",
		args: []Argument{
			{
				name: makeToken("arr", Ident),
				pos:  0,
			},
			{
				name: makeToken("sep", Ident),
				pos:  1,
				expr: makeLiteral(makeToken(" ", String)),
			},
		},
		exec: join,
	},
	"contains": {
		name: "contains",
		args: []Argument{
			{
				name: makeToken("str", Ident),
				pos:  0,
			},
			{
				name: makeToken("substr", Ident),
				pos:  1,
			},
		},
		exec: contains,
	},
	"substr": {
		name: "substr",
		args: []Argument{
			{
				name: makeToken("str", Ident),
				pos:  0,
			},
			{
				name: makeToken("pos", Ident),
				pos:  1,
				expr: makeLiteral(makeToken("0", Integer)),
			},
			{
				name: makeToken("len", Ident),
				pos:  2,
				expr: makeLiteral(makeToken("0", Integer)),
			},
		},
		exec: substring,
	},
	"trim": {
		name: "trim",
		args: []Argument{
			{
				name: makeToken("str", Ident),
				pos:  0,
			},
			{
				name: makeToken("char", Ident),
				pos:  1,
				expr: makeLiteral(makeToken("", Integer)),
			},
		},
		exec: trim,
	},
	"replace": {
		name: "replace",
		args: []Argument{
			{
				name: makeToken("str", Ident),
				pos:  0,
			},
			{
				name: makeToken("src", Ident),
				pos:  1,
			},
			{
				name: makeToken("dst", Ident),
				pos:  2,
			},
			{
				name: makeToken("count", Ident),
				pos:  3,
				expr: makeLiteral(makeToken("0", Integer)),
			},
		},
		exec: replace,
	},
	"dirname": {
		name:  "dirname",
		alias: []string{"dir"},
		args: []Argument{
			{
				name: makeToken("path", Ident),
				pos:  0,
			},
		},
		exec: dirname,
	},
	"basename": {
		name:  "base",
		alias: []string{"base"},
		args: []Argument{
			{
				name: makeToken("path", Ident),
				pos:  0,
			},
		},
		exec: basename,
	},
	"isfile": {
		name: "isfile",
		args: []Argument{
			{
				name: makeToken("path", Ident),
				pos:  0,
			},
		},
		exec: isFile,
	},
	"isdir": {
		name: "isdir",
		args: []Argument{
			{
				name: makeToken("path", Ident),
				pos:  0,
			},
		},
		exec: isDir,
	},
	"read": {
		name: "read",
		args: []Argument{
			{
				name:     makeToken("file", Ident),
				pos:      0,
				variadic: true,
			},
		},
		exec: read,
	},
	"b64encode": {
		name: "b64encode",
		args: []Argument{
			{
				name: makeToken("str", Ident),
				pos:  0,
			},
			{
				name: makeToken("url", Ident),
				pos:  1,
				expr: makeLiteral(makeToken("false", Boolean)),
			},
		},
		exec: base64Encode,
	},
	"b64decode": {
		name: "b64decode",
		args: []Argument{
			{
				name: makeToken("str", Ident),
				pos:  0,
			},
			{
				name: makeToken("url", Ident),
				pos:  1,
				expr: makeLiteral(makeToken("false", Boolean)),
			},
		},
		exec: base64Decode,
	},
}

func (b Builtin) Eval(e Environment) (Value, error) {
	return b.exec(e)
}

func (b Builtin) copyArgs() []Argument {
	as := make([]Argument, len(b.args))
	copy(as, b.args)
	return as
}

func typeof(e Environment) (Value, error) {
	obj, err := e.Resolve("obj")
	if err != nil {
		return nil, err
	}
	var kind string
	switch obj.score() {
	case scoreInt:
		kind = "integer"
	case scoreDouble:
		kind = "double"
	case scoreTime:
		kind = "moment"
	case scoreBool:
		kind = "boolean"
	case scoreSlice:
		kind = "array"
	default:
		return nil, fmt.Errorf("unsupported type")
	}
	return makeText(kind), nil
}

func length(e Environment) (Value, error) {
	obj, err := e.Resolve("obj")
	if err != nil {
		return nil, err
	}
	size := -1
	switch v := obj.(type) {
	case Text:
		size = len(v.inner)
	case Slice:
		size = len(v.inner)
	default:
	}
	return makeInt(int64(size)), nil
}

func randn(e Environment) (Value, error) {
	num, err := intFromEnv(e, "num")
	if err != nil {
		return nil, err
	}
	return makeInt(rand.Int63n(num)), nil
}

func first(e Environment) (Value, error) {
	xs, err := sliceFromEnv(e, "arr")
	if err != nil {
		return nil, err
	}
	if len(xs) == 0 {
		return nil, fmt.Errorf("empty array")
	}
	return xs[0], nil
}

func last(e Environment) (Value, error) {
	xs, err := sliceFromEnv(e, "arr")
	if err != nil {
		return nil, err
	}
	if len(xs) == 0 {
		return nil, fmt.Errorf("empty array")
	}
	return xs[len(xs)-1], nil
}

func sequence(e Environment) (Value, error) {
	var (
		fst  int64
		lst  int64
		step int64
		err  error
	)
	if fst, err = intFromEnv(e, "first"); err != nil {
		return nil, err
	}
	if lst, err = intFromEnv(e, "last"); err != nil {
		return nil, err
	}
	if step, err = intFromEnv(e, "step"); err != nil && !errors.Is(err, ErrUndefined) {
		return nil, err
	}

	if lst < fst {
		step = -step
	}
	var xs []Value
	for fst < lst {
		xs = append(xs, makeInt(fst))
		fst += step
	}
	return makeSlice(xs), nil
}

func sqrt(e Environment) (Value, error) {
	value, err := doubleFromEnv(e, "num")
	if err != nil {
		return nil, err
	}
	return makeDouble(math.Sqrt(value)), nil
}

func abs(e Environment) (Value, error) {
	value, err := doubleFromEnv(e, "num")
	if err != nil {
		return nil, err
	}
	return makeDouble(math.Abs(value)), nil
}

func max(e Environment) (Value, error) {
	vs, err := sliceFromEnv(e, "args")
	if err != nil {
		return nil, err
	}
	if len(vs) == 1 {
		return vs[0], nil
	}
	result := vs[0]
	for i := 1; i < len(vs); i++ {
		cmp, err := vs[i].compare(result)
		if err != nil {
			return nil, err
		}
		if cmp > 0 {
			result = vs[i]
		}
	}
	return result, nil
}

func min(e Environment) (Value, error) {
	vs, err := sliceFromEnv(e, "args")
	if err != nil {
		return nil, err
	}
	if len(vs) == 1 {
		return vs[0], nil
	}
	result := vs[0]
	for i := 1; i < len(vs); i++ {
		cmp, err := vs[i].compare(result)
		if err != nil {
			return nil, err
		}
		if cmp < 0 {
			result = vs[i]
		}
	}
	return result, nil
}

func all(e Environment) (Value, error) {
	vs, err := sliceFromEnv(e, "args")
	if err != nil {
		return nil, err
	}
	if len(vs) == 0 {
		return makeBool(true), nil
	}
	for _, v := range vs {
		if !v.isTrue() {
			return makeBool(false), nil
		}
	}
	return makeBool(true), nil
}

func any(e Environment) (Value, error) {
	vs, err := sliceFromEnv(e, "args")
	if err != nil {
		return nil, err
	}
	if len(vs) == 0 {
		return makeBool(false), nil
	}
	for _, v := range vs {
		if v.isTrue() {
			return makeBool(true), nil
		}
	}
	return makeBool(false), nil
}

func avg(e Environment) (Value, error) {
	vs, err := sliceFromEnv(e, "args")
	if err != nil {
		return nil, err
	}
	var value float64
	for _, v := range vs {
		v, err := toFloat(v)
		if err != nil {
			return nil, err
		}
		value += v
	}
	return makeDouble(value / float64(len(vs))), nil
}

func replace(e Environment) (Value, error) {
	var (
		value string
		bef   string
		aft   string
		count int64
		err   error
	)
	if value, err = stringFromEnv(e, "str"); err != nil {
		return nil, err
	}
	if bef, err = stringFromEnv(e, "src"); err != nil {
		return nil, err
	}
	if aft, err = stringFromEnv(e, "dst"); err != nil {
		return nil, err
	}
	if count, err = intFromEnv(e, "count"); err != nil {
		return nil, err
	}
	value = strings.Replace(value, bef, aft, int(count))
	return makeText(value), nil
}

func trim(e Environment) (Value, error) {
	value, err := stringFromEnv(e, "str")
	if err != nil {
		return nil, err
	}
	char, err := stringFromEnv(e, "char")
	if err != nil && !errors.Is(err, ErrUndefined) {
		return nil, err
	}
	if char == "" {
		return makeText(strings.TrimSpace(value)), nil
	}
	return makeText(strings.Trim(value, char)), nil
}

func upper(e Environment) (Value, error) {
	str, err := stringFromEnv(e, "str")
	if err != nil {
		return nil, err
	}
	return makeText(strings.ToUpper(str)), nil
}

func lower(e Environment) (Value, error) {
	str, err := stringFromEnv(e, "str")
	if err != nil {
		return nil, err
	}
	return makeText(strings.ToLower(str)), nil
}

func contains(e Environment) (Value, error) {
	value, err := stringFromEnv(e, "str")
	if err != nil {
		return nil, err
	}
	substr, err := stringFromEnv(e, "substr")
	if err != nil {
		return nil, err
	}
	return makeBool(strings.Contains(value, substr)), nil
}

func substring(e Environment) (Value, error) {
	return nil, nil
}

func split(e Environment) (Value, error) {
	value, err := stringFromEnv(e, "str")
	if err != nil {
		return nil, err
	}
	sep, err := stringFromEnv(e, "sep")
	if err != nil {
		return nil, err
	}
	var (
		parts  = strings.Split(value, sep)
		values = make([]Value, len(parts))
	)
	for i := range parts {
		values[i] = makeText(parts[i])
	}
	return makeSlice(values), nil
}

func join(e Environment) (Value, error) {
	sep, err := stringFromEnv(e, "sep")
	if err != nil {
		return nil, err
	}
	vs, err := sliceFromEnv(e, "arr")
	if err != nil {
		return nil, err
	}
	parts := make([]string, len(vs))
	for i := range vs {
		parts[i], err = toText(vs[i])
		if err != nil {
			return nil, err
		}
	}
	return makeText(strings.Join(parts, sep)), nil
}

func isDir(e Environment) (Value, error) {
	value, err := stringFromEnv(e, "path")
	if err != nil {
		return nil, err
	}
	i, err := os.Stat(value)
	if err != nil {
		return nil, err
	}
	return makeBool(i.Mode().IsDir()), nil
}

func isFile(e Environment) (Value, error) {
	value, err := stringFromEnv(e, "path")
	if err != nil {
		return nil, err
	}
	i, err := os.Stat(value)
	if err != nil {
		return nil, err
	}
	return makeBool(i.Mode().IsRegular()), nil
}

func read(e Environment) (Value, error) {
	value, err := stringFromEnv(e, "file")
	if err != nil {
		return nil, err
	}
	bs, err := os.ReadFile(value)
	if err != nil {
		return nil, err
	}
	return makeText(string(bs)), nil
}

func dirname(e Environment) (Value, error) {
	value, err := stringFromEnv(e, "path")
	if err != nil {
		return nil, err
	}
	return makeText(filepath.Dir(value)), nil
}

func basename(e Environment) (Value, error) {
	value, err := stringFromEnv(e, "path")
	if err != nil {
		return nil, err
	}
	return makeText(filepath.Base(value)), nil
}

func base64Encode(e Environment) (Value, error) {
	value, err := stringFromEnv(e, "str")
	if err != nil {
		return nil, err
	}
	value = base64.StdEncoding.EncodeToString([]byte(value))
	return makeText(value), nil
}

func base64Decode(e Environment) (Value, error) {
	value, err := stringFromEnv(e, "str")
	if err != nil {
		return nil, err
	}
	str, err := base64.StdEncoding.DecodeString(value)
	return makeText(string(str)), err
}

func invalidArgument(fn string) error {
	return fmt.Errorf("%s: %w given", fn, ErrArgument)
}

func missingArgument(fn, arg string) error {
	return fmt.Errorf("%s: %w %s", fn, ErrMissing, arg)
}
