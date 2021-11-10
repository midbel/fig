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
	default:
		err = fmt.Errorf("value (%s) can not be decoded from %T", value.Kind(), n)
	}
	return err
}

func decodeOption(opt *option, v reflect.Value) error {
	var (
		value reflect.Value
		err   error
	)
	switch k := v.Kind(); k {
	case reflect.String:
		s, err1 := opt.GetString()
		if err1 != nil {
			err = err1
			break
		}
		value = reflect.ValueOf(s)
	case reflect.Bool:
		b, err1 := opt.GetBool()
		if err1 != nil {
			err = err1
			break
		}
		value = reflect.ValueOf(b)
	case reflect.Float32, reflect.Float64:
		f, err1 := opt.GetFloat()
		if err1 != nil {
			err = err1
			break
		}
		value = reflect.ValueOf(f)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err1 := opt.GetInt()
		if err1 != nil {
			err = err1
			break
		}
		value = reflect.ValueOf(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, err1 := opt.GetInt()
		if err1 != nil {
			err = err1
			break
		}
		value = reflect.ValueOf(uint64(i))
	case reflect.Slice, reflect.Array:
		arr, err := opt.getArray()
		if err != nil {
			return err
		}
		return decode(arr, v)
	default:
		return fmt.Errorf("primitive type expected! got %s", k)
	}
	if err != nil {
		return err
	}
	v.Set(value)
	return nil
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
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("struct type expected! got %s", v.Kind())
	}
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if !f.CanSet() {
			continue
		}
		var (
			ft  = t.Field(i)
			tag = strings.ToLower(ft.Name)
		)
		switch v := ft.Tag.Get("fig"); v {
		case "":
		case "-":
			continue
		default:
			tag = v
		}
		node, ok := obj.Props[tag]
		if !ok {
			continue
		}
		if err := decode(node, f); err != nil {
			return err
		}
	}
	return nil
}
