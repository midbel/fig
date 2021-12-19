package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/midbel/fig"
)

func main() {
	var (
		scan  = flag.Bool("s", false, "scan file")
		parse = flag.Bool("p", false, "parse file")
		debug = flag.Bool("d", false, "debug file")
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
	} else if *debug {
		err = fig.Debug(r, os.Stdout)
	} else {
		err = decodeFile(r)
	}
	fmt.Println("elapsed:", time.Since(now))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func decodeFile(r io.Reader) error {
	var (
		dat  = make(map[string]interface{})
		dec  = fig.NewDecoder(r)
		fmap = fig.FuncMap{
			"repeat": strings.Repeat,
			"upper":  strings.ToUpper,
			"lower":  strings.ToLower,
			"join":   strings.Join,
			"uuid3":  func(str string) string { return str },
		}
	)
	dec.Funcs(fmap)
	dec.Define("env", "environment")
	if err := dec.Decode(&dat); err != nil {
		return err
	}
	pprint(dat)
	return nil
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
	n, err := fig.Parse(r)
	if err == nil {
		fmt.Printf("node: %#v\n", n)
	}
	return err
}

func pprint(data map[string]interface{}) {
	pretty(data, 0)
	fmt.Println()
}

func pretty(data map[string]interface{}, level int) {
	fmt.Println("{")
	prefix := strings.Repeat(" ", level)
	for k, v := range data {
		fmt.Printf("%s  %s: ", prefix, k)
		switch dat := v.(type) {
		case []interface{}:
			prettyArray(dat, level)
		case map[string]interface{}:
			pretty(dat, level+2)
		default:
			fmt.Printf("%v", dat)
		}
		fmt.Println()
	}
	fmt.Printf("%s}", prefix)
}

func prettyArray(data []interface{}, level int) {
	fmt.Print("[")
	for i := range data {
		if i > 0 {
			fmt.Print(", ")
		}
		switch dat := data[i].(type) {
		case map[string]interface{}:
			pretty(dat, level+2)
		case []interface{}:
			prettyArray(dat, level)
		default:
			fmt.Printf("%v", dat)
		}
	}
	fmt.Print("]")
}
