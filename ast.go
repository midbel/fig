package fig

import (
	"fmt"
	"strings"
)

type node interface {
	fmt.Stringer
}

type note struct {
	Token Token
}

func (n note) String() string {
	return fmt.Sprintf("comment(%s)", n.Token.Literal)
}

type option struct {
	Name  string
	Value node
}

func (o option) String() string {
	return fmt.Sprintf("option(%s, %s)", o.Name, o.Value.String())
}

type object struct {
	Props   map[string]node
	Comment node
}

func (o object) String() string {
	return "object()"
}

type array struct {
	Nodes   []node
	Comment node
}

func (a array) String() string {
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

type literal struct {
	Token   Token
	Comment node
}

func createLiteral(tok Token) literal {
	return literal{
		Token: tok,
	}
}

func (i literal) String() string {
	return fmt.Sprintf("literal(%s)", i.Token.Literal)
}

type macro struct {
	Name    string
	Args    []node
	Named   map[string]node
	Comment node
}
