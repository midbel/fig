package fig

import (
	"strings"
	"testing"
)

const sample = `
package = fig
version = "1.0.0"

dev {
  mail = "dev@midbel.org"
  mail = "noreply@midbel.org"
  mail = "info@midbel.org"
  repo = "https://github.com/midbel"
}

changelog {
  date    = 2021-05-07
  version = join([0, 2, 0], ".")
}

changelog {
  date    = 2021-05-07
  version = join([0, 1, 0], ".")
}

resource binary {
  path = "bin/figdebug"
  mode = 0o755
}

resource binary {
  path = "bin/figtest"
  mode = 0o755
}

resource doc {
  type = man
  path = "docs/fig.1.gz"
  mode = 0o644
}

resource doc {
  type = readme
  path = "docs/README"
  mode = 0o644
}
`

func TestParse(t *testing.T) {
	obj, err := Parse(strings.NewReader(sample))
	if err != nil {
		t.Errorf("invalid document %s", err)
		return
	}
	testOption(t, obj, []string{"package", "version"})
	testObject(t, obj, []string{"dev", "resource"})
	testList(t, obj, []string{"changelog"})

	dev, ok := obj.nodes["dev"].(*Object)
	if !ok {
		t.Errorf("rev: expected %T, got %T", dev, obj.nodes["resource"])
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
			t.Errorf("object %s: key not found!", k)
			continue
		}
		opt, ok := o.(*Object)
		if !ok {
			t.Errorf("types mismatched! expected %T, got %T", opt, o)
			continue
		}
	}
}

func testList(t *testing.T, obj *Object, keys []string) {
	t.Helper()
	for _, k := range keys {
		o, ok := obj.nodes[k]
		if !ok {
			t.Errorf("list %s: key not found!", k)
			continue
		}
		opt, ok := o.(List)
		if !ok {
			t.Errorf("types mismatched! expected %T, got %T", opt, o)
			continue
		}
	}
}

func testOption(t *testing.T, obj *Object, keys []string) {
	t.Helper()
	for _, k := range keys {
		o, ok := obj.nodes[k]
		if !ok {
			t.Errorf("option %s: key not found!", k)
			continue
		}
		opt, ok := o.(Option)
		if !ok {
			t.Errorf("types mismatched! expected %T, got %T", opt, o)
			continue
		}
	}
}
