package main

import (
	"log"

	"github.com/nats-io/nats.go"
	"github.com/vizicist/palette/hostwin"
	"github.com/vizicist/palette/kit"
	"github.com/vizicist/palette/twinsys"
)

func main() {

	err := kit.RegisterAndInit(hostwin.NewHost("palette_control"))
	if err != nil {
		log.Fatal(err)
	}

	kit.LogInfo("palette_control started")
	if kit.TheNats != nil {
		err = kit.TheNats.Subscribe(">", myMsgHandler)
		kit.LogIfError(err)
	}
	twinsys.Run()
	select {}
}

func myMsgHandler(msg *nats.Msg) {
	data := string(msg.Data)
	kit.LogInfo("myMsgHandler", "msg", msg.Subject, "data", data)
}
