package fig_test

import (
	"os"
	"testing"

	"github.com/midbel/fig"
)

func TestParser(t *testing.T) {
	r, err := os.Open("testdata/spec.fig")
	if err != nil {
		t.Fatalf("fail to open file spec file")
	}
	defer r.Close()

	_, err = fig.Parse(r)
	if err != nil {
		t.Fatalf("fail to parse spec file: %s", err)
	}
}
