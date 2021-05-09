package fig

import (
	"os"
	"testing"
)

func TestParse(t *testing.T) {
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
