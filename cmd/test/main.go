package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/midbel/fig"
)

func main() {
	var (
		scan  = flag.Bool("s", false, "scan file")
		parse = flag.Bool("p", false, "parse file")
	)
	flag.Parse()
	r, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer r.Close()

	now := time.Now()
	if *scan {
		err = scanFile(r)
	} else if *parse {
		err = parseFile(r)
	} else {
		// err = queryFile(r, flag.Args())
	}
	fmt.Println("elapsed:", time.Since(now))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// func queryFile(r io.Reader, key []string) error {
// 	doc, err := fig.ParseDocument(r)
// 	if err != nil {
// 		return err
// 	}
// 	for _, k := range key[1:] {
// 		ks := strings.Split(k, "/")
// 		str, err := doc.Value(ks...)
// 		if err != nil {
// 			return fmt.Errorf("%s: %s", k, err)
// 		}
// 		fmt.Printf("%s: %v", k, str)
// 		fmt.Println()
// 	}
// 	return nil
// }

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
	return fig.Parse(r)
}
