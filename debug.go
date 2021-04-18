package fig

import (
	"fmt"
	"io"
	"strings"
)

func Debug(r io.Reader) error {
	obj, err := Parse(r)
	if err != nil {
		return err
	}
	return debugObject(obj, -2, true)
}

func debugObject(obj *Object, level int, label bool) error {
	name := obj.name.Input
	if name == "" {
		name = "document"
	}
	level += 2
	fmt.Print(strings.Repeat(" ", level))
	if label {
		fmt.Printf("object(%s) {", name)
	} else {
		fmt.Print("{")
	}
	fmt.Println()
	for _, n := range obj.nodes {
		switch n := n.(type) {
		case Option:
			debugOption(n, level)
		case List:
			debugList(n, level)
		case *Object:
			debugObject(n, level, true)
		default:
			return fmt.Errorf("unexpected node type %T", n)
		}
	}
	fmt.Print(strings.Repeat(" ", level))
	fmt.Println("}")
	return nil
}

func debugList(i List, level int) error {
	level += 2
	fmt.Print(strings.Repeat(" ", level))
	fmt.Printf("list(%s) [", i.name.Input)
	fmt.Println()
	for _, n := range i.nodes {
		switch n := n.(type) {
		case Option:
			debugOption(n, level)
		case *Object:
			debugObject(n, level, false)
		default:
			return fmt.Errorf("unexpected node type %T", n)
		}
	}
	fmt.Print(strings.Repeat(" ", level))
	fmt.Println("]")
	return nil
}

func debugOption(opt Option, level int) {
	level += 2
	fmt.Print(strings.Repeat(" ", level))
	fmt.Printf("%s: %s", opt.name.Input, opt.expr)
	fmt.Println()
}
