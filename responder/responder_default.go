package responder

import (
	"github.com/vizicist/palette/engine"
)

func init() {
	// engine.AddResponder("default", NewResponder_default))
	engine.AddResponder("default", &Responder_default{})
}

type Responder_default struct {
}

func (r *Responder_default) OnMidiEvent(ctx *engine.ResponderContext, me engine.MidiEvent) {
	engine.DebugLogOfType("midi", "Responder_default.onMidiEvent", "me", me)
	// just echo it back out
	ctx.ScheduleBytesNow(me.Bytes)
}

func (r *Responder_default) OnCursorEvent(ctx *engine.ResponderContext, ce engine.CursorEvent) {
	if ce.Ddu == "down" || ce.Ddu == "drag" {

		pitch := uint8(ce.X * 126.0)
		velocity := uint8(ce.Z * 1280)
		duration := engine.QuarterNote
		synth := "0103 Ambient_E-Guitar"
		nt := engine.NewNote(pitch, velocity, duration, synth)

		ctx.Log("Responder_default.OnCursorEvent", "pitch", pitch, "vel", velocity, "dur", duration)

		phr := engine.NewPhrase().InsertNote(nt)
		ctx.SchedulePhraseNow(phr)
	}
}
