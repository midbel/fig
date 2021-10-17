package fig

type node interface{}

type comment struct {
	list  []string
}

type option struct {
	comment
	name   string
	values node
}

type object struct {
	props map[string]node
	comment node
}

type array struct {
	nodes []node
	comment node
}

type literal struct {
	token Token
	comment node
}

type macro struct {
	name  string
	args  []node
	named map[string]node
	comment node
}
