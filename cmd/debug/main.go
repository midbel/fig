package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/midbel/fig"
)

func main() {
	quiet := flag.Bool("q", false, "quiet")
	flag.Parse()
	r, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer r.Close()

	var w io.Writer = io.Discard
	if !*quiet {
		w = os.Stdout
	}
	if err := fig.Debug(w, r); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
