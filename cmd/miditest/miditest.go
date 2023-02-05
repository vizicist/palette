package main

import (
	"fmt"
	"github.com/vizicist/palette/rtmidi"
)

func main() {
	fmt.Printf("rtmidi start")
	tjt := rtmidi.Tjt()
	fmt.Printf("tjt = %d\n",tjt)
}
