package fig

import (
	"fmt"
	"strconv"
	"strings"
)

type NodeType int8

const (
	TypeLiteral NodeType = iota
	TypeOption
	TypeArray
	TypeObject
)

type Node interface {
	fmt.Stringer
	Type() NodeType
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
	return fmt.Sprintf("option(%s, %s)", o.Ident, o.Value.String())
}

func (_ *option) Type() NodeType {
	return TypeOption
}

type object struct {
	Name    string
	Props   map[string]Node
	Comment Node
}

func createObject(ident string) *object {
	return &object{
		Name:  ident,
		Props: make(map[string]Node),
	}
}

func (o *object) String() string {
	return fmt.Sprintf("object(%s)", o.Name)
}

func (_ *object) Type() NodeType {
	return TypeObject
}

func (o *object) getObject(ident string, last bool) (*object, error) {
	nest, ok := o.Props[ident]
	if !ok {
		nest := createObject(ident)
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
	nest := createObject(ident)
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
	var err error
	switch n := n.(type) {
	case *option:
		err = o.registerOption(n)
	case *object:
		err = o.registerObject(n)
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
	curr, ok := o.Props[opt.Ident]
	if !ok {
		o.Props[opt.Ident] = opt
		return nil
	}
	switch prev := curr.(type) {
	case *option:
		arr := createArray()
		arr.Append(prev)
		arr.Append(opt)
		curr = arr
	case *array:
		if err := prev.Append(opt); err != nil {
			return err
		}
		curr = prev
	default:
		return fmt.Errorf("%s: option can not be registered", opt.Ident)
	}
	o.Props[opt.Ident] = curr
	return nil
}

func (o *object) merge(node Node) error {
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
	if curr.Type() != TypeObject {
		return fmt.Errorf("%s is not an object", obj.Name)
	}
	other := curr.(*object)
	for k := range obj.Props {
		if err := other.set(obj.Props[k]); err != nil {
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
	if len(a.Nodes) > 0 && n.Type() != a.Nodes[0].Type() {
		return fmt.Errorf("node can not be appended")
	}
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

func (i *literal) GetBool() (bool, error) {
	return strconv.ParseBool(i.Token.Literal)
}

func (i *literal) GetFloat() (float64, error) {
	return strconv.ParseFloat(i.Token.Literal, 64)
}

type macro struct {
	Name    string
	Args    []Node
	Named   map[string]Node
	Comment Node
}
