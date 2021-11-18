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

type macroFunc func(root, obj Node, args []Node, kwargs map[string]Node) error

type macrodef struct {
	macroFunc
	withobject bool
}

func createMacroDef(fn macroFunc, with bool) macrodef {
	return macrodef{
		macroFunc:  fn,
		withobject: with,
	}
}

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

const (
	argFile   = "file"
	argFatal  = "fatal"
	argMeth   = "method"
	argName   = "name"
	argFields = "fields"
	argDepth  = "depth"
)

func Define(root, nest Node, args []Node, kwargs map[string]Node) error {
	if len(args) == 0 && len(kwargs) == 0 {
		return fmt.Errorf("no enough arguments supplied")
	}
	var (
		name   string
		method string
		err    error
	)
	if name, err = getString(0, argName, args, kwargs); err != nil {
		return err
	}
	if method, err = getString(1, argMeth, args, kwargs); err != nil && !errors.Is(err, errBadArgument) {
		return err
	}
	_ = method
	obj, ok := root.(*object)
	if !ok {
		return fmt.Errorf("root should be an object! got %T", root)
	}
	obj.define(name, nest)
	return nil
}

func Apply(root, _ Node, args []Node, kwargs map[string]Node) error {
	if len(args) == 0 && len(kwargs) == 0 {
		return fmt.Errorf("no enough arguments supplied")
	}
	var (
		name   string
		fields []string
		depth  int64
		method string
		err    error
	)
	if name, err = getString(0, argName, args, kwargs); err != nil {
		return err
	}
	if fields, err = getStringSlice(1, argFields, args, kwargs); err != nil && !errors.Is(err, errBadArgument) {
		return err
	}
	if depth, err = getInt(2, argDepth, args, kwargs); err != nil && !errors.Is(err, errBadArgument) {
		return err
	}
	if method, err = getString(3, argMeth, args, kwargs); err != nil && !errors.Is(err, errBadArgument) {
		return err
	}
	obj, ok := root.(*object)
	if !ok {
		return fmt.Errorf("root should be an object! got %T", root)
	}
	other, err := obj.apply(name, fields, depth)
	if err != nil {
		return err
	}
	switch do := strategyFromString(method); do {
	case sReplace:
		err = obj.replace(other)
	case sAppend:
		err = obj.insert(other)
	case sMerge:
		err = obj.merge(other)
	default:
		err = fmt.Errorf("unknown/unsupported insertion method supplied")
	}
	return nil
}

func Include(root, _ Node, args []Node, kwargs map[string]Node) error {
	var (
		file   string
		name   string
		method string
		fatal  bool
		err    error
	)

	if file, err = getString(0, argFile, args, kwargs); err != nil {
		return err
	}
	if name, err = getString(1, argName, args, kwargs); err != nil && !errors.Is(err, errBadArgument) {
		return err
	}
	if fatal, err = getBool(2, argFatal, args, kwargs); err != nil && !errors.Is(err, errBadArgument) {
		return err
	}
	if method, err = getString(3, argMeth, args, kwargs); err != nil && !errors.Is(err, errBadArgument) {
		return err
	}
	n, err := include(file, name, fatal)
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

func include(file, name string, fatal bool) (Node, error) {
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
		if name != "" {
			obj.Name = name
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

func getStringSlice(at int, field string, args []Node, kwargs map[string]Node) ([]string, error) {
	n, err := checkHas(at+1, field, args, kwargs)
	if err != nil {
		return nil, err
	}
	arg, ok := n.(Argument)
	if ok {
		str, err := arg.GetString()
		return []string{str}, err
	}
	arr, ok := n.(*array)
	if !ok {
		return nil, fmt.Errorf("%s: node can not be used as argument", field)
	}
	var str []string
	for _, n := range arr.Nodes {
		if arg, ok = n.(Argument); !ok {
			return nil, fmt.Errorf("%s: node can not be used as argument", field)
		}
		s, err := arg.GetString()
		if err != nil {
			return nil, err
		}
		str = append(str, s)
	}
	return str, nil
}

func getString(at int, field string, args []Node, kwargs map[string]Node) (string, error) {
	n, err := checkHas(at+1, field, args, kwargs)
	if err != nil {
		return "", err
	}
	arg, ok := n.(Argument)
	if !ok {
		return "", fmt.Errorf("%s: node can not be used as argument", field)
	}
	return arg.GetString()
}

func getBool(at int, field string, args []Node, kwargs map[string]Node) (bool, error) {
	n, err := checkHas(at+1, field, args, kwargs)
	if err != nil {
		return false, err
	}
	arg, ok := n.(Argument)
	if !ok {
		return false, fmt.Errorf("%s: node can not be used argument", field)
	}
	return arg.GetBool()
}

func getInt(at int, field string, args []Node, kwargs map[string]Node) (int64, error) {
	n, err := checkHas(at+1, field, args, kwargs)
	if err != nil {
		return 0, err
	}
	arg, ok := n.(Argument)
	if !ok {
		return 0, fmt.Errorf("%s: node can not be used argument", field)
	}
	return arg.GetInt()
}

func checkHas(at int, field string, args []Node, kwargs map[string]Node) (Node, error) {
	n, ok := kwargs[field]
	if len(args) < at && !ok {
		return nil, errArgument(field, "not supplied")
	}
	if len(args) >= at {
		if ok {
			return nil, errArgument(field, "given as positional and keyword")
		}
		n = args[at-1]
	}
	return n, nil
}

func errArgument(field, msg string) error {
	return fmt.Errorf("%s: %w %s", field, errBadArgument, msg)
}
