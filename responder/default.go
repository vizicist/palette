package responder

import (
	"github.com/vizicist/palette/engine"
)

func init() {
	engine.AddResponder("default", &Responder_default{})
	AllResponders["default"] = &Responder_default{}
}

type Responder_default struct {
}

func (r *Responder_default) OnMidiEvent(ctx *engine.ResponderContext, me engine.MidiEvent) {
	pe := ctx.MidiEventToPhraseElement(me)
	ctx.Log("Responder_default.onMidiEvent", "me", me)
	if pe != nil {
		ctx.SchedulePhraseElementNow(pe)
	}
}

func (r *Responder_default) OnCursorEvent(ctx *engine.ResponderContext, ce engine.CursorEvent) {
	if ce.Ddu == "down" || ce.Ddu == "drag" {

		pitch := uint8(ce.X * 126.0)
		velocity := uint8(ce.Z * 1280)
		duration := engine.QuarterNote
		synth := "0103 Ambient_E-Guitar"
		pe := &engine.PhraseElement{
			AtClick: ctx.CurrentClick(),
			Source:  "default",
			Value:   engine.NewNoteFull(pitch, velocity, duration, synth),
		}
		ctx.SchedulePhraseElementNow(pe)
	}
}
