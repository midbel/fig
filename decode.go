package fig

import (
	"fmt"
	"io"
	"reflect"
	"strings"
)

func Decode(r io.Reader, v interface{}) error {
	n, err := Parse(r)
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
		if err = decodeMap(obj, v); err == nil {
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
		return decodeMap(obj, value)
	}
	return decode(n, value.Elem())
}

func decode(n Node, value reflect.Value) error {
	var err error
	switch n := n.(type) {
	case *array:
		err = decodeArray(n, value)
	case *object:
		err = decodeObject(n, value)
	case *option:
		err = decodeOption(n, value)
	case *literal:
		err = decodeLiteral(n, value)
	default:
		err = fmt.Errorf("value (%s) can not be decoded from %T", value.Kind(), n)
	}
	return err
}

func decodeInterface(lit *literal, v reflect.Value) error {
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

func decodeLiteral(lit *literal, v reflect.Value) error {
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

func decodeOption(opt *option, v reflect.Value) error {
	switch opt.Value.Type() {
	case TypeLiteral:
		lit, _ := opt.getLiteral()
		if k := v.Kind(); k == reflect.Interface {
			return decodeInterface(lit, v)
		}
		return decodeLiteral(lit, v)
	case TypeArray:
		return decode(opt.Value, v)
	default:
		return fmt.Errorf("literal/array/slice expected!")
	}
}

func decodeArray(arr *array, v reflect.Value) error {
	if k := v.Kind(); k != reflect.Slice && k != reflect.Array {
		return fmt.Errorf("slice/array type expected! got %s", k)
	}
	vs := reflect.MakeSlice(v.Type(), 0, v.Len())
	for _, n := range arr.Nodes {
		vf := reflect.New(v.Type().Elem()).Elem()
		if err := decode(n, vf); err != nil {
			return err
		}
		vs = reflect.Append(vs, vf)
	}
	v.Set(vs)
	return nil
}

func decodeObject(obj *object, v reflect.Value) error {
	switch k := v.Kind(); k {
	case reflect.Map:
		return decodeMap(obj, v)
	case reflect.Struct:
	case reflect.Slice, reflect.Array:
		n := reflect.New(v.Type().Elem()).Elem()
		if err := decodeObject(obj, n); err != nil {
			return err
		}
		v.Set(reflect.Append(v, n))
		return nil
	case reflect.Ptr:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return decodeObject(obj, v.Elem())
	case reflect.Interface:
		m := reflect.ValueOf(make(map[string]interface{}))
		if err := decodeMap(obj, m); err != nil {
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
		if err := decode(node, f); err != nil {
			return err
		}
	}
	return nil
}

func decodeMap(obj *object, v reflect.Value) error {
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
			err = decodeMap(o, vf)
		case *option:
			vf = reflect.New(v.Type().Elem()).Elem()
			err = decodeOption(o, vf)
		case *array:
			var (
				s = reflect.SliceOf(v.Type().Elem())
				f = reflect.MakeSlice(s, 0, len(o.Nodes))
			)
			vf = reflect.New(f.Type()).Elem()
			err = decodeArray(o, vf)
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
