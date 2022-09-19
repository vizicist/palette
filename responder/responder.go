package responder

import (
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
	"github.com/vizicist/palette/engine"
)

// This package is the external interface that Responders use
// to register themselves, receive events, and do things
// like play notes or spawn sprites.
// Should probably be a separate package.

type Responder struct {
	onCursorEvent func(e engine.CursorDeviceEvent)
}

func NewResponder() *Responder {
	p := &Responder{}
	return p
}

func (p *Responder) OnCursorEvent(f func(e engine.CursorDeviceEvent)) {
	p.onCursorEvent = f
}

func ResponderOutputSubject(id string) string {
	return fmt.Sprintf("responder.output.%s", id)
}

type ResponderCallback func(args map[string]string)

func (p *Responder) RunForever() error {

	err := engine.SubscribeNATS(engine.PaletteOutputEventSubject, p.responderNATSCallback)
	if err != nil {
		return err
	}
	select {} // block forever
}

func (responder *Responder) responderNATSCallback(msg *nats.Msg) {
	data := string(msg.Data)
	args, err := engine.StringMap(data)
	log.Printf("responderNATSCallback: args=%v\n", args)
	if err != nil {
		log.Printf("natsCallback: err=%s\n", err)
		return
	}
	log.Printf("responderNATSCallback args=%+v\n", args)
	switch args["event"] {
	case "cursor_down", "cursor_drag", "cursor_up":
		ce := engine.ArgsToCursorDeviceEvent(args)
		if responder.onCursorEvent != nil {
			responder.onCursorEvent(ce)
		}
	}
}

/*
// PlayNote is intended for use by a responder, to play a Note
func PlayNote(note *Note, source string) error {

	params := JsonObject("source", source, "note", note.String())
	args := JsonObject(
		// "nuid", MyNUID(),
		"api", "sound.playnote",
		"params", jsonEscape(params),
	)
	return NATSPublish(PaletteAPISubject, args)
}
*/
