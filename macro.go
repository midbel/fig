package fig

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

type macroFunc func(Node, []Argument, map[string]Argument) error

var errBadArgument = errors.New("argument")

type strategy int

const (
	sInvalid strategy = iota
	sAppend
	sMerge
	sReplace
)

func strategyFromString(str string) strategy {
	s := sInvalid
	switch str {
	case "merge", "", "default":
		s = sMerge
	case "append":
		s = sAppend
	case "replace":
		s = sReplace
	default:
	}
	return s
}

func (s strategy) Valid() bool {
	return s > sInvalid && s <= sReplace
}

type Argument interface {
	GetString() (string, error)
	GetFloat() (float64, error)
	GetInt() (int64, error)
	GetBool() (bool, error)
}

const (
	argFile  = "file"
	argKey   = "key"
	argFatal = "fatal"
	argMeth  = "method"
)

func Include(root Node, args []Argument, kwargs map[string]Argument) error {
	var (
		file   string
		key    string
		method string
		fatal  bool
		err    error
	)

	if file, err = getString(0, argFile, args, kwargs); err != nil {
		return err
	}
	if key, err = getString(1, argKey, args, kwargs); err != nil && !errors.Is(err, errBadArgument) {
		return err
	}
	if fatal, err = getBool(2, argFatal, args, kwargs); err != nil && !errors.Is(err, errBadArgument) {
		return err
	}
	if method, err = getString(3, argMeth, args, kwargs); err != nil && !errors.Is(err, errBadArgument) {
		return err
	}
	n, err := include(file, key, fatal)
	if err != nil || n == nil {
		return err
	}
	obj, ok := root.(*object)
	if !ok {
		return fmt.Errorf("root should be an object! got %T", root)
	}
	switch do := strategyFromString(method); do {
	case sReplace:
		err = obj.replace(n)
	case sAppend:
		err = obj.insert(n)
	case sMerge:
		err = obj.merge(n)
	default:
		err = fmt.Errorf("unknown/unsupported insertion method supplied")
	}
	return err
}

func include(file, key string, fatal bool) (Node, error) {
	var (
		u, _ = url.Parse(file)
		rc   io.ReadCloser
		err  error
	)
	switch u.Scheme {
	case "", "file":
		rc, err = readFile(file)
	case "http", "https":
		rc, err = readRemote(file)
	default:
		err = fmt.Errorf("%s: unsupported scheme (%s)", u.Scheme, file)
	}
	if err != nil {
		if !fatal {
			err = nil
		}
		return nil, err
	}
	defer rc.Close()

	node, err := Parse(rc)
	if err != nil {
		if !fatal {
			err = nil
		}
		return nil, err
	}
	obj, ok := node.(*object)
	if ok {
		obj.Name = filepath.Base(file)
		if key != "" {
			obj.Name = key
		}
	}
	return node, nil
}

func readFile(file string) (io.ReadCloser, error) {
	return os.Open(file)
}

func readRemote(file string) (io.ReadCloser, error) {
	res, err := http.Get(file)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf(res.Status)
	}
	return res.Body, nil
}

func getString(at int, field string, args []Argument, kwargs map[string]Argument) (string, error) {
	arg, err := checkHas(at+1, field, args, kwargs)
	if err != nil {
		return "", err
	}
	return arg.GetString()
}

func getBool(at int, field string, args []Argument, kwargs map[string]Argument) (bool, error) {
	arg, err := checkHas(at+1, field, args, kwargs)
	if err != nil {
		return false, err
	}
	return arg.GetBool()
}

func checkHas(at int, field string, args []Argument, kwargs map[string]Argument) (Argument, error) {
	arg, ok := kwargs[field]
	if len(args) < at && !ok {
		return nil, errArgument(field, "not supplied")
	}
	if len(args) >= at {
		if ok {
			return nil, errArgument(field, "given as positional and keyword")
		}
		arg = args[at-1]
	}
	return arg, nil
}

func errArgument(field, msg string) error {
	return fmt.Errorf("%s: %w %s", field, errBadArgument, msg)
}
