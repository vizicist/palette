package responder

import (
	"github.com/vizicist/palette/engine"
)

func init() {
	engine.AddResponder("default", NewResponder_default())
}

type Responder_default struct {
	// ctx *engine.ResponderContext
}

func NewResponder_default() *Responder_default {
	p := &Responder_default{}
	return p
}

/////////////////////////// external interface

func (r *Responder_default) OnCursorEvent(ctx *engine.ResponderContext, ce engine.CursorEvent) {
	clicks := ctx.CurrentClick()
	if ce.Ddu == "down" || ce.Ddu == "drag" {
		nt := r.cursorToNote(ce)
		engine.Info("Responder_default.OnCursorEvent", "ce", ce, "note", nt)
		ctx.ScheduleNoteAt(nt, clicks)
		engine.Info("Schedule is now", "schedule", ctx.ScheduleDebug())
	}

}

/////////////////////////// internal things

func (r *Responder_default) cursorToNote(ce engine.CursorEvent) *engine.Note {
	pitch := uint8(ce.X * 126.0)
	velocity := uint8(ce.Z * 128.0)
	duration := 2 * engine.QuarterNote
	synth := "0103 Ambient_E-Guitar"
	return engine.NewNote(pitch, velocity, duration, synth)
}
