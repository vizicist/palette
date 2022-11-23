package responder

import (
	"github.com/vizicist/palette/engine"
)

func init() {
	engine.AddResponder("logger", &Responder_logger{})
	AllResponders["logger"] = &Responder_logger{}
}

type Responder_logger struct{}

func (r *Responder_logger) OnMidiEvent(ctx *engine.ResponderContext, me engine.MidiEvent) {
	engine.Info("Responder_logger.onMidiEvent", "me", me)
}

func (r *Responder_logger) OnCursorEvent(ctx *engine.ResponderContext, ce engine.CursorEvent) {
	engine.Info("Responder_logger.onCursorEvent", "ce", ce)
}
