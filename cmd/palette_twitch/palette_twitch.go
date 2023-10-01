package main

import (
	"log"

	"github.com/vizicist/palette/hostwin"
	"github.com/vizicist/palette/kit"
)

func main() {
	
	err := kit.RegisterAndInit(hostwin.NewHost("twitchtest"))
	if err != nil {
		log.Fatal(err)
	}

	kit.StartTwitch()
}
