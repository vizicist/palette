package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"github.com/vizicist/palette/parse"
)

func main() {

	out := flag.String("out", "", "output file")

	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Printf("Usage: palette_pk2go [-out={outputfile}] {inputfile}\n")
		os.Exit(1)
	}
	infname := args[0]
	inbase, found := strings.CutSuffix(infname, ".k")
	if !found {
		fmt.Printf("Error: only works on *.k files\n")
		os.Exit(1)
	}
	if *out == "" {
		outfname := inbase + ".go"
		out = &outfname
	}
	contents, err := os.ReadFile(infname)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	outf, err := os.Create(*out)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	outf.WriteString("// Generated by palette_pk2go, do not edit.\n")
	outf.WriteString("package kitlib\n\n")
	outf.WriteString("import (\n\t\"github.com/vizicist/palette/kit\"\n)\n")
	outf.WriteString("var dummy kit.Phrase\n")
	parser := parse.PkNewParser()
	lex := &parse.PkLex{
		Line: contents,
		Outf: outf,
	}
	r := parser.Parse(lex)
	if r != 0 {
		fmt.Printf("Parse returns ERROR!?\n")
	}
	outf.Close()
}
