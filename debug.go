package fig

import (
	"fmt"
	"io"
	"strings"
)

func Debug(r io.Reader, w io.Writer) error {
	n, err := Parse(r)
	if err != nil {
		return err
	}
	debugNode(w, n, 0)
	return nil
}

func debugNode(w io.Writer, n Node, level int) {
	prefix := strings.Repeat(" ", level)
	switch n := n.(type) {
	case *object:
		fmt.Fprint(w, prefix)
		fmt.Fprint(w, n)
		fmt.Fprintln(w, "[")
		for _, n := range n.Nodes {
			debugNode(w, n, level+2)
		}
		fmt.Fprint(w, prefix)
		fmt.Fprintln(w, "]")
	case *array:
		fmt.Fprint(w, prefix)
		fmt.Fprintln(w, "array[")
		for _, n := range n.Nodes {
			debugNode(w, n, level+2)
		}
		fmt.Fprint(w, prefix)
		fmt.Fprintln(w, "]")
	default:
		fmt.Fprint(w, prefix)
		fmt.Fprintln(w, n)
	}
}
