package responder

import (
	"github.com/vizicist/palette/engine"
)

func init() {
	RegisterResponder("default", &Responder_default{})
}

type Responder_default struct {
}

func (r *Responder_default) OnMidiEvent(ctx *engine.ResponderContext, me engine.MidiEvent) {
	ctx.Log("Responder_default.onMidiEvent", "me", me)
	phr := ctx.MidiEventToPhrase(me)
	if phr != nil {
		ctx.SchedulePhraseNow(phr)
	}
}

func (r *Responder_default) OnCursorEvent(ctx *engine.ResponderContext, ce engine.CursorEvent) {
	if ce.Ddu == "down" { // || ce.Ddu == "drag" {

		pitch := uint8(ce.X * 126.0)
		velocity := uint8(ce.Z * 1280)
		duration := engine.QuarterNote
		synth := "0103 Ambient_E-Guitar"
		ctx.ScheduleNoteNow(pitch, velocity, duration, synth)
	}
}
