package fig

import (
	"fmt"
	"io"
	"reflect"
	"strings"
)

type FuncMap map[string]interface{}

type Decoder struct {
	read io.Reader
	set  FuncMap
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		read: r,
		set:  make(FuncMap),
	}
}

func (d *Decoder) Funcs(set FuncMap) {
	for k, v := range set {
		d.set[k] = v
	}
}

func (d *Decoder) Decode(v interface{}) error {
	n, err := Parse(d.read)
	if err != nil {
		return err
	}
	value := reflect.ValueOf(v)
	if value.Kind() != reflect.Ptr || value.IsNil() {
		return fmt.Errorf("expecting not nil ptr")
	}
	if value.Kind() == reflect.Interface && value.NumMethod() == 0 {
		var (
			m = make(map[string]interface{})
			v = reflect.ValueOf(m).Elem()
		)
		obj, ok := n.(*object)
		if !ok {
			return fmt.Errorf("root node is not an object")
		}
		if err = d.decodeMap(obj, v); err == nil {
			value.Set(v)
		}
		return err
	}
	if value.Kind() == reflect.Map {
		if value.IsNil() {
			m := make(map[string]interface{})
			value.Set(reflect.ValueOf(m))
		}
		obj, ok := n.(*object)
		if !ok {
			return fmt.Errorf("root node is not an object")
		}
		return d.decodeMap(obj, value)
	}
	return d.decode(n, value.Elem())
}

func (d *Decoder) decode(n Node, value reflect.Value) error {
	var err error
	switch n := n.(type) {
	case *array:
		err = d.decodeArray(n, value)
	case *object:
		err = d.decodeObject(n, value)
	case *option:
		err = d.decodeOption(n, value)
	case *literal:
		err = d.decodeLiteral(n, value)
	default:
		err = fmt.Errorf("value (%s) can not be decoded from %T", value.Kind(), n)
	}
	return err
}

func (d *Decoder) decodeInterface(lit *literal, v reflect.Value) error {
	var (
		val reflect.Value
		err error
	)
	switch lit.Token.Type {
	case String, Heredoc, Ident:
		s, err1 := lit.GetString()
		if err1 != nil {
			err = err1
			break
		}
		val = reflect.ValueOf(s)
	case Boolean:
		b, err1 := lit.GetBool()
		if err1 != nil {
			err = err1
			break
		}
		val = reflect.ValueOf(b)
	case Integer:
		i, err1 := lit.GetInt()
		if err1 != nil {
			err = err1
			break
		}
		val = reflect.ValueOf(i)
	case Float:
		f, err1 := lit.GetFloat()
		if err1 != nil {
			err = err1
			break
		}
		val = reflect.ValueOf(f)
	default:
		if v.Kind() == reflect.Interface {
			i, err1 := lit.Get()
			if err1 != nil {
				err = err1
				break
			}
			v.Set(reflect.ValueOf(i))
		}
		err = fmt.Errorf("primitive type expected!")
	}
	if err == nil {
		v.Set(val)
	}
	return err
}

func (d *Decoder) decodeLiteral(lit *literal, v reflect.Value) error {
	var err error
	switch k := v.Kind(); k {
	case reflect.String:
		s, err1 := lit.GetString()
		if err1 != nil {
			err = err1
			break
		}
		v.SetString(s)
	case reflect.Bool:
		b, err1 := lit.GetBool()
		if err1 != nil {
			err = err1
			break
		}
		v.SetBool(b)
	case reflect.Float32, reflect.Float64:
		f, err1 := lit.GetFloat()
		if err1 != nil {
			err = err1
			break
		}
		v.SetFloat(f)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err1 := lit.GetInt()
		if err1 != nil {
			err = err1
			break
		}
		v.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, err1 := lit.GetUint()
		if err1 != nil {
			err = err1
			break
		}
		v.SetUint(i)
	case reflect.Interface:
		i, err1 := lit.Get()
		if err1 != nil {
			err = err1
			break
		}
		v.Set(reflect.ValueOf(i))
	default:
		return fmt.Errorf("primitive type expected! got %s", k)
	}
	return err
}

func (d *Decoder) decodeOption(opt *option, v reflect.Value) error {
	switch opt.Value.Type() {
	case TypeLiteral:
		lit, _ := opt.getLiteral()
		if k := v.Kind(); k == reflect.Interface {
			return d.decodeInterface(lit, v)
		}
		return d.decodeLiteral(lit, v)
	case TypeArray:
		return d.decode(opt.Value, v)
	case TypeCall:
		return d.decodeCall(opt.Value.(*call), v)
	default:
		return fmt.Errorf("literal/array/slice expected!")
	}
}

var errtype = reflect.TypeOf((*error)(nil)).Elem()

func (d *Decoder) decodeCall(c *call, v reflect.Value) error {
	call := reflect.ValueOf(d.set[c.Ident])
	if call.Kind() != reflect.Func {
		return fmt.Errorf("%s: undefined function", c.Ident)
	}
	var (
		typ  = call.Type()
		nin  = typ.NumIn()
		nout = typ.NumOut()
		args []reflect.Value
	)
	if nout == 0 || nout > 2 || nin != len(c.Args) {
		return fmt.Errorf("%s: invalid function signature ", c.Ident)
	}
	for i := 0; i < nin; i++ {
		f := reflect.New(typ.In(i)).Elem()
		if err := d.decode(c.Args[i], f); err != nil {
			return err
		}
		args = append(args, f)
	}
	ret := call.Call(args)
	if len(ret) == 2 {
		if ret[1].Type() != errtype {
			return fmt.Errorf("return value should be of type error")
		}
		err, _ := ret[1].Interface().(error)
		if err != nil {
			return err
		}
	}
	if !ret[0].Type().AssignableTo(v.Type()) && !ret[0].Type().ConvertibleTo(v.Type()) {
		return fmt.Errorf("return value can not be assigned to %s", v.Type())
	}
	v.Set(ret[0].Convert(v.Type()))
	return nil
}

func (d *Decoder) decodeArray(arr *array, v reflect.Value) error {
	if k := v.Kind(); k != reflect.Slice && k != reflect.Array {
		return fmt.Errorf("slice/array type expected! got %s", k)
	}
	vs := reflect.MakeSlice(v.Type(), 0, v.Len())
	for _, n := range arr.Nodes {
		vf := reflect.New(v.Type().Elem()).Elem()
		if err := d.decode(n, vf); err != nil {
			return err
		}
		vs = reflect.Append(vs, vf)
	}
	v.Set(vs)
	return nil
}

func (d *Decoder) decodeObject(obj *object, v reflect.Value) error {
	switch k := v.Kind(); k {
	case reflect.Struct:
	case reflect.Map:
		return d.decodeMap(obj, v)
	case reflect.Slice, reflect.Array:
		n := reflect.New(v.Type().Elem()).Elem()
		if err := d.decodeObject(obj, n); err != nil {
			return err
		}
		v.Set(reflect.Append(v, n))
		return nil
	case reflect.Ptr:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return d.decodeObject(obj, v.Elem())
	case reflect.Interface:
		m := reflect.ValueOf(make(map[string]interface{}))
		if err := d.decodeMap(obj, m); err != nil {
			return err
		}
		v.Set(m)
		return nil
	default:
		return fmt.Errorf("struct/slice/array type expected! got %s", v.Kind())
	}
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if !f.CanSet() {
			continue
		}
		var (
			ft  = t.Field(i)
			tag = ft.Name
		)
		switch v := ft.Tag.Get("fig"); v {
		case "":
		case "-":
			continue
		default:
			tag = v
		}
		node, ok := obj.Props[tag]
		if !ok && tag == ft.Name {
			node, ok = obj.Props[strings.ToLower(tag)]
			if !ok {
				continue
			}
		}
		if err := d.decode(node, f); err != nil {
			return err
		}
	}
	return nil
}

func (d *Decoder) decodeMap(obj *object, v reflect.Value) error {
	key := v.Type().Key()
	if k := key.Kind(); k != reflect.String {
		return fmt.Errorf("key should be of type string")
	}
	if v.IsNil() {
		v.Set(reflect.MakeMap(v.Type()))
	}
	for k, o := range obj.Props {
		var (
			vf  reflect.Value
			err error
		)
		switch o := o.(type) {
		case *object:
			vf = reflect.MakeMap(v.Type())
			err = d.decodeMap(o, vf)
		case *option:
			vf = reflect.New(v.Type().Elem()).Elem()
			err = d.decodeOption(o, vf)
		case *array:
			var (
				s = reflect.SliceOf(v.Type().Elem())
				f = reflect.MakeSlice(s, 0, len(o.Nodes))
			)
			vf = reflect.New(f.Type()).Elem()
			err = d.decodeArray(o, vf)
		default:
			err = fmt.Errorf("%s: can not decode %T", k, o)
		}
		if err != nil {
			return err
		}
		v.SetMapIndex(reflect.ValueOf(k), vf)
	}
	return nil
}
