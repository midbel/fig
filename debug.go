package fig

import (
	"fmt"
	"io"
	"strings"
)

func Debug(w io.Writer, r io.Reader) error {
	obj, err := Parse(r)
	if err != nil {
		return err
	}
	return debugObject(w, obj, -2, true)
}

func debugObject(w io.Writer, obj *Object, level int, label bool) error {
	name := obj.name.Input
	if name == "" {
		name = "document"
	}
	level += 2
	fmt.Print(strings.Repeat(" ", level))
	if label {
		fmt.Fprintf(w, "object(%s) {", name)
	} else {
		fmt.Fprint(w, "{")
	}
	fmt.Fprintln(w)
	for _, n := range obj.nodes {
		switch n := n.(type) {
		case Option:
			debugOption(w, n, level)
		case List:
			debugList(w, n, level)
		case Func:
			debugFunc(w, n, level)
		case *Object:
			debugObject(w, n, level, true)
		default:
			return fmt.Errorf("unexpected node type %T", n)
		}
	}
	fmt.Fprint(w, strings.Repeat(" ", level))
	fmt.Fprintln(w, "}")
	return nil
}

func debugList(w io.Writer, i List, level int) error {
	level += 2
	fmt.Fprint(w, strings.Repeat(" ", level))
	fmt.Fprintf(w, "list(%s) [", i.name.Input)
	fmt.Fprintln(w)
	for _, n := range i.nodes {
		switch n := n.(type) {
		case Option:
			debugOption(w, n, level)
		case *Object:
			debugObject(w, n, level, false)
		default:
			return fmt.Errorf("unexpected node type %T", n)
		}
	}
	fmt.Fprint(w, strings.Repeat(" ", level))
	fmt.Fprintln(w, "]")
	return nil
}

func debugFunc(w io.Writer, fn Func, level int) {
	level += 2
	fmt.Fprint(w, strings.Repeat(" ", level))
	fmt.Fprint(w, fn)
	fmt.Fprintln(w)
}

func debugOption(w io.Writer, opt Option, level int) {
	level += 2
	fmt.Fprint(w, strings.Repeat(" ", level))
	fmt.Fprintf(w, "%s: ", opt.name.Input)
	debugExpr(w, opt.expr)
	fmt.Fprintln(w)
}

func debugExpr(w io.Writer, e Expr) {
	switch e := e.(type) {
	case Literal:
		fmt.Fprintf(w, "literal(%s)", e.tok.Input)
	case Variable:
		fmt.Fprintf(w, "variable(%s)", e.tok.Input)
	case Identifier:
		fmt.Fprintf(w, "identifier(%s)", e.tok.Input)
	case Template:
		fmt.Fprint(w, "template(")
		for i, e := range e.expr {
			if i > 0 {
				fmt.Fprint(w, ", ")
			}
			debugExpr(w, e)
		}
		fmt.Fprint(w, ")")
	case Unary:
		fmt.Fprint(w, "unary(right: ")
		debugExpr(w, e.right)
		fmt.Fprintf(w, ", op: %s)", types[e.op])
	case Binary:
		fmt.Fprint(w, "binary(left: ")
		debugExpr(w, e.left)
		fmt.Fprint(w, ", right: ")
		debugExpr(w, e.right)
		fmt.Fprintf(w, ", op: %s)", types[e.op])
	case Array:
		fmt.Fprint(w, "array(")
		for i, e := range e.expr {
			if i > 0 {
				fmt.Fprint(w, ", ")
			}
			debugExpr(w, e)
		}
		fmt.Fprint(w, ")")
	default:
	}
}
