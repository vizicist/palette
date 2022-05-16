package main

import (
	"fmt"

	"github.com/vizicist/palette/kit"
)

var vars = map[string]interface{}{
	"A": 1,
	"B": 1,
}

func main() {
	eval("print(999)")
	// eval("A IS B")
}

func eval(s string) {
	err := kit.Parse(s, vars)
	fmt.Printf("s=%s err=%v\n", s, err)
}
