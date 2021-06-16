package fig

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

func getTry(args map[string]Expr, env Environment) (bool, error) {
	var (
		val Value
		err error
	)
	if tmp, ok := args["try"]; ok {
		if val, err = tmp.Eval(env); err != nil {
			return false, err
		}
	}
	return val != nil && val.isTrue(), nil
}

func getKey(args map[string]Expr, env Environment) (string, error) {
	var (
		val Value
		err error
	)
	if tmp, ok := args["key"]; ok {
		if val, err = tmp.Eval(env); err != nil {
			return "", err
		}
	} else {
		return "", nil
	}
	return toText(val)
}

func getInsert(args map[string]Expr, env Environment) (string, error) {
	var (
		val Value
		err error
	)
	if tmp, ok := args["insert"]; ok {
		if val, err = tmp.Eval(env); err != nil {
			return "", err
		}
	} else {
		return "", nil
	}
	return toText(val)
}

func getFile(args map[string]Expr, try bool, env Environment) (*Object, error) {
	var (
		val Value
		err error
	)
	tmp, ok := args["file"]
	if !ok {
		return nil, missingArgument("include", "file")
	}
	if val, err = tmp.Eval(env); err != nil {
		return nil, err
	}
	file, err := toText(val)
	if err != nil {
		return nil, err
	}
	rc, err := includeFile(file)
	if err != nil {
		if try {
			err = nil
		}
		return nil, err
	}
	defer rc.Close()

	return Parse(rc)
}

func include(args map[string]Expr) (Node, error) {
	if len(args) == 0 {
		return nil, invalidArgument("include")
	}
	var (
		err error
		env = EmptyEnv()
	)

	try, err := getTry(args, env)
	if err != nil {
		return nil, err
	}

	key, err := getKey(args, env)
	if err != nil {
		return nil, err
	}

	node, err := getFile(args, try, env)
	if err != nil || node == nil {
		return nil, err
	}
	if key != "" {
		node.name = makeToken(key, Ident)
		node.insmode, err = getInsert(args, env)
		if err != nil {
			return nil, err
		}
	}
	return node, nil
}

func includeFile(file string) (io.ReadCloser, error) {
	u, err := url.Parse(file)
	if err != nil {
		return nil, err
	}
	var rc io.ReadCloser
	switch u.Scheme {
	case "", "file":
		r, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		rc = r
	case "http", "https":
		resp, err := http.Get(file)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code")
		}
		rc = resp.Body
	default:
		return nil, fmt.Errorf("%s: unsupported protocol", u.Scheme)
	}
	return rc, nil
}
