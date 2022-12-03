package agent

import (
	"github.com/vizicist/palette/engine"
)

func init() {
	RegisterAgent("chillian", &Agent_chillian{})
}

type Agent_chillian struct {
}

func (r *Agent_chillian) Start(ctx *engine.AgentContext) {
	engine.Info("Agent_chillian.Start")
}

func (r *Agent_chillian) OnMidiEvent(ctx *engine.AgentContext, me engine.MidiEvent) {
	ctx.Log("Agent_chillian.onMidiEvent", "me", me)
	// just echo it back out
	// ctx.ScheduleBytesNow(me.Bytes)
}

func (r *Agent_chillian) OnCursorEvent(ctx *engine.AgentContext, ce engine.CursorEvent) {
	if ce.Ddu == "down" { // || ce.Ddu == "drag" {

		pitch := uint8(ce.X * 126.0)
		velocity := uint8(ce.Z * 1280)
		duration := 2 * engine.QuarterNote
		synth := "0103 Ambient_E-Guitar"
		nt := engine.NewNoteFull(pitch, velocity, duration, synth)

		ctx.Log("Agent_chillian.OnCursorEvent", "pitch", pitch, "vel", velocity, "dur", duration)

		pe := &engine.PhraseElement{
			AtClick: ctx.CurrentClick(),
			Source:  "chillian",
			Value:   nt,
		}
		phr := engine.NewPhrase().InsertElement(pe)
		ctx.SchedulePhraseNow(phr)
	}
}
