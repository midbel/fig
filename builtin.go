package fig

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrArgument = errors.New("invalid number of argument(s)")
	ErrMissing  = errors.New("missing argument")
)

var builtins = map[string]func(...Value) (Value, error){
	"len":              length,
	"rand":             rand,
	"sqrt":             sqrt,
	"abs":              abs,
	"max":              max,
	"min":              min,
	"all":              all,
	"any":              any,
	"avg":              avg,
	"upper":            upper,
	"lower":            lower,
	"split":            split,
	"join":             join,
	"contains":         contains,
	"substr":           substring,
	"trim":             trim,
	"replace":          replace,
	"dirname":          dirname,
	"dir":              dirname,
	"basename":         basename,
	"base":             basename,
	"isdir":            isDir,
	"isfile":           isFile,
	"base64_encode":    base64EncodeStd,
	"base64_decode":    base64DecodeStd,
	"base64_urlencode": base64EncodeUrl,
	"base64_urldecode": base64DecodeUrl,
}

func length(vs ...Value) (Value, error) {
	if len(vs) != 1 {
		return nil, invalidArgument("len")
	}
	size := -1
	switch v := vs[0].(type) {
	case Text:
		size = len(v.inner)
	default:
	}
	return makeInt(int64(size)), nil
}

func rand(vs ...Value) (Value, error) {
	return nil, nil
}

func sqrt(vs ...Value) (Value, error) {
	if len(vs) != 1 {
		return nil, invalidArgument("sqrt")
	}
	value, err := toFloat(vs[0])
	if err != nil {
		return nil, err
	}
	return makeDouble(math.Sqrt(value)), nil
}

func abs(vs ...Value) (Value, error) {
	if len(vs) != 1 {
		return nil, invalidArgument("sqrt")
	}
	value, err := toFloat(vs[0])
	if err != nil {
		return nil, err
	}
	return makeDouble(math.Abs(value)), nil
}

func max(vs ...Value) (Value, error) {
	if len(vs) == 0 {
		return nil, invalidArgument("max")
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

func min(vs ...Value) (Value, error) {
	if len(vs) == 0 {
		return nil, invalidArgument("min")
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

func all(vs ...Value) (Value, error) {
	if len(vs) == 0 {
		return makeBool(false), nil
	}
	for _, v := range vs {
		if !v.isTrue() {
			return makeBool(false), nil
		}
	}
	return makeBool(true), nil
}

func any(vs ...Value) (Value, error) {
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

func avg(vs ...Value) (Value, error) {
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

func replace(vs ...Value) (Value, error) {
	var (
		value string
		bef   string
		aft   string
		err   error
	)
	if value, err = toText(vs[0]); err != nil {
		return nil, err
	}
	if bef, err = toText(vs[1]); err != nil {
		return nil, err
	}
	if aft, err = toText(vs[2]); err != nil {
		return nil, err
	}
	value = strings.ReplaceAll(value, bef, aft)
	return makeText(value), nil
}

func trim(vs ...Value) (Value, error) {
	if len(vs) == 0 || len(vs) > 2 {
		return nil, invalidArgument("trim")
	}
	value, err := toText(vs[0])
	if err != nil {
		return nil, err
	}
	if len(vs) == 1 {
		return makeText(strings.TrimSpace(value)), nil
	}
	str, err := toText(vs[1])
	if err != nil {
		return nil, err
	}
	return makeText(strings.Trim(value, str)), nil
}

func upper(vs ...Value) (Value, error) {
	if len(vs) != 1 {
		return nil, invalidArgument("upper")
	}
	str, err := toText(vs[0])
	if err != nil {
		return nil, err
	}
	return makeText(strings.ToUpper(str)), nil
}

func lower(vs ...Value) (Value, error) {
	if len(vs) != 1 {
		return nil, invalidArgument("lower")
	}
	str, err := toText(vs[0])
	if err != nil {
		return nil, err
	}
	return makeText(strings.ToLower(str)), nil
}

func contains(vs ...Value) (Value, error) {
	if len(vs) <= 1 {
		return nil, invalidArgument("contains")
	}
	value, err := toText(vs[0])
	if err != nil {
		return nil, err
	}
	for i := 1; i < len(vs); i++ {
		str, err := toText(vs[i])
		if err != nil {
			return nil, err
		}
		if strings.Contains(value, str) {
			return makeBool(true), nil
		}
	}
	return makeBool(false), nil
}

func substring(vs ...Value) (Value, error) {
	return nil, nil
}

func split(vs ...Value) (Value, error) {
	if len(vs) != 2 {
		return nil, invalidArgument("split")
	}
	value, err := toText(vs[0])
	if err != nil {
		return nil, err
	}
	char, err := toText(vs[1])
	if err != nil {
		return nil, err
	}
	_ = strings.Split(value, char)
	return nil, nil
}

func join(vs ...Value) (Value, error) {
	if len(vs) != 2 {
		return nil, invalidArgument("join")
	}
	var value []string
	char, err := toText(vs[1])
	if err != nil {
		return nil, err
	}
	return makeText(strings.Join(value, char)), nil
}

func isDir(vs ...Value) (Value, error) {
	if len(vs) != 1 {
		return nil, invalidArgument("isdir")
	}
	value, err := toText(vs[0])
	if err != nil {
		return nil, err
	}
	i, err := os.Stat(value)
	if err != nil {
		return nil, err
	}
	return makeBool(i.Mode().IsDir()), nil
}

func isFile(vs ...Value) (Value, error) {
	if len(vs) != 1 {
		return nil, invalidArgument("isfile")
	}
	value, err := toText(vs[0])
	if err != nil {
		return nil, err
	}
	i, err := os.Stat(value)
	if err != nil {
		return nil, err
	}
	return makeBool(i.Mode().IsRegular()), nil
}

func dirname(vs ...Value) (Value, error) {
	if len(vs) != 1 {
		return nil, invalidArgument("dirname")
	}
	value, err := toText(vs[0])
	if err != nil {
		return nil, err
	}
	return makeText(filepath.Dir(value)), nil
}

func basename(vs ...Value) (Value, error) {
	if len(vs) != 1 {
		return nil, invalidArgument("basename")
	}
	value, err := toText(vs[0])
	if err != nil {
		return nil, err
	}
	return makeText(filepath.Base(value)), nil
}

func base64EncodeStd(vs ...Value) (Value, error) {
	if len(vs) != 1 {
		return nil, invalidArgument("base64_encode")
	}
	value, err := toText(vs[0])
	if err != nil {
		return nil, err
	}
	value = base64.StdEncoding.EncodeToString([]byte(value))
	return makeText(value), nil
}

func base64DecodeStd(vs ...Value) (Value, error) {
	if len(vs) != 1 {
		return nil, invalidArgument("base64_decode")
	}
	value, err := toText(vs[0])
	if err != nil {
		return nil, err
	}
	str, err := base64.StdEncoding.DecodeString(value)
	return makeText(string(str)), err
}

func base64EncodeUrl(vs ...Value) (Value, error) {
	if len(vs) != 1 {
		return nil, invalidArgument("base64_urlencode")
	}
	value, err := toText(vs[0])
	if err != nil {
		return nil, err
	}
	value = base64.URLEncoding.EncodeToString([]byte(value))
	return makeText(value), nil
}

func base64DecodeUrl(vs ...Value) (Value, error) {
	if len(vs) != 1 {
		return nil, invalidArgument("base64_urldecode")
	}
	value, err := toText(vs[0])
	if err != nil {
		return nil, err
	}
	str, err := base64.URLEncoding.DecodeString(value)
	return makeText(string(str)), err
}

func invalidArgument(fn string) error {
	return fmt.Errorf("%s: %w given", fn, ErrArgument)
}

func missingArgument(fn, arg string) error {
	return fmt.Errorf("%s: %w %s", fn, ErrMissing, arg)
}
