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
	default:
		return fmt.Errorf("primitive type expected! got %s", k)
	}
	return err
}

func decodeOption(opt *option, v reflect.Value) error {
	switch opt.Value.Type() {
	case TypeLiteral:
		lit, _ := opt.getLiteral()
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
	case reflect.Struct:
	case reflect.Slice, reflect.Array:
		n := reflect.New(v.Type().Elem()).Elem()
		if err := decodeObject(obj, n); err != nil {
			return err
		}
		v.Set(reflect.Append(v, n))
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
