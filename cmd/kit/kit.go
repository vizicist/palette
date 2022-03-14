package main

import (
	"fmt"
	"github.com/vizicist/palette/kit"
)


func main() {
	vars := map[string]interface{}{
		"A": 1,
		"B": 1,
	}
	f := kit.Parse("NOT (A IS B)", vars)
	fmt.Printf("%t\n", f) // false

	t := kit.Parse("A IS B", vars)
	fmt.Printf("%t\n", t) // true
}
