package fig

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Setter interface {
	Set(string) error
}

type Updater interface {
	Update() error
}

type FuncMap map[string]interface{}

type Decoder struct {
	read    io.Reader
	fmap    FuncMap
	options *env
	locals  *env
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		read:    r,
		fmap:    make(FuncMap),
		options: emptyEnv(),
		locals:  emptyEnv(),
	}
}

func (d *Decoder) Define(ident string, value interface{}) {
	d.locals.define(ident, value)
}

func (d *Decoder) Funcs(set FuncMap) {
	for k, v := range set {
		d.fmap[k] = v
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
	if isEmpty(value) {
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
		var ok bool
		if ok, err = d.decodeArrayFromInterface(n, value); ok {
			break
		}
		err = d.decodeArray(n, value)
	case *object:
		err = d.decodeObject(n, value)
	case *option:
		err = d.decodeOption(n, value)
	case *literal:
		err = d.decodeLiteral(n, value)
	case *call:
		err = d.decodeCall(n, value)
	case *variable:
		err = d.decodeVariable(n, value)
	default:
		err = fmt.Errorf("value (%s) can not be decoded from %T", value.Kind(), n)
	}
	if err == nil {
		err = d.triggerUpdate(value)
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

func (d *Decoder) decodeSlice(slc *slice, v reflect.Value) error {
	if err := d.decode(slc.Node, v); err != nil {
		return err
	}
	arr := v
	if isEmpty(v) {
		arr = reflect.ValueOf(v.Interface())
	}
	if !isArray(arr) || arr.Len() == 0 {
		return fmt.Errorf("%s can not be sliced", v.Type())
	}
	if slc.IsIndex() {
		from := slc.From()
		if from < 0 {
			from = arr.Len() + from
		}
		if from >= arr.Len() || from < 0 {
			return fmt.Errorf("index out of range (%d >= %d)", slc.from.index, v.Len())
		}
		v.Set(arr.Index(from))
		return nil
	}
	if slc.IsCopy() {
		var (
			s = reflect.SliceOf(arr.Type().Elem())
			f = reflect.MakeSlice(s, arr.Len(), arr.Len())
		)
		reflect.Copy(f, arr)
		v.Set(f)
		return nil
	}

	reindex := func(v int, to bool) (int, error) {
		if to && v == 0 {
			return arr.Len(), nil
		}
		if v < 0 {
			v = arr.Len() + v
		}
		if v < 0 || v >= arr.Len() {
			return v, fmt.Errorf("index out of range")
		}
		return v, nil
	}
	var (
		from, err1 = reindex(slc.From(), false)
		to, err2   = reindex(slc.To(), !slc.to.set)
	)
	if err1 != nil {
		return fmt.Errorf("from: %w", err1)
	}
	if err2 != nil {
		return fmt.Errorf("to: %w", err2)
	}
	if from > to {
		return fmt.Errorf("invalid slice index (%d > %d)", from, to)
	}
	v.Set(arr.Slice(from, to))
	return nil
}

func (d *Decoder) decodeTemplate(tpl *template) (Node, error) {
	var str strings.Builder
	for _, n := range tpl.Nodes {
		switch n.Type() {
		case TypeLiteral:
			s, _ := n.(*literal).GetString()
			str.WriteString(s)
		case TypeVariable:
			val, err := d.resolveVariable(n.(*variable))
			if err != nil {
				return nil, err
			}
			switch v := val.(type) {
			case string:
				str.WriteString(v)
			case int64:
				str.WriteString(strconv.FormatInt(v, 10))
			case float64:
				str.WriteString(strconv.FormatFloat(v, 'f', -1, 64))
			case bool:
				str.WriteString(strconv.FormatBool(v))
			default:
			}
		default:
			return nil, fmt.Errorf("unexpected node type")
		}
	}
	return createLiteralFromString(str.String()), nil
}

func (d *Decoder) resolveVariable(ident *variable) (interface{}, error) {
	var (
		val interface{}
		err error
	)
	if ident.IsLocal() {
		val, err = d.options.resolve(ident.Name())
	} else {
		val, err = d.locals.resolve(ident.Name())
	}
	return val, err
}

func (d *Decoder) decodeVariable(ident *variable, v reflect.Value) error {
	val, err := d.resolveVariable(ident)
	if err != nil {
		return err
	}
	var (
		value = reflect.ValueOf(val)
		typ   = value.Type()
	)
	if typ.AssignableTo(v.Type()) {
		v.Set(value)
	} else if typ.ConvertibleTo(v.Type()) {
		v.Set(value.Convert(v.Type()))
	} else {
		return fmt.Errorf("%s: %s can not be assigned to %s", ident.Name(), typ, v.Type())
	}
	return nil
}

func (d *Decoder) decodeOption(opt *option, v reflect.Value) error {
	if opt.Value == nil {
		return nil
	}
	if ok, err := d.decodeSetter(v, opt); ok {
		return err
	}
	var err error
	switch opt.Value.Type() {
	case TypeLiteral:
		lit, _ := opt.getLiteral()
		if k := v.Kind(); k == reflect.Interface {
			err = d.decodeInterface(lit, v)
			break
		}
		err = d.decodeLiteral(lit, v)
	case TypeTemplate:
		opt.Value, err = d.decodeTemplate(opt.Value.(*template))
		if err != nil {
			break
		}
		return d.decodeOption(opt, v)
	case TypeArray:
		var ok bool
		if ok, err = d.decodeArrayFromInterface(opt.Value, v); ok {
			break
		}
		err = d.decode(opt.Value, v)
	case TypeCall:
		err = d.decodeCall(opt.Value.(*call), v)
	case TypeVariable:
		err = d.decodeVariable(opt.Value.(*variable), v)
	case TypeSlice:
		err = d.decodeSlice(opt.Value.(*slice), v)
	default:
		err = fmt.Errorf("literal/array/slice expected!")
	}
	if err == nil && v.CanInterface() {
		d.define(opt.Ident, v.Interface())
	}
	return err
}

var errtype = reflect.TypeOf((*error)(nil)).Elem()

func (d *Decoder) decodeCall(c *call, v reflect.Value) error {
	call := reflect.ValueOf(d.fmap[c.Ident])
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

func (d *Decoder) decodeArrayFromInterface(n Node, v reflect.Value) (bool, error) {
	if !isEmpty(v) {
		return false, nil
	}
	var (
		s  = reflect.SliceOf(v.Type())
		f  = reflect.MakeSlice(s, 0, 2)
		vf = reflect.New(f.Type()).Elem()
	)
	if err := d.decode(n, vf); err != nil {
		return true, err
	}
	v.Set(vf)
	return true, nil
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
	d.push()
	defer d.pop()

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
		node, ok := obj.take(tag)
		if !ok && tag == ft.Name {
			node, ok = obj.take(strings.ToLower(tag))
			if !ok {
				continue
			}
		}
		if node == nil {
			continue
		}
		if ok, err := d.decodeSpecial(f, node); ok {
			if err != nil {
				return err
			}
			continue
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

	d.push()
	defer d.pop()

	for i, o := range obj.Nodes {
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
			err = fmt.Errorf("%s: can not decode %T", obj.Revex[i], o)
		}
		if err != nil {
			return err
		}
		v.SetMapIndex(reflect.ValueOf(obj.Revex[i]), vf)
	}
	return nil
}

func (d *Decoder) decodeSpecial(v reflect.Value, n Node) (bool, error) {
	var (
		err error
		nok bool
	)
	switch t := v.Type(); {
	case t == timetype:
		err = d.decodeTime(v, n)
	case t == urltype:
		err = d.decodeURL(v, n)
	case t == regextype:
		err = d.decodeRegex(v, n)
	case t == iptype:
		err = d.decodeIP(v, n)
	default:
		nok = true
	}
	return !nok, err
}

var timeformat = []string{
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05Z",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

func (d *Decoder) decodeTime(v reflect.Value, n Node) error {
	opt, ok := n.(*option)
	if !ok {
		return fmt.Errorf("decoding time: option expected")
	}
	var (
		str, err = opt.GetString()
		mmt      time.Time
	)
	if err != nil {
		return err
	}
	for _, f := range timeformat {
		mmt, err = time.Parse(f, str)
		if err == nil {
			v.Set(reflect.ValueOf(mmt))
			return nil
		}
	}
	return err
}

func (d *Decoder) decodeURL(v reflect.Value, n Node) error {
	opt, ok := n.(*option)
	if !ok {
		return fmt.Errorf("decoding url: option expected")
	}
	str, err := opt.GetString()
	if err != nil {
		return err
	}
	u, err := url.Parse(str)
	if err != nil {
		return err
	}
	v.Set(reflect.ValueOf(*u))
	return nil
}

func (d *Decoder) decodeIP(v reflect.Value, n Node) error {
	opt, ok := n.(*option)
	if !ok {
		return fmt.Errorf("decoding IP: option expected")
	}
	str, err := opt.GetString()
	if err != nil {
		return err
	}
	ip := net.ParseIP(str)
	v.Set(reflect.ValueOf(ip))
	return nil
}

func (d *Decoder) decodeRegex(v reflect.Value, n Node) error {
	opt, ok := n.(*option)
	if !ok {
		return fmt.Errorf("decoding regexp: option expected")
	}
	str, err := opt.GetString()
	if err != nil {
		return err
	}
	rxp, err := regexp.Compile(str)
	if err != nil {
		return err
	}
	v.Set(reflect.ValueOf(*rxp))
	return nil
}

var (
	settertype = reflect.TypeOf((*Setter)(nil)).Elem()
	updatetype = reflect.TypeOf((*Updater)(nil)).Elem()
	timetype   = reflect.TypeOf((*time.Time)(nil)).Elem()
	urltype    = reflect.TypeOf((*url.URL)(nil)).Elem()
	regextype  = reflect.TypeOf((*regexp.Regexp)(nil)).Elem()
	iptype     = reflect.TypeOf((*net.IP)(nil)).Elem()
)

func (d *Decoder) triggerUpdate(v reflect.Value) error {
	if v.CanInterface() && v.Type().Implements(updatetype) {
		return v.Interface().(Updater).Update()
	}
	if v.CanAddr() {
		v = v.Addr()
		if v.CanInterface() && v.Type().Implements(updatetype) {
			return v.Interface().(Updater).Update()
		}
	}
	return nil
}

func (d *Decoder) decodeSetter(v reflect.Value, opt *option) (bool, error) {
	decode := func() error {
		str, err := opt.GetString()
		if err != nil {
			return err
		}
		return v.Interface().(Setter).Set(str)
	}
	if v.CanInterface() && v.Type().Implements(settertype) {
		return true, decode()
	}
	if v.CanAddr() {
		v = v.Addr()
		if v.CanInterface() && v.Type().Implements(settertype) {
			return true, decode()
		}
	}
	return false, nil
}

func (d *Decoder) define(ident string, value interface{}) {
	d.options.define(ident, value)
}

func (d *Decoder) push() {
	d.options = enclosedEnv(d.options)
}

func (d *Decoder) pop() {
	d.options = d.options.unwrap()
}

func isEmpty(v reflect.Value) bool {
	return v.Kind() == reflect.Interface && v.NumMethod() == 0
}

func isArray(v reflect.Value) bool {
	return v.Kind() == reflect.Array || v.Kind() == reflect.Slice
}
