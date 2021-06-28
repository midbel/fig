package fig

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"
)

type Setter interface {
	Set(interface{}) error
}

type Document struct {
	root *Object
	env  Environment
}

func Decode(r io.Reader, v interface{}) error {
	doc, err := ParseDocument(r)
	if err != nil {
		return err
	}
	return doc.Decode(v)
}

func ParseDocument(r io.Reader) (*Document, error) {
	return ParseDocumentWithEnv(r, EmptyEnv())
}

func ParseDocumentWithEnv(r io.Reader, env Environment) (*Document, error) {
	root, err := Parse(r)
	if err != nil {
		return nil, err
	}
	return createDocument(root, env), nil
}

func createDocument(root *Object, env Environment) *Document {
	return &Document{
		root: root,
		env:  env,
	}
}

func (d *Document) DefineInt(str string, i int64) {
	d.env.Define(str, makeInt(i))
}

func (d *Document) DefineBool(str string, b bool) {
	d.env.Define(str, makeBool(b))
}

func (d *Document) DefineDouble(str string, f float64) {
	d.env.Define(str, makeDouble(f))
}

func (d *Document) DefineText(str string, t string) {
	d.env.Define(str, makeText(str))
}

func (d *Document) DefineTime(str string, t time.Time) {
	d.env.Define(str, makeMoment(t))
}

func (d *Document) Int(paths ...string) (int64, error) {
	v, err := d.eval(paths)
	if err != nil {
		return 0, err
	}
	i, err := v.toInt()
	if err != nil {
		return 0, err
	}
	return toInt(i)
}

func (d *Document) IntArray(paths ...string) ([]int64, error) {
	v, err := d.eval(paths)
	if err != nil {
		return nil, err
	}
	s, ok := v.(Slice)
	if !ok {
		i, err := toInt(v)
		if err != nil {
			return nil, err
		}
		return []int64{i}, nil
	}
	vs := make([]int64, len(s.inner))
	for i := range s.inner {
		vs[i], err = toInt(s.inner[i])
		if err != nil {
			return nil, err
		}
	}
	return vs, nil
}

func (d *Document) Float(paths ...string) (float64, error) {
	v, err := d.eval(paths)
	if err != nil {
		return 0, err
	}
	f, err := v.toDouble()
	if err != nil {
		return 0, err
	}
	return toFloat(f)
}

func (d *Document) FloatArray(paths ...string) ([]float64, error) {
	v, err := d.eval(paths)
	if err != nil {
		return nil, err
	}
	s, ok := v.(Slice)
	if !ok {
		i, err := toFloat(v)
		if err != nil {
			return nil, err
		}
		return []float64{i}, nil
	}
	vs := make([]float64, len(s.inner))
	for i := range s.inner {
		vs[i], err = toFloat(s.inner[i])
		if err != nil {
			return nil, err
		}
	}
	return vs, nil
}

func (d *Document) Bool(paths ...string) (bool, error) {
	v, err := d.eval(paths)
	if err != nil {
		return false, err
	}
	return v.isTrue(), nil
}

func (d *Document) BoolArray(paths ...string) ([]bool, error) {
	v, err := d.eval(paths)
	if err != nil {
		return nil, err
	}
	s, ok := v.(Slice)
	if !ok {
		return []bool{v.isTrue()}, nil
	}
	vs := make([]bool, len(s.inner))
	for i := range s.inner {
		vs[i] = s.inner[i].isTrue()
	}
	return vs, nil
}

func (d *Document) Time(paths ...string) (time.Time, error) {
	v, err := d.eval(paths)
	if err != nil {
		return time.Time{}, err
	}
	t, err := v.toMoment()
	if err != nil {
		return time.Time{}, err
	}
	return toTime(t)
}

func (d *Document) TimeArray(paths ...string) ([]time.Time, error) {
	v, err := d.eval(paths)
	if err != nil {
		return nil, err
	}
	s, ok := v.(Slice)
	if !ok {
		i, err := toTime(v)
		if err != nil {
			return nil, err
		}
		return []time.Time{i}, nil
	}
	vs := make([]time.Time, len(s.inner))
	for i := range s.inner {
		vs[i], err = toTime(s.inner[i])
		if err != nil {
			return nil, err
		}
	}
	return vs, nil
}

func (d *Document) Text(paths ...string) (string, error) {
	v, err := d.eval(paths)
	if err != nil {
		return "", err
	}
	t, err := v.toText()
	if err != nil {
		return "", err
	}
	return toText(t)
}

func (d *Document) TextArray(paths ...string) ([]string, error) {
	v, err := d.eval(paths)
	if err != nil {
		return nil, err
	}
	s, ok := v.(Slice)
	if !ok {
		i, err := toText(v)
		if err != nil {
			return nil, err
		}
		return []string{i}, nil
	}
	vs := make([]string, len(s.inner))
	for i := range s.inner {
		vs[i], err = toText(s.inner[i])
		if err != nil {
			return nil, err
		}
	}
	return vs, nil
}

func (d *Document) Value(paths ...string) (interface{}, error) {
	v, err := d.eval(paths)
	if err != nil {
		return nil, err
	}
	return toInterface(v), nil
}

func (d *Document) Slice(paths ...string) ([]interface{}, error) {
	_, err := d.eval(paths)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (d *Document) Document(paths ...string) (*Document, error) {
	var n = d.root
	for i := 0; i < len(paths); i++ {
		obj, ok := n.nodes[paths[i]]
		if !ok {
			return nil, fmt.Errorf("%s: object not found", paths[i])
		}
		n, ok = obj.(*Object)
		if !ok {
			return nil, fmt.Errorf("%s: not an object", paths[i])
		}
	}
	doc := Document{
		root: n,
	}
	return &doc, nil
}

func (d *Document) Decode(v interface{}) error {
	return d.DecodeWithEnv(v, d.env)
}

func (d *Document) DecodeWithEnv(v interface{}, env Environment) error {
	old := d.env
	defer func() {
		d.env = old
	}()
	d.env = env
	return d.decode(reflect.ValueOf(v).Elem(), d.root)
}

func (d *Document) eval(paths []string) (Value, error) {
	rs, err := d.find(paths)
	if err != nil {
		return nil, err
	}
	if len(rs) == 0 {
		return nil, fmt.Errorf("no result match")
	}
	var arr []Value
	for _, r := range rs {
		v, err := r.Eval(d.env)
		if err != nil {
			return nil, err
		}
		arr = append(arr, v)
	}
	if len(arr) == 1 {
		return arr[0], nil
	}
	return makeSlice(arr), nil
}

func (d *Document) find(paths []string) ([]result, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("empty path!")
	}
	list := []*Object{d.root.copy()}
	return findExpr(d.root, list, paths)
}

type result struct {
	Expr
	List []*Object
}

func makeResult(e Expr, list []*Object) result {
	return result{
		Expr: e,
		List: list,
	}
}

func (r result) Eval(e Environment) (Value, error) {
	return r.Expr.Eval(createEnv(reverseList(r.List), e))
}

func findExpr(root *Object, list []*Object, paths []string) ([]result, error) {
	var err error
	for i := 0; i < len(paths)-1; i++ {
		n, err := root.getNode(paths[i])
		if err != nil {
			return nil, err
		}
		switch n := n.(type) {
		case List:
			var rs []result
			for _, n := range n.nodes {
				obj, ok := n.(*Object)
				if !ok {
					continue
				}
				r, err := findExpr(obj, list, paths[i+1:])
				if err != nil {
					return nil, err
				}
				rs = append(rs, r...)
			}
			return rs, nil
		case *Object:
			root = n
		default:
			return nil, fmt.Errorf("unexpected node type %T", n)
		}
		list = append(list, root.copy())
	}
	n, err := root.getNode(paths[len(paths)-1])
	if err != nil {
		return nil, err
	}
	if len(list) > 1 {
		list[len(list)-1].unregister(paths[len(paths)-1])
	}

	var rs []result
	switch n := n.(type) {
	case Option:
		rs = append(rs, makeResult(n.expr, list))
	case List:
		for _, n := range n.nodes {
			o, ok := n.(Option)
			if !ok {
				return nil, fmt.Errorf("unexpected node type %T", n)
			}
			rs = append(rs, makeResult(o.expr, list))
		}
	default:
		return nil, fmt.Errorf("unexpected node type %T", n)
	}
	return rs, nil
}

func (d *Document) decode(v reflect.Value, root *Object) error {
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("unexpected value type %s - struct expected", v.Kind())
	}
	var (
		typ = v.Type()
		key string
	)
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if !f.CanSet() {
			continue
		}
		var (
			yf = typ.Field(i)
			required bool
		)
		switch key = yf.Tag.Get("fig"); key {
		case "":
			key = strings.ToLower(yf.Name)
		case "-":
			continue
		default:
			parts := strings.Split(key, ",")
			key = strings.TrimSpace(parts[0])
			if len(parts) > 1 {
				required = strings.TrimSpace(parts[1]) == "required"
			}
		}
		n, err := root.getNode(key)
		if err != nil {
			if errors.Is(err, ErrUndefined) && !required {
				continue
			}
			return err
		}
		if err := d.decodeNode(f, n); err != nil {
			return fmt.Errorf("fail to decode %s: %v", key, err)
		}
	}
	return nil
}

func (d *Document) decodeNode(v reflect.Value, n Node) error {
	var err error
	switch n := n.(type) {
	case *Object:
		err = d.decode(v, n)
	case Option:
		err = d.decodeOption(v, n)
	case List:
		err = d.decodeList(v, n)
	default:
		err = fmt.Errorf("%T: can not decode node type", n)
	}
	return err
}

func (d *Document) decodeList(v reflect.Value, list List) error {
	if len(list.nodes) == 0 {
		return nil
	}
	if k := v.Kind(); k != reflect.Array && k != reflect.Slice && isSetter(v) {
		value, err := list.Eval(d.env)
		if err == nil {
			_, err = d.decodeSetter(v, value)
		}
		return err
	}
	var (
		err error
		typ = v.Type().Elem()
		vs = reflect.MakeSlice(reflect.SliceOf(typ), 0, 10)
	)
	switch {
	case isOption(list.nodes[0]) == nil:
		for _, n := range list.nodes {
			el := reflect.New(typ).Elem()
			if err = d.decodeOption(el, n.(Option)); err != nil {
				break
			}
			vs = reflect.Append(vs, el)
		}
	case isObject(list.nodes[0]) == nil:
		for _, n := range list.nodes {
			el := reflect.New(typ).Elem()
			if err = d.decode(el, n.(*Object)); err != nil {
				break
			}
			vs = reflect.Append(vs, el)
		}
	default:
		err = fmt.Errorf("%T: can not decode list with node type", list.nodes[0])
	}
	v.Set(vs)
	return nil
}

func (d *Document) decodeOption(f reflect.Value, opt Option) error {
	value, err := opt.Eval(d.env)
	if err != nil {
		return err
	}
	if ok, err := d.decodeSetter(f, value); ok || err != nil {
		return err
	}
	var (
		v = reflect.ValueOf(value)
		t = v.Type()
		set bool
	)
	if t.AssignableTo(f.Type()) {
		f.Set(v)
		set = true
	}
	if !set && t.ConvertibleTo(f.Type()) {
		f.Set(v.Convert(f.Type()))
		set = true
	}
	if !set {
		return fmt.Errorf("%s: fail to decode option into %s", opt.name.Input, f.Type())
	}
	return nil
}

func (d *Document) decodeSetter(f reflect.Value, value interface{}) (bool, error) {
	if f.CanInterface() && f.Type().Implements(settype) {
		return true, f.Interface().(Setter).Set(value)
	}
	if f.CanAddr() {
		a := f.Addr()
		if a.CanInterface() && a.Type().Implements(settype) {
			return true, a.Interface().(Setter).Set(value)
		}
	}
	return false, nil
}

func isSetter(f reflect.Value) bool {
	if f.CanInterface() && f.Type().Implements(settype) {
		return true
	}
	if f.CanAddr() {
		return isSetter(f.Addr())
	}
	return false
}

var (
	timetype = reflect.TypeOf((*time.Time)(nil)).Elem()
	bytetype = reflect.TypeOf((*[]byte)(nil)).Elem()
	settype  = reflect.TypeOf((*Setter)(nil)).Elem()
)

func reverseList(list []*Object) []*Object {
	size := len(list)
	for i, j := 0, size-1; i < size/2; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
	return list
}
