package fig

import (
	"fmt"
	"strconv"
	"strings"
)

type NodeType int8

const (
	TypeLiteral NodeType = iota
	TypeVariable
	TypeOption
	TypeArray
	TypeObject
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
	Props    map[string]Node
	Partials map[string]Node
	Comment  Node
}

func createObject(ident string) *object {
	return enclosedObject(ident, nil)
}

func enclosedObject(ident string, parent *object) *object {
	return &object{
		parent:   parent,
		Name:     ident,
		Props:    make(map[string]Node),
		Partials: make(map[string]Node),
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
	for k, v := range o.Props {
		obj.Props[k] = v.clone()
	}
	return obj
}

func (o *object) define(ident string, n Node) error {
	obj, ok := n.(*object)
	if !ok {
		return fmt.Errorf("%s is not an object", ident)
	}
	obj.Name = ident
	obj.parent = nil
	o.Partials[ident] = obj
	return nil
}

func (o *object) get(ident string, keys []string, depth int64) (Node, error) {
	n, ok := o.Partials[ident]
	if ok {
		obj, ok := n.(*object)
		if ok {
			return o.pluck(obj, keys, depth)
		}
	}
	if o.parent != nil {
		return o.parent.get(ident, keys, depth)
	}
	return nil, fmt.Errorf("%s: undefined node", ident)
}

func (o *object) pluck(ori *object, keys []string, depth int64) (Node, error) {
	obj := createObject(ori.Name)
	for _, k := range keys {
		v, ok := ori.Props[k]
		if !ok {
			return nil, fmt.Errorf("%s: undefined property in %s", k, ori.Name)
		}
		// TODO: if v is an object, we have to check the depth value to see how depth
		// we have to go when we clone it
		obj.Props[k] = v.clone()
	}
	return obj, nil
}

func (o *object) resolve(ident string) (*option, error) {
	n, ok := o.Props[ident]
	if ok {
		opt, ok := n.(*option)
		if ok {
			return opt, nil
		}
	}
	if o.parent != nil {
		return o.parent.resolve(ident)
	}
	return nil, fmt.Errorf("%s: undefined option", ident)
}

func (o *object) getObject(ident string, last bool) (*object, error) {
	nest, ok := o.Props[ident]
	if !ok {
		// nest := createObject(ident)
		nest := enclosedObject(ident, o)
		o.Props[ident] = nest
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
			return nil, fmt.Errorf("%s is not an object", ident)
		}
		return obj.(*object), nil
	default:
		return nil, fmt.Errorf("%s is not an object", ident)
	}
}

func (o *object) getLastObject(ident string, parent Node) (*object, error) {
	nest := enclosedObject(ident, o)
	// nest := createObject(ident)
	switch curr := parent.(type) {
	case *object:
		arr := createArray()
		arr.Append(curr)
		arr.Append(nest)
		o.Props[ident] = arr
	case *array:
		curr.Append(nest)
		o.Props[ident] = curr
	default:
		return nil, fmt.Errorf("%s is not an object", parent)
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
		return fmt.Errorf("node can not be registered")
	}
	return err
}

func (o *object) registerObject(obj *object) error {
	curr, ok := o.Props[obj.Name]
	if !ok {
		o.Props[obj.Name] = obj
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
		return fmt.Errorf("%s: object can not be registered", obj.Name)
	}
	o.Props[obj.Name] = curr
	return nil
}

func (o *object) registerOption(opt *option) error {
	switch v := opt.Value.(type) {
	case *variable:
		res, err := o.resolve(v.Ident.Literal)
		if err != nil {
			return err
		}
		opt.Value = res.Value.clone()
	case *array:
		for i := range v.Nodes {
			if vr, ok := v.Nodes[i].(*variable); ok {
				res, err := o.resolve(vr.Ident.Literal)
				if err != nil {
					return err
				}
				v.Nodes[i] = res.clone()
			}
		}
	default:
	}
	curr, ok := o.Props[opt.Ident]
	if !ok {
		o.Props[opt.Ident] = opt
		return nil
	}

	c, ok := curr.(*option)
	if !ok {
		return fmt.Errorf("%s: option can not be registered", opt.Ident)
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
		return fmt.Errorf("%s: option can not be registered", opt.Ident)
	}
	return nil
}

func (o *object) merge(node Node) error {
	if node.Type() != TypeObject {
		return fmt.Errorf("node is not an object")
	}
	obj := node.(*object)
	for k := range obj.Props {
		if err := o.set(obj.Props[k]); err != nil {
			return err
		}
	}
	return nil
}

func (o *object) replace(node Node) error {
	if node.Type() != TypeObject {
		return fmt.Errorf("node is not an object")
	}
	obj := node.(*object)
	o.Props[obj.Name] = obj
	return nil
}

func (o *object) insert(node Node) error {
	if node.Type() != TypeObject {
		return fmt.Errorf("node is not an object")
	}
	var (
		obj      = node.(*object)
		curr, ok = o.Props[obj.Name]
	)
	if !ok {
		o.Props[obj.Name] = node
		return nil
	}
	var err error
	switch curr.Type() {
	case TypeObject:
		arr := createArray()
		arr.Append(curr)
		arr.Append(node)
		o.Props[obj.Name] = arr
	case TypeArray:
		arr := curr.(*array)
		err = arr.Append(node)
	default:
		err = fmt.Errorf("%s is not an object", obj.Name)
	}
	return err
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
	// if len(a.Nodes) > 0 && n.Type() != a.Nodes[0].Type() {
	// 	return fmt.Errorf("node can not be appended")
	// }
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
	for i := range arr.Nodes {
		arr.Nodes = append(arr.Nodes, arr.Nodes[i].clone())
	}
	return arr
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

func (_ *variable) Type() NodeType {
	return TypeVariable
}

func (v *variable) String() string {
	return fmt.Sprintf("variable(%s)", v.Ident.Literal)
}

func (v *variable) clone() Node {
	return createVariable(v.Ident)
}

type literal struct {
	Token   Token
	Mul     Token
	Comment Node
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

func (i *literal) GetInt() (int64, error) {
	return strconv.ParseInt(i.Token.Literal, 0, 64)
}

func (i *literal) GetUint() (uint64, error) {
	return strconv.ParseUint(i.Token.Literal, 0, 64)
}

func (i *literal) GetBool() (bool, error) {
	return strconv.ParseBool(i.Token.Literal)
}

func (i *literal) GetFloat() (float64, error) {
	return strconv.ParseFloat(i.Token.Literal, 64)
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

type macro struct {
	Name    string
	Args    []Node
	Named   map[string]Node
	Comment Node
}
