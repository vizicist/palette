package main

import (
	"log"

	"github.com/vizicist/palette/hostwin"
	"github.com/vizicist/palette/twinsys"
	"github.com/vizicist/palette/kit"
	"github.com/nats-io/nats.go"
)

func main() {
	
	err := kit.RegisterAndInit(hostwin.NewHost("palette_control"))
	if err != nil {
		log.Fatal(err)
	}

	kit.LogInfo("palette_control started")
	err = kit.TheNats.Subscribe(">",myMsgHandler)
	if err != nil {
		kit.LogError(err)
		return
	}
	twinsys.Run()
	select {}
}

func myMsgHandler(msg *nats.Msg) {
	data := string(msg.Data)
	kit.LogInfo("myMsgHandler", "msg", msg.Subject, "data", data)
}
