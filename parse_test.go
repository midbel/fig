package fig

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestParseCall(t *testing.T) {
	data := []struct {
		Input string
		Args  []Argument
		Err   error
	}{
		{
			Input: "call = func()",
		},
		{
			Input: "call = func(0)",
			Args:  []Argument{},
		},
		{
			Input: "call = func(0, 'str')",
			Args:  []Argument{},
		},
		{
			Input: "call = func(0, arg='str')",
			Args:  []Argument{},
		},
	}
	for _, d := range data {
		p, err := NewParser(strings.NewReader(d.Input))
		if err != nil {
			t.Errorf("fail to init parser")
			continue
		}
		obj := createObject()
		err = p.parse(obj)
		if d.Err != nil {
			if err == nil {
				t.Errorf("expected error but parse succeed!")
			} else if !errors.Is(err, d.Err) {
				t.Errorf("errors mismatched! want %s, got %s", d.Err, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("fail to parse %s: %s", d.Input, err)
			continue
		}
		opt, ok := obj.nodes["call"].(Option)
		if !ok {
			t.Errorf("call option not found")
			continue
		}
		fn, ok := opt.expr.(Call)
		if !ok {
			t.Errorf("call type mismatched! expected %T, got %T", fn, opt.expr)
			continue
		}
		if !testArguments(t, fn.args, d.Args) {
			continue
		}
	}
}

func TestParseFunc(t *testing.T) {
	data := []struct {
		Input string
		Args  []Argument
		Body  Expr
		Err   error
	}{
		{
			Input: "func() {}",
		},
		{
			Input: "func(arg) {}",
			Args: []Argument{
				{
					name: makeToken("arg", Ident),
					pos:  0,
				},
			},
		},
		{
			Input: "func(pos,\n kw=0\n) {}",
			Args: []Argument{
				{
					name: makeToken("pos", Ident),
					pos:  0,
				},
				{
					name: makeToken("kw", Ident),
					pos:  1,
					expr: makeLiteral(makeToken("0", Integer)),
				},
			},
		},
		{
			Input: "func(arg=0, pos) {}",
			Err:   ErrSyntax,
		},
		{
			Input: "func(\"str\", 125) {}",
			Err:   ErrUnexpected,
		},
		{
			Input: "func(args,) {}",
			Err:   ErrSyntax,
		},
		{
			Input: "func(args\n\targs) {}",
			Err:   ErrSyntax,
		},
		{
			Input: "func(args,",
			Err:   ErrSyntax,
		},
	}
	for _, d := range data {
		p, err := NewParser(strings.NewReader(d.Input))
		if err != nil {
			t.Errorf("fail to init parser")
			continue
		}
		obj := createObject()
		err = p.parse(obj)
		if d.Err != nil {
			if err == nil {
				t.Errorf("expected error but parse succeed!")
			} else if !errors.Is(err, d.Err) {
				t.Errorf("errors mismatched! want %s, got %s", d.Err, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("fail to parse %s: %s", d.Input, err)
			continue
		}
		fn, ok := obj.nodes["func"].(Func)
		if !ok {
			t.Errorf("func is not a function! want %T, got %T", fn, obj.nodes["func"])
			continue
		}
		if !testArguments(t, fn.args, d.Args) {
			continue
		}
	}
}

func testArguments(t *testing.T, got, want []Argument) bool {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("number of arguments mismatched! want %d, got %d", len(want), len(got))
		return false
	}
	for i := range want {
		if !want[i].name.Equal(got[i].name) {
			t.Errorf("argument name mismatched! want %s, got %s", want[i].name, got[i].name)
			return false
		}
		if want[i].isPositional() && !got[i].isPositional() {
			t.Errorf("expected poitional argument, but keyword argument: %s", got[i].name)
			return false
		}
		if want[i].isPositional() {
			continue
		}
		switch e := want[i].expr.(type) {
		case Literal:
			testLiteralValue(t, e, got[i].expr)
		default:
		}
	}
	return true
}

func TestParseOption(t *testing.T) {
	data := []struct {
		Input string
		Want  Expr
		Err   error
	}{
		{
			Input: "key = value",
			Want:  makeLiteral(makeToken("value", Ident)),
		},
		{
			Input: "key = 'value'",
			Want:  makeLiteral(makeToken("value", String)),
		},
		{
			Input: "key = 100_000",
			Want:  makeLiteral(makeToken("100000", Integer)),
		},
		{
			Input: "key = 0.31e-3",
			Want:  makeLiteral(makeToken("0.31e-3", Float)),
		},
		{
			Input: "key = false",
			Want:  makeLiteral(makeToken("false", Boolean)),
		},
		{
			Input: "key = $var",
			Want:  makeVariable(makeToken("var", LocalVar)),
		},
		{
			Input: "key = @var",
			Want:  makeVariable(makeToken("var", EnvVar)),
		},
		{
			Input: "key = ",
			Err:   ErrUnexpected,
		},
	}
	for _, d := range data {
		p, err := NewParser(strings.NewReader(d.Input))
		if err != nil {
			t.Errorf("fail to init parser: %s", err)
			continue
		}
		obj := createObject()
		err = p.parse(obj)
		if d.Err != nil {
			if err == nil {
				t.Errorf("expected error but parse succeed: %s", d.Input)
			} else if !errors.Is(err, d.Err) {
				t.Errorf("errors mismatched! want %s, got %s", d.Err, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("fail to parse %s: %s", d.Input, err)
			continue
		}
		testOptionValue(t, obj.nodes["key"], d.Want)
	}
}

func testOptionValue(t *testing.T, n Node, expr Expr) {
	t.Helper()
	opt, ok := n.(Option)
	if !ok {
		t.Errorf("expecting Option, got %T", n)
		return
	}
	switch e := expr.(type) {
	case Literal:
		testLiteralValue(t, e, opt.expr)
	}
}

func testLiteralValue(t *testing.T, want Literal, got Expr) {
	g, ok := got.(Literal)
	if !ok {
		t.Errorf("expecting Literal, got %T", got)
		return
	}
	if !want.tok.Equal(g.tok) {
		t.Errorf("literal mismatched! want %s, got %s", want.tok.Input, g.tok.Input)
	}
}

func TestParse(t *testing.T) {
	t.Run("mix", testParseMix)
	t.Run("package", testParseSimple)
}

func testParseMix(t *testing.T) {
	r, err := os.Open("testdata/main.fig")
	if err != nil {
		t.Fatalf("fail to open sample file")
		return
	}
	defer r.Close()

	if _, err = Parse(r); err != nil {
		t.Errorf("invalid document %s", err)
		return
	}
}

func testParseSimple(t *testing.T) {
	r, err := os.Open("testdata/package.fig")
	if err != nil {
		t.Fatalf("fail to open sample file")
		return
	}
	defer r.Close()

	obj, err := Parse(r)
	if err != nil {
		t.Errorf("invalid document %s", err)
		return
	}
	testOption(t, obj, []string{"package", "version"})
	testObject(t, obj, []string{"dev", "resource"})
	testList(t, obj, []string{"changelog"})

	dev, ok := obj.nodes["dev"].(*Object)
	if !ok {
		t.Errorf("dev: expected %T, got %T", dev, obj.nodes["resource"])
		return
	}
	testOption(t, dev, []string{"repo"})
	testList(t, dev, []string{"mail"})

	sub, ok := obj.nodes["resource"].(*Object)
	if !ok {
		t.Errorf("resource: expected %T, got %T", sub, obj.nodes["resource"])
		return
	}
	testList(t, sub, []string{"doc", "binary"})
}

func testObject(t *testing.T, obj *Object, keys []string) {
	t.Helper()
	for _, k := range keys {
		o, ok := obj.nodes[k]
		if !ok {
			t.Errorf("%s(object): key not found!", k)
			continue
		}
		opt, ok := o.(*Object)
		if !ok {
			t.Errorf("%s(object): types mismatched! expected %T, got %T", k, opt, o)
			continue
		}
	}
}

func testList(t *testing.T, obj *Object, keys []string) {
	t.Helper()
	for _, k := range keys {
		o, ok := obj.nodes[k]
		if !ok {
			t.Errorf("%s(list): key not found!", k)
			continue
		}
		opt, ok := o.(List)
		if !ok {
			t.Errorf("%s(list): types mismatched! expected %T, got %T", k, opt, o)
			continue
		}
	}
}

func testOption(t *testing.T, obj *Object, keys []string) {
	t.Helper()
	for _, k := range keys {
		o, ok := obj.nodes[k]
		if !ok {
			t.Errorf("%s(option): key not found!", k)
			continue
		}
		opt, ok := o.(Option)
		if !ok {
			t.Errorf("%s(option): types mismatched! expected %T, got %T", k, opt, o)
			continue
		}
	}
}
