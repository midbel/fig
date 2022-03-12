package fig

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type macroFunc func(root, obj Node, env *Env, args []Node, kwargs map[string]Node) error

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
	argAs     = "as"
	argFields = "fields"
	argDepth  = "depth"
	argCount  = "count"
	argKey    = "key"
	argCmd    = "command"
)

func Script(root, _ Node, env *Env, args []Node, kwargs map[string]Node) error {
	var (
		mcall = callMacro(root, env)
		key   string
		cmd   string
		err   error
	)
	if key, err = mcall.GetString(0, argKey, args, kwargs); err != nil {
		return err
	}
	if cmd, err = mcall.GetString(1, argCmd, args, kwargs); err != nil {
		return err
	}
	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		return err
	}
	obj, ok := root.(*object)
	if !ok {
		return fmt.Errorf("root should be an object! got %T", root)
	}
	opt := createOption(key, createLiteralFromString(string(out)))
	return obj.set(opt)
}

func Register(root, _ Node, env *Env, args []Node, kwargs map[string]Node) error {
	if len(kwargs) > 0 {
		return fmt.Errorf("register does not accept keyword arguments")
	}
	if len(args) != 2 {
		return fmt.Errorf("register: wrong number of arguments given")
	}
	obj, ok := root.(*object)
	if !ok {
		return fmt.Errorf("root should be an object! got %T", root)
	}
	var (
		mcall = callMacro(root, env)
		ident string
		value string
		err   error
	)
	if ident, err = mcall.GetString(0, "", args, kwargs); err != nil {
		return err
	}
	if value, err = mcall.GetString(1, "", args, kwargs); err != nil {
		return err
	}
	obj.register(ident, value)
	return nil
}

func IfDef(root, nest Node, env *Env, args []Node, kwargs map[string]Node) error {
	if len(kwargs) > 0 {
		return fmt.Errorf("ifdef does not accept keyword arguments")
	}
	if len(args) != 1 {
		return fmt.Errorf("ifdef: wrong number of arguments given")
	}
	mcall := callMacro(root, env)
	if mcall.IsDefined(args[0]) {
		root, ok := root.(*object)
		if !ok {
			return nil
		}
		return root.merge(nest)
	}
	return nil
}

func IfNotDef(root, nest Node, env *Env, args []Node, kwargs map[string]Node) error {
	if len(kwargs) > 0 {
		return fmt.Errorf("ifndef does not accept keyword arguments")
	}
	if len(args) != 1 {
		return fmt.Errorf("ifndef: wrong number of arguments given")
	}
	mcall := callMacro(root, env)
	if mcall.IsNotDefined(args[0]) {
		root, ok := root.(*object)
		if !ok {
			return nil
		}
		return root.merge(nest)
	}
	return nil
}

func IfEq(root, nest Node, env *Env, args []Node, kwargs map[string]Node) error {
	if len(kwargs) > 0 {
		return fmt.Errorf("ifeq does not accept keyword arguments")
	}
	if len(args) <= 1 {
		return fmt.Errorf("ifeq: not enough argument given")
	}
	var (
		mcall    = callMacro(root, env)
		str, err = mcall.GetString(0, "", args, kwargs)
	)
	if err != nil {
		return err
	}
	ok := mcall.Compare(str, args[1:], func(s1, s2 string) bool {
		return s1 == s2
	})
	if root, ok1 := root.(*object); ok1 && ok {
		return root.merge(nest)
	}
	return nil
}

func IfNotEq(root, nest Node, env *Env, args []Node, kwargs map[string]Node) error {
	if len(kwargs) > 0 {
		return fmt.Errorf("ifneq does not accept keyword arguments")
	}
	if len(args) <= 1 {
		return fmt.Errorf("ifneq: not enough argument given")
	}
	var (
		mcall    = callMacro(root, env)
		str, err = mcall.GetString(0, "", args, kwargs)
	)
	if err != nil {
		return err
	}
	ok := mcall.Compare(str, args[1:], func(s1, s2 string) bool {
		return s1 != s2
	})
	if root, ok1 := root.(*object); ok1 && ok {
		return root.merge(nest)
	}
	return nil
}

func ReadFile(root, _ Node, env *Env, args []Node, kwargs map[string]Node) error {
	if len(args) == 0 && len(kwargs) == 0 {
		return fmt.Errorf("no enough arguments supplied")
	}
	var (
		mcall = callMacro(root, env)
		name  string
		file  string
		err   error
	)
	if file, err = mcall.GetString(0, argFile, args, kwargs); err != nil {
		return err
	}
	if name, err = mcall.GetString(1, argName, args, kwargs); err != nil && !errors.Is(err, errBadArgument) {
		return err
	}
	if name == "" {
		name = filepath.Base(file)
		for ext := filepath.Ext(name); ext != ""; ext = filepath.Ext(name) {
			name = strings.TrimSuffix(name, ext)
		}
	}
	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	obj, ok := root.(*object)
	if !ok {
		return fmt.Errorf("root should be an object! got %T", root)
	}
	opt := createOption(name, createLiteralFromString(string(content)))
	return obj.set(opt)
}

func Repeat(root, nest Node, env *Env, args []Node, kwargs map[string]Node) error {
	if len(args) == 0 && len(kwargs) == 0 {
		return fmt.Errorf("no enough arguments supplied")
	}
	var (
		mcall = callMacro(root, env)
		count int64
		name  string
		err   error
	)
	if count, err = mcall.GetInt(0, argCount, args, kwargs); err != nil {
		return err
	}
	if name, err = mcall.GetString(1, argName, args, kwargs); err != nil {
		return err
	}
	obj, ok := root.(*object)
	if !ok {
		return fmt.Errorf("root should be an object! got %T", root)
	}
	return obj.repeat(count, name, nest)
}

func Extend(root, nest Node, env *Env, args []Node, kwargs map[string]Node) error {
	if len(args) == 0 && len(kwargs) == 0 {
		return fmt.Errorf("no enough arguments supplied")
	}
	var (
		mcall = callMacro(root, env)
		name  string
		as    string
		err   error
	)
	if name, err = mcall.GetString(0, argName, args, kwargs); err != nil {
		return err
	}
	if as, err = mcall.GetString(1, argAs, args, kwargs); err != nil && !errors.Is(err, errBadArgument) {
		return err
	}
	obj, ok := root.(*object)
	if !ok {
		return fmt.Errorf("root should be an object! got %T", root)
	}
	return obj.extend(name, as, nest)
}

func Define(root, nest Node, env *Env, args []Node, kwargs map[string]Node) error {
	if len(args) == 0 && len(kwargs) == 0 {
		return fmt.Errorf("no enough arguments supplied")
	}
	var (
		mcall  = callMacro(root, env)
		name   string
		method string
		err    error
	)
	if name, err = mcall.GetString(0, argName, args, kwargs); err != nil {
		return err
	}
	if method, err = mcall.GetString(1, argMeth, args, kwargs); err != nil && !errors.Is(err, errBadArgument) {
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

func Apply(root, _ Node, env *Env, args []Node, kwargs map[string]Node) error {
	if len(args) == 0 && len(kwargs) == 0 {
		return fmt.Errorf("no enough arguments supplied")
	}
	var (
		mcall  = callMacro(root, env)
		name   string
		fields []string
		depth  int64
		method string
		err    error
	)
	if name, err = mcall.GetString(0, argName, args, kwargs); err != nil {
		return err
	}
	if fields, err = mcall.GetStringArray(1, argFields, args, kwargs); err != nil && !errors.Is(err, errBadArgument) {
		return err
	}
	if depth, err = mcall.GetInt(2, argDepth, args, kwargs); err != nil && !errors.Is(err, errBadArgument) {
		return err
	}
	if method, err = mcall.GetString(3, argMeth, args, kwargs); err != nil && !errors.Is(err, errBadArgument) {
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

func Include(root, _ Node, env *Env, args []Node, kwargs map[string]Node) error {
	var (
		mcall  = callMacro(root, env)
		file   string
		name   string
		method string
		fatal  bool
		err    error
	)

	if file, err = mcall.GetString(0, argFile, args, kwargs); err != nil {
		return err
	}
	if name, err = mcall.GetString(1, argName, args, kwargs); err != nil && !errors.Is(err, errBadArgument) {
		return err
	}
	if fatal, err = mcall.GetBool(2, argFatal, args, kwargs); err != nil && !errors.Is(err, errBadArgument) {
		return err
	}
	if method, err = mcall.GetString(3, argMeth, args, kwargs); err != nil && !errors.Is(err, errBadArgument) {
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

type macrocall struct {
	args   []Node
	kwargs map[string]Node
	root   Node
	env    *Env
}

func callMacro(root Node, env *Env) macrocall {
	return macrocall{
		root: root,
		env:  env,
	}
}

func (c macrocall) IsDefined(n Node) bool {
	v, ok := n.(*variable)
	if !ok {
		return ok
	}
	obj, ok := c.root.(*object)
	if !ok {
		return ok
	}
	var err error
	if v.IsLocal() {
		_, err = obj.resolve(v.Name())
	} else {
		_, err = c.env.resolve(v.Name())
	}
	return err == nil
}

func (c macrocall) IsNotDefined(n Node) bool {
	v, ok := n.(*variable)
	if !ok {
		return ok
	}
	obj, ok := c.root.(*object)
	if !ok {
		return ok
	}
	var err error
	if v.IsLocal() {
		_, err = obj.resolve(v.Name())
	} else {
		_, err = c.env.resolve(v.Name())
	}
	return err != nil
}

func (c macrocall) Compare(str string, args []Node, cmp func(string, string) bool) bool {
	for _, other := range args {
		other, _ := c.getString(other, str)
		if cmp(str, other) {
			return true
		}
	}
	return false
}

func (c macrocall) GetStringArray(at int, field string, args []Node, kwargs map[string]Node) ([]string, error) {
	n, err := checkHas(at+1, field, args, kwargs)
	if err != nil {
		return nil, err
	}
	arg, ok := n.(Argument)
	if ok {
		str, err := arg.GetString()
		return []string{str}, err
	}
	arr, err := c.getArray(n, field)
	if err != nil {
		return nil, err
	}
	var str []string
	for _, n := range arr.Nodes {
		s, err := c.getString(n, field)
		if err != nil {
			return nil, fmt.Errorf("%s: node can not be used as argument", field)
		}
		str = append(str, s)
	}
	return str, nil
}

func (c macrocall) GetString(at int, field string, args []Node, kwargs map[string]Node) (string, error) {
	n, err := checkHas(at+1, field, args, kwargs)
	if err != nil {
		return "", err
	}
	return c.getString(n, field)
}

func (c macrocall) GetInt(at int, field string, args []Node, kwargs map[string]Node) (int64, error) {
	n, err := checkHas(at+1, field, args, kwargs)
	if err != nil {
		return 0, err
	}
	return c.getInt(n, field)
}

func (c macrocall) GetBool(at int, field string, args []Node, kwargs map[string]Node) (bool, error) {
	n, err := checkHas(at+1, field, args, kwargs)
	if err != nil {
		return false, err
	}
	return c.getBool(n, field)
}

func (c macrocall) getBool(n Node, field string) (bool, error) {
	arg, ok := n.(Argument)
	if ok {
		return arg.GetBool()
	}
	b, ok := tryBoolFromVar(n, c.env, c.root)
	if !ok {
		return false, fmt.Errorf("%s: node can not be used argument", field)
	}
	return b, nil
}

func (c macrocall) getString(n Node, field string) (string, error) {
	arg, ok := n.(Argument)
	if ok {
		return arg.GetString()
	}
	str, ok := tryStringFromVar(n, c.env, c.root)
	if !ok {
		return "", fmt.Errorf("%s: node can not be used as argument", field)
	}
	return str, nil
}

func (c macrocall) getInt(n Node, field string) (int64, error) {
	arg, ok := n.(Argument)
	if ok {
		return arg.GetInt()
	}
	num, ok := tryIntFromVar(n, c.env, c.root)
	if !ok {
		return 0, fmt.Errorf("%s: node can not be used argument", field)
	}
	return num, nil
}

func (c macrocall) getArray(n Node, field string) (*array, error) {
	arr, ok := n.(*array)
	if ok {
		return arr, nil
	}
	arr, ok = tryFromVarArray(n, c.env, c.root)
	if !ok {
		return nil, fmt.Errorf("%s: node can not be used as argument", field)
	}
	return arr, nil
}

func tryBoolFromVar(n Node, env *Env, root Node) (bool, bool) {
	val, ok := tryFromVar(n, env, root)
	if !ok {
		return ok, ok
	}
	var b bool
	switch val := val.(type) {
	default:
		return false, false
	case string:
		x, err := strconv.ParseBool(val)
		if err != nil {
			return false, false
		}
		b = x
	case int64:
		b = val != 0
	case bool:
		b = val
	}
	return b, true
}

func tryIntFromVar(n Node, env *Env, root Node) (int64, bool) {
	val, ok := tryFromVar(n, env, root)
	if !ok {
		return 0, ok
	}
	var num int64
	switch val := val.(type) {
	default:
		return 0, false
	case string:
		n, err := strconv.ParseInt(val, 0, 64)
		if err != nil {
			return 0, false
		}
		num = n
	case int64:
		num = val
	case bool:
		if val {
			num = 1
		}
	}
	return num, ok
}

func tryStringFromVar(n Node, env *Env, root Node) (string, bool) {
	val, ok := tryFromVar(n, env, root)
	if !ok {
		return "", ok
	}
	var str string
	switch val := val.(type) {
	default:
		return "", false
	case string:
		str = val
	case int64:
		str = strconv.FormatInt(val, 10)
	case bool:
		str = strconv.FormatBool(val)
	}
	return str, true
}

func tryFromVar(n Node, env *Env, root Node) (interface{}, bool) {
	v, ok := n.(*variable)
	if !ok {
		return nil, ok
	}
	var val interface{}
	if v.IsLocal() {
		val, ok = resolveFromNode(root, v.Name())
	} else {
		v, err := env.resolve(v.Name())
		if err != nil {
			return 0, false
		}
		val = v
	}
	return val, ok
}

func tryFromVarArray(n Node, env *Env, root Node) (*array, bool) {
	v, ok := n.(*variable)
	if !ok {
		return nil, ok
	}
	obj, ok := root.(*object)
	if !ok {
		return nil, false
	}
	for _, n := range obj.Nodes {
		opt, ok := n.(*option)
		if !ok {
			continue
		}
		if opt.Ident == v.Name() {
			arr, ok := opt.Value.(*array)
			return arr, ok
		}
	}
	return nil, false
}

func resolveFromNode(node Node, ident string) (interface{}, bool) {
	n, ok := node.(*object)
	if !ok {
		return nil, ok
	}
	v, err := n.resolve(ident)
	if err != nil {
		return nil, false
	}
	return v, true
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
	if field == "" {
		field = "node"
	}
	return fmt.Errorf("%s: %w %s", field, errBadArgument, msg)
}
