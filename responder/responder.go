package responder

import (
	"log"

	"github.com/nats-io/nats.go"
	"github.com/vizicist/palette/engine"
)

type Responder struct {
	subject       string
	onCursorEvent func(e engine.CursorDeviceEvent)
}

func NewResponder(subject string) *Responder {
	p := &Responder{subject: subject}
	return p
}

func (p *Responder) OnCursorEvent(f func(e engine.CursorDeviceEvent)) {
	p.onCursorEvent = f
}

func (p *Responder) RunForever() error {

	log.Printf("Responder.RunForever: Subscribing to %s\n", engine.PaletteOutputEventSubject)
	err := engine.SubscribeNATS(engine.PaletteOutputEventSubject, p.responderNATSCallback)
	if err != nil {
		return err
	}
	select {} // block forever
}

func (responder *Responder) responderNATSCallback(msg *nats.Msg) {
	data := string(msg.Data)
	args, err := engine.StringMap(data)
	if err != nil {
		log.Printf("natsCallback: err=%s\n", err)
		return
	}
	responder.Respond(args)
}

func (responder *Responder) Respond(args map[string]string) {
	log.Printf("responderNATSCallback args=%+v\n", args)
	switch args["event"] {
	case "cursor_down", "cursor_drag", "cursor_up":
		ce := engine.ArgsToCursorDeviceEvent(args)
		if responder.onCursorEvent != nil {
			responder.onCursorEvent(ce)
		}
	}
}
