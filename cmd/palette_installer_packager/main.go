package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vizicist/palette/internal/installerbundle"
)

func main() {
	stub := flag.String("stub", "", "path to the native installer stub")
	source := flag.String("source", "", "payload directory")
	output := flag.String("output", "", "output installer path")
	kind := flag.String("kind", "", "installer kind: app or data")
	version := flag.String("version", "", "Palette version")
	dataName := flag.String("data-name", "", "data set name for a data installer")
	deleteNames := flag.String("delete", "", "comma-separated relative paths removed during install")
	flag.Parse()

	if *stub == "" || *source == "" || *output == "" || *kind == "" || *version == "" {
		flag.Usage()
		os.Exit(2)
	}
	manifest := installerbundle.Manifest{
		Kind:     *kind,
		Version:  *version,
		DataName: *dataName,
	}
	if *deleteNames != "" {
		manifest.Delete = strings.Split(*deleteNames, ",")
	}
	if err := installerbundle.Pack(*stub, *source, *output, manifest); err != nil {
		fmt.Fprintln(os.Stderr, "pack installer:", err)
		os.Exit(1)
	}
	fmt.Println("Installer created:", *output)
}
