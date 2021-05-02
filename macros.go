package fig

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

func include(args map[string]Expr) (Node, error) {
	if len(args) == 0 {
		return nil, invalidArgument("include")
	}
	var (
		val Value
		err error
		env = EmptyEnv()
	)
	if tmp, ok := args["try"]; ok {
		if val, err = tmp.Eval(env); err != nil {
			return nil, err
		}
	}
	try := val != nil && val.isTrue()

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

	node, err := Parse(rc)
	if err != nil && !try {
		return nil, err
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
