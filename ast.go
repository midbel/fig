package fig

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	errObject   = errors.New("not an object")
	errRegister = errors.New("node can not be registered")
)

type NodeType int8

const (
	TypeLiteral NodeType = iota
	TypeVariable
	TypeSlice
	TypeOption
	TypeArray
	TypeObject
	TypeCall
)

type Node interface {
	fmt.Stringer
	Type() NodeType
	clone() Node
}

type note struct {
	Token Token
}

func (n note) String() string {
	return fmt.Sprintf("comment(%s)", n.Token.Literal)
}

type option struct {
	Ident string
	Value Node
}

func createOption(ident string, value Node) *option {
	return &option{
		Ident: ident,
		Value: value,
	}
}

func (o *option) String() string {
	if o.Value == nil {
		return fmt.Sprintf("option(%s)", o.Ident)
	}
	return fmt.Sprintf("option(%s, %s)", o.Ident, o.Value.String())
}

func (o *option) clone() Node {
	return createOption(o.Ident, o.Value.clone())
}

func (o *option) GetString() (string, error) {
	i, err := o.getLiteral()
	if err != nil {
		return "", err
	}
	return i.GetString()
}

func (o *option) GetBool() (bool, error) {
	i, err := o.getLiteral()
	if err != nil {
		return false, err
	}
	return i.GetBool()
}

func (o *option) GetInt() (int64, error) {
	i, err := o.getLiteral()
	if err != nil {
		return 0, err
	}
	return i.GetInt()
}

func (o *option) GetUint() (uint64, error) {
	i, err := o.getLiteral()
	if err != nil {
		return 0, err
	}
	return i.GetUint()
}

func (o *option) GetFloat() (float64, error) {
	i, err := o.getLiteral()
	if err != nil {
		return 0, err
	}
	return i.GetFloat()
}

func (o *option) getLiteral() (*literal, error) {
	i, ok := o.Value.(*literal)
	if !ok {
		return nil, fmt.Errorf("%s is not a literal", o.Ident)
	}
	return i, nil
}

func (o *option) getArray() (*array, error) {
	a, ok := o.Value.(*array)
	if !ok {
		return nil, fmt.Errorf("%s is not an array", o.Ident)
	}
	return a, nil
}

func (_ *option) Type() NodeType {
	return TypeOption
}

type object struct {
	parent *object

	Name     string
	Partials map[string]Node
	Comment  Node

	Index map[string]int
	Revex map[int]string
	Nodes []Node
}

func createObject(ident string) *object {
	return enclosedObject(ident, nil)
}

func enclosedObject(ident string, parent *object) *object {
	return &object{
		parent:   parent,
		Name:     ident,
		Partials: make(map[string]Node),
		Index:    make(map[string]int),
		Revex:    make(map[int]string),
	}
}

func (o *object) String() string {
	return fmt.Sprintf("object(%s)", o.Name)
}

func (_ *object) Type() NodeType {
	return TypeObject
}

func (o *object) clone() Node {
	// INFO: we don't include parent in clone object because clone is only used
	// when "plucking" a defined object to be inserted somewhere else in the tree
	// then, the plucked object will have it's parent field set properly
	obj := createObject(o.Name)
	for k, i := range o.Index {
		obj.put(k, o.at(i).clone())
	}
	return obj
}

func (o *object) repeat(count int64, name string, nest Node) error {
	if count <= 1 {
		return fmt.Errorf("repeat can not be less or equal to 1! got %d", count)
	}
	obj, ok := nest.(*object)
	if !ok {
		return notAnObject("node")
	}
	obj.Name = name
	arr := createArray()
	for i := int64(0); i < count; i++ {
		arr.Append(obj.clone())
	}
	return o.set(arr)
}

func (o *object) extend(name, as string, n Node) error {
	ori, ok := o.Partials[name]
	if !ok {
		return fmt.Errorf("%s: undefined node", name)
	}
	obj, ok := ori.(*object)
	if !ok {
		return notAnObject(name)
	}
	nest, ok := n.(*object)
	if !ok {
		return notAnObject("node")
	}
	if as != "" {
		obj = obj.clone().(*object)
	}
	if err := obj.merge(nest); err != nil {
		return err
	}
	if as == "" {
		o.Partials[name] = obj
	} else {
		_, ok := o.Partials[as]
		if ok {
			return fmt.Errorf("%s can not be replaced by extended object", as)
		}
		obj.Name = as
		o.Partials[as] = obj
	}
	return nil
}

func (o *object) define(ident string, n Node) error {
	obj, ok := n.(*object)
	if !ok {
		return notAnObject(ident)
	}
	obj.Name = ident
	obj.parent = nil
	o.Partials[ident] = obj
	return nil
}

func (o *object) apply(ident string, keys []string, depth int64) (Node, error) {
	n, ok := o.Partials[ident]
	if ok {
		obj, ok := n.(*object)
		if ok {
			return o.pluck(obj, keys, depth)
		}
	}
	if o.parent != nil {
		return o.parent.apply(ident, keys, depth)
	}
	return nil, fmt.Errorf("%s: undefined node", ident)
}

func (o *object) pluck(ori *object, keys []string, depth int64) (Node, error) {
	obj := createObject(ori.Name)
	if len(keys) == 0 {
		for k := range ori.Index {
			keys = append(keys, k)
		}
	}
	for _, k := range keys {
		v, ok := ori.take(k)
		if !ok {
			return nil, fmt.Errorf("%s: undefined property in %s", k, ori.Name)
		}
		// TODO: if v is an object, we have to check the depth value to see how depth
		// we have to go when we clone it
		obj.put(k, v.clone())
	}
	return obj, nil
}

func (o *object) getObject(ident string, last bool) (*object, error) {
	nest, ok := o.take(ident)
	if !ok {
		// nest := createObject(ident)
		nest := enclosedObject(ident, o)
		o.put(ident, nest)
		return nest, nil
	}
	if last {
		return o.getLastObject(ident, nest)
	}
	switch nest.Type() {
	case TypeObject:
		return nest.(*object), nil
	case TypeArray:
		var (
			arr = nest.(*array)
			obj = arr.Nodes[len(arr.Nodes)-1]
		)
		if obj.Type() != TypeObject {
			return nil, notAnObject(ident)
		}
		return obj.(*object), nil
	default:
		return nil, notAnObject(ident)
	}
}

func (o *object) getLastObject(ident string, parent Node) (*object, error) {
	nest := enclosedObject(ident, o)
	switch curr := parent.(type) {
	case *object:
		arr := createArray()
		arr.Append(curr)
		arr.Append(nest)
		o.put(ident, arr)
	case *array:
		curr.Append(nest)
		o.put(ident, curr)
	default:
		return nil, notAnObject("node")
	}
	return nest, nil
}

func (o *object) set(n Node) error {
	// TODO: make sure that when an object is set, its parent field is set properly
	var err error
	switch n := n.(type) {
	case *option:
		err = o.registerOption(n)
	case *object:
		err = o.registerObject(n)
	case *array:
		for _, n := range n.Nodes {
			if err := o.set(n); err != nil {
				return err
			}
		}
	default:
		return errRegister
	}
	return err
}

func (o *object) registerObject(obj *object) error {
	curr, ok := o.take(obj.Name)
	if !ok {
		o.put(obj.Name, obj)
		return nil
	}
	switch prev := curr.(type) {
	case *object:
		arr := createArray()
		arr.Append(prev)
		arr.Append(obj)
		curr = arr
	case *array:
		if err := prev.Append(obj); err != nil {
			return err
		}
		curr = prev
	default:
		return fmt.Errorf("%s: %w", obj.Name, errRegister)
	}
	o.put(obj.Name, curr)
	return nil
}

func (o *object) registerOption(opt *option) error {
	curr, ok := o.take(opt.Ident)
	if !ok {
		o.put(opt.Ident, opt)
		return nil
	}
	c, ok := curr.(*option)
	if !ok {
		return fmt.Errorf("%s: %w", opt.Ident, errRegister)
	}
	switch val := c.Value.(type) {
	case *literal:
		arr := createArray()
		arr.Append(val)
		arr.Append(opt.Value)
		c.Value = arr
	case *array:
		return val.Append(opt.Value)
	default:
		return fmt.Errorf("%s: %w", opt.Ident, errRegister)
	}
	return nil
}

func (o *object) merge(node Node) error {
	if node.Type() != TypeObject {
		return notAnObject("node")
	}
	obj := node.(*object)
	for k := range obj.Index {
		n, ok := obj.take(k)
		if !ok {
			return fmt.Errorf("%s: property not found", k)
		}
		if err := o.set(n); err != nil {
			return err
		}
	}
	return nil
}

func (o *object) replace(node Node) error {
	if node.Type() != TypeObject {
		return notAnObject("node")
	}
	obj := node.(*object)
	o.put(obj.Name, obj)
	return nil
}

func (o *object) insert(node Node) error {
	if node.Type() != TypeObject {
		return notAnObject("node")
	}
	var (
		obj      = node.(*object)
		curr, ok = o.take(obj.Name)
	)
	if !ok {
		o.put(obj.Name, node)
		return nil
	}
	var err error
	switch curr.Type() {
	case TypeObject:
		arr := createArray()
		arr.Append(curr)
		arr.Append(node)
		o.put(obj.Name, arr)
	case TypeArray:
		arr := curr.(*array)
		err = arr.Append(node)
	default:
		return notAnObject(obj.Name)
	}
	return err
}

func (o *object) at(i int) Node {
	return o.Nodes[i]
}

func (o *object) take(ident string) (Node, bool) {
	i, ok := o.Index[ident]
	if !ok {
		return nil, ok
	}
	return o.at(i), ok
}

func (o *object) put(ident string, node Node) {
	i, ok := o.Index[ident]
	if !ok {
		z := len(o.Nodes)
		o.Index[ident] = z
		o.Revex[z] = ident
		o.Nodes = append(o.Nodes, node)
		return
	}
	o.Nodes[i] = node
}

type array struct {
	Nodes   []Node
	Comment Node
}

func createArray() *array {
	var arr array
	return &arr
}

func (a *array) Append(n Node) error {
	a.Nodes = append(a.Nodes, n)
	return nil
}

func (a *array) String() string {
	var str strings.Builder
	str.WriteString("array(")
	for i := range a.Nodes {
		if i > 0 {
			str.WriteString(", ")
		}
		str.WriteString(a.Nodes[i].String())
	}
	str.WriteString(")")
	return str.String()
}

func (_ *array) Type() NodeType {
	return TypeArray
}

func (a *array) clone() Node {
	arr := createArray()
	for i := range a.Nodes {
		arr.Nodes = append(arr.Nodes, a.Nodes[i].clone())
	}
	return arr
}

type slice struct {
	Node
	from struct {
		index int64
		set   bool
	}
	to struct {
		index int64
		set   bool
	}
}

func createSlice(node Node) *slice {
	return &slice{
		Node: node,
	}
}

func (s *slice) From() int {
	return int(s.from.index)
}

func (s *slice) To() int {
	return int(s.to.index)
}

func (s *slice) IsCopy() bool {
	return !s.from.set && !s.to.set
}

func (s *slice) IsIndex() bool {
	return s.from == s.to
}

func (s *slice) String() string {
	return fmt.Sprintf("slice(%s, from: %d, to: %d)", s.Node, s.from.index, s.to.index)
}

func (s *slice) Type() NodeType {
	return TypeSlice
}

func (s *slice) clone() Node {
	return s
}

type Argument interface {
	GetString() (string, error)
	GetFloat() (float64, error)
	GetInt() (int64, error)
	GetUint() (uint64, error)
	GetBool() (bool, error)
}

type variable struct {
	Ident Token
}

func createVariable(tok Token) *variable {
	return &variable{
		Ident: tok,
	}
}

func (v *variable) IsLocal() bool {
	return v.Ident.Type == LocalVar
}

func (v *variable) Name() string {
	return v.Ident.Literal
}

func (_ *variable) Type() NodeType {
	return TypeVariable
}

func (v *variable) String() string {
	return fmt.Sprintf("variable(%s)", v.Ident.Literal)
}

func (v *variable) clone() Node {
	return createVariable(v.Ident)
}

const (
	si   = 1000
	iec  = 1024
	min  = 60
	hour = min * min
	day  = hour * 24
	week = day * 7
	year = day * 365
)

var multipliers = map[string]func(float64) float64{
	"K":  multiplyFloat(si),
	"M":  multiplyFloat(si * si),
	"G":  multiplyFloat(si * si * si),
	"T":  multiplyFloat(si * si * si * si),
	"Kb": multiplyFloat(iec),
	"Mb": multiplyFloat(iec * iec),
	"Gb": multiplyFloat(iec * iec * iec),
	"Tb": multiplyFloat(iec * iec * iec * iec),
	"s":  multiplyFloat(1),
	"m":  multiplyFloat(min),
	"h":  multiplyFloat(hour),
	"d":  multiplyFloat(day),
	"w":  multiplyFloat(week),
	"y":  multiplyFloat(year),
	"ms": multiplyFloat(1 / si),
	"":   multiplyFloat(1),
}

func multiplyFloat(mul float64) func(float64) float64 {
	return func(v float64) float64 {
		return v * mul
	}
}

func convertFloat(str, mul string) (float64, error) {
	v, err := strconv.ParseFloat(str, 64)
	if err != nil {
		i, err := strconv.ParseInt(str, 0, 64)
		return float64(i), err
	}
	fn, ok := multipliers[mul]
	if !ok {
		return v, fmt.Errorf("%s: unsupported/unknown multipliers", mul)
	}
	return fn(v), nil
}

type literal struct {
	Token   Token
	Mul     Token
	Comment Node
}

func createLiteralFromString(str string) *literal {
	tok := Token{
		Literal: str,
		Type:    String,
	}
	return createLiteral(tok)
}

func createLiteral(tok Token) *literal {
	return &literal{
		Token: tok,
	}
}

func (i *literal) String() string {
	if i.Mul.isZero() {
		return fmt.Sprintf("literal(%s)", i.Token.Literal)
	}
	return fmt.Sprintf("literal(%s, mul: %s)", i.Token.Literal, i.Mul.Literal)
}

func (_ *literal) Type() NodeType {
	return TypeLiteral
}

func (i *literal) GetString() (string, error) {
	return i.Token.Literal, nil
}

func (i *literal) GetBool() (bool, error) {
	return strconv.ParseBool(i.Token.Literal)
}

func (i *literal) GetInt() (int64, error) {
	v, err := convertFloat(i.Token.Literal, i.Mul.Literal)
	return int64(v), err
}

func (i *literal) GetUint() (uint64, error) {
	v, err := convertFloat(i.Token.Literal, i.Mul.Literal)
	return uint64(v), err
}

func (i *literal) GetFloat() (float64, error) {
	return convertFloat(i.Token.Literal, i.Mul.Literal)
}

func (i *literal) Get() (interface{}, error) {
	switch i.Token.Type {
	case Boolean:
		return i.GetBool()
	case String, Heredoc:
		return i.GetString()
	case Integer:
		return i.GetInt()
	case Float:
		return i.GetFloat()
	default:
		return nil, fmt.Errorf("unknown literal type")
	}
}

func (i *literal) clone() Node {
	n := createLiteral(i.Token)
	n.Mul = i.Mul
	return n
}

type call struct {
	Ident  string
	Args   []Node
	Kwargs map[string]Node
}

func createCall(ident string) *call {
	return &call{
		Ident:  ident,
		Kwargs: make(map[string]Node),
	}
}

func (_ *call) Type() NodeType {
	return TypeCall
}

func (c *call) String() string {
	return fmt.Sprintf("call(%s)", c.Ident)
}

func (c *call) clone() Node {
	a := createCall(c.Ident)
	for i := range c.Args {
		a.Args = append(a.Args, c.Args[i].clone())
	}
	for k, v := range c.Kwargs {
		a.Kwargs[k] = v.clone()
	}
	return nil
}

func notAnObject(what string) error {
	return fmt.Errorf("%s is %w", what, errObject)
}
