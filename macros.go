package fig

import (
	"os"
)

// type MacroFunc func(map[string]Expr) (Node, error)

func include(args []Expr) (Node, error) {
	if len(args) == 0 {
		return nil, invalidArgument("include")
	}
	var (
		env    = EmptyEnv()
		values = make([]Value, len(args))
	)
	for i := range args {
		a, err := args[i].Eval(env)
		if err != nil {
			return nil, err
		}
		values[i] = a
	}
	file, err := toText(values[0])
	if err != nil {
		return nil, err
	}
	r, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return Parse(r)
}
