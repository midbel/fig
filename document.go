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
	// d.env = chain(env, d.env)
	d.env = env
	return d.decode(reflect.ValueOf(v).Elem(), nil)
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

func (d *Document) list(paths []string) []*Object {
	if len(paths) == 0 {
		return nil
	}
	return findObjects(d.root, paths)
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

func findObjects(root *Object, paths []string) []*Object {
	if len(paths) == 1 {
		return []*Object{root}
	}
	var list []*Object
	for i := 0; i < len(paths)-1; i++ {
		n, err := root.getNode(paths[i])
		if err != nil {
			break
		}
		obj, ok := n.(*Object)
		if !ok {
			return nil
		}
		root = obj
	}
	n, err := root.getNode(paths[len(paths)-1])
	if err != nil {
		return nil
	}
	switch n := n.(type) {
	case *Object:
		list = append(list, n)
	case List:
		for _, n := range n.nodes {
			obj, ok := n.(*Object)
			if !ok {
				return nil
			}
			list = append(list, obj)
		}
	default:
		return nil
	}
	return list
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

func (d *Document) decodeSetter(f reflect.Value, paths []string) (bool, error) {
	if f.CanInterface() && f.Type().Implements(settype) {
		v, err := d.Value(paths...)
		if err != nil {
			return false, err
		}
		return true, f.Interface().(Setter).Set(v)
	}
	if f.CanAddr() {
		a := f.Addr()
		if a.CanInterface() && a.Type().Implements(settype) {
			v, err := d.Value(paths...)
			if err != nil {
				return false, err
			}
			return true, a.Interface().(Setter).Set(v)
		}
	}
	return false, nil
}

func (d *Document) decode(v reflect.Value, paths []string) error {
	if ok, err := d.decodeSetter(v, paths); ok || err != nil {
		return err
	}
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
			rq bool
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
				rq = strings.TrimSpace(parts[1]) == "required"
			}
		}
		if err := d.decodeInto(f, append(paths, key)); err != nil {
			if errors.Is(err, ErrUndefined) && !rq {
				continue
			}
			return err
		}
	}
	return nil
}

func (d *Document) decodeInto(f reflect.Value, paths []string) error {
	if ok, err := d.decodeSetter(f, paths); ok || err != nil {
		return err
	}
	switch f.Kind() {
	case reflect.String:
		v, err := d.Text(paths...)
		if err != nil {
			return err
		}
		f.SetString(v)
	case reflect.Bool:
		v, err := d.Bool(paths...)
		if err != nil {
			return err
		}
		f.SetBool(v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := d.Int(paths...)
		if err != nil {
			return err
		}
		f.SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := d.Int(paths...)
		if err != nil {
			return err
		}
		f.SetUint(uint64(v))
	case reflect.Float32, reflect.Float64:
		v, err := d.Float(paths...)
		if err != nil {
			return err
		}
		f.SetFloat(v)
	case reflect.Ptr:
		if f.IsNil() || !f.IsValid() {
			f.Set(reflect.New(f.Type().Elem()))
		}
		return d.decode(f.Elem(), paths)
	case reflect.Interface:
		if f.IsNil() || !f.IsValid() {
			return fmt.Errorf("invalid value given")
		}
		k := f.Elem().Kind()
		if k == reflect.Struct {
			e := reflect.New(f.Elem().Type())
			if err := d.decodeInto(e, paths); err != nil {
				return err
			}
			f.Set(reflect.Indirect(e))
			return nil
		}
		return d.decodeInto(f.Elem(), paths)
	case reflect.Struct:
		if f.Type() == timetype {
			v, err := d.Time(paths...)
			if err != nil {
				return err
			}
			f.Set(reflect.ValueOf(v))
			return nil
		}
		doc, err := d.Document(paths...)
		if err != nil {
			return err
		}
		return doc.decode(f, nil)
	case reflect.Slice:
		return d.decodeSlice(f, paths)
	case reflect.Array:
	case reflect.Map:
	default:
	}
	return nil
}

func (d *Document) decodeSlice(f reflect.Value, paths []string) error {
	typ := f.Type().Elem()
	if typ.Kind() == reflect.Struct {
		var (
			vs = reflect.MakeSlice(reflect.SliceOf(typ), 0, 10)
			el = reflect.New(typ).Elem()
		)
		for _, root := range d.list(paths) {
			doc := createDocument(root, d.env)
			if err := doc.decode(el, nil); err != nil {
				return err
			}
			vs = reflect.Append(vs, el)
		}
		f.Set(vs)
		return nil
	}
	vs, err := asSlice(d, paths)
	if err != nil {
		return err
	}
	for i := 0; i < vs.Len(); i++ {
		var (
			v  = reflect.ValueOf(vs.Index(i).Interface())
			vt = v.Type()
			cv = vt.ConvertibleTo(typ)
		)
		if !vt.AssignableTo(typ) && !cv {
			return fmt.Errorf("%s can not be assigned to %s", vt, typ)
		}
		if cv {
			v = v.Convert(typ)
		}
		f = reflect.Append(f, v)
		// f.Set(reflect.Append(f, v))
	}
	return nil
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

func asSlice(d *Document, paths []string) (reflect.Value, error) {
	v, err := d.Value(paths...)
	if err != nil {
		return reflect.Value{}, err
	}
	if k := reflect.TypeOf(v).Kind(); k != reflect.Slice {
		v = []interface{}{v}
	}
	return reflect.ValueOf(v), nil
}
