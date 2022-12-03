package agent

import (
	"github.com/vizicist/palette/engine"
)

func init() {
	RegisterAgent("default", &Agent_default{})
}

type Agent_default struct {
	ctx *engine.AgentContext

	MIDIOctaveShift  int
	MIDINumDown      int
	MIDIThru         bool
	MIDIThruScadjust bool
	MIDISetScale     bool
	MIDIQuantized    bool
	MIDIUseScale     bool // if true, scadjust uses "external" Scale
	TransposePitch   int
	// externalScale    *engine.Scale
}

func (r *Agent_default) OnMidiEvent(ctx *engine.AgentContext, me engine.MidiEvent) {

	/*
		if r.ctx.MIDIThru {
			player.PassThruMIDI(e)
		}
		if player.MIDISetScale {
			r.handleMIDISetScaleNote(e)
		}
	*/
	ctx.Log("Agent_default.onMidiEvent", "me", me)
	phr := ctx.MidiEventToPhrase(me)
	if phr != nil {
		ctx.SchedulePhraseNow(phr)
	}
}

func (r *Agent_default) OnCursorEvent(ctx *engine.AgentContext, ce engine.CursorEvent) {
	r.ctx = ctx
	if ce.Ddu == "down" || ce.Ddu == "drag" {

		// pitch := uint8(ce.X * 126.0)
		pitch := r.cursorToPitch(ce)
		velocity := uint8(ce.Z * 1280)
		duration := 2 * engine.QuarterNote
		synth := "0103 Ambient_E-Guitar"
		ctx.ScheduleNoteNow(pitch, velocity, duration, synth)
	}
}

func (r *Agent_default) cursorToPitch(ce engine.CursorEvent) uint8 {
	pitchmin := r.ctx.ParamIntValue("sound.pitchmin")
	pitchmax := r.ctx.ParamIntValue("sound.pitchmax")
	dp := pitchmax - pitchmin + 1
	p1 := int(ce.X * float32(dp))
	p := uint8(pitchmin + p1%dp)
	/*
		chromatic := r.ctx.ParamBoolValue("sound.chromatic")
		if !chromatic {
			scale := r.ctx.GetScale()
			p = scale.ClosestTo(p)
			// MIDIOctaveShift might be negative
			i := int(p) + 12*player.MIDIOctaveShift
			for i < 0 {
				i += 12
			}
			for i > 127 {
				i -= 12
			}
			p = uint8(i + player.TransposePitch)
		}
	*/
	return p
}

/*
func (r *Agent_default) handleMIDISetScaleNote(e engine.MidiEvent) {
	status := e.Status() & 0xf0
	pitch := int(e.Data1())
	if status == 0x90 {
		/		// If there are no notes held down (i.e. this is the first), clear the scale
		if player.MIDINumDown < 0 {
			// this can happen when there's a Read error that misses a noteon
			player.MIDINumDown = 0
		}
		if player.MIDINumDown == 0 {
			player.clearExternalScale()
		}
		player.setExternalScale(pitch%12, true)
		player.MIDINumDown++
		if pitch < 60 {
			player.MIDIOctaveShift = -1
		} else if pitch > 72 {
			player.MIDIOctaveShift = 1
		} else {
			player.MIDIOctaveShift = 0
		}
	} else if status == 0x80 {
		player.MIDINumDown--
	}
}
*/
