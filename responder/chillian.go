package responder

import (
	"github.com/vizicist/palette/engine"
)

func init() {
	RegisterResponder("chillian", &Responder_chillian{})
}

type Responder_chillian struct {
}

func (r *Responder_chillian) OnMidiEvent(ctx *engine.ResponderContext, me engine.MidiEvent) {
	ctx.Log("Responder_chillian.onMidiEvent", "me", me)
	// just echo it back out
	// ctx.ScheduleBytesNow(me.Bytes)
}

func (r *Responder_chillian) OnCursorEvent(ctx *engine.ResponderContext, ce engine.CursorEvent) {
	if ce.Ddu == "down" { // || ce.Ddu == "drag" {

		pitch := uint8(ce.X * 126.0)
		velocity := uint8(ce.Z * 1280)
		duration := 2 * engine.QuarterNote
		synth := "0103 Ambient_E-Guitar"
		nt := engine.NewNoteFull(pitch, velocity, duration, synth)

		ctx.Log("Responder_chillian.OnCursorEvent", "pitch", pitch, "vel", velocity, "dur", duration)

		pe := &engine.PhraseElement{
			AtClick: ctx.CurrentClick(),
			Source:  "chillian",
			Value:   nt,
		}
		phr := engine.NewPhrase().InsertElement(pe)
		ctx.SchedulePhraseNow(phr)
	}
}
