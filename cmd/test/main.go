package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/midbel/fig"
)

func main() {
	scan := flag.Bool("s", false, "scan file")
	flag.Parse()
	r, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer r.Close()

	if *scan {
		err = scanFile(r)
	} else {
		err = parseFile(r)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func scanFile(r io.Reader) error {
	s, err := fig.Scan(r)
	if err != nil {
		return err
	}
	for i := 0; ; i++ {
		tok := s.Scan()
		if tok.Type == fig.EOF {
			break
		}
		if tok.Type == fig.Invalid || tok.Type == 0 {
			return fmt.Errorf("invalid token found %s", tok)
		}
		fmt.Println(i, tok.Position, tok)
	}
	return nil
}

func parseFile(r io.Reader) error {
	_, err := fig.Parse(r)
	return err
}
