package main

import (
	"fmt"
	"github.com/vizicist/palette/rtmidi"
)

func main() {
	fmt.Printf("rtmidi start")
	apis := rtmidi.CompiledAPI()
	fmt.Printf("compiled apis = %v\n",apis)
}
