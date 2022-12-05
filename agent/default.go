package agent

import (
	"fmt"

	"github.com/vizicist/palette/engine"
)

func init() {
	RegisterAgent("default", &Agent_default{
		layer: map[string]*engine.Layer{},
	})
}

type Agent_default struct {
	ctx   *engine.AgentContext
	layer map[string]*engine.Layer

	// layers      []string
	// layerParams map[string]*engine.ParamValues

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

func (agent *Agent_default) Start(ctx *engine.AgentContext) {

	engine.Info("Agent_default.Start")

	/*
		presets := map[string]string{
			"A": "snap.Yellow_Spheres",
			"B": "snap.Blue_Spheres",
			"C": "snap.Orange_Spheres",
			"D": "snap.Pink_Spheres",
		}
		shapes := map[string]string{
			"A": "square",
			"B": "circle",
			"C": "triangle",
			"D": "line",
		}
	*/

	ctx.AllowSource("A", "B", "C", "D")

	a := ctx.MakeLayer("layerA")
	a.Set("visual.resolumeport", "3334")
	a.Set("visual.shape", "circle")
	a.Apply(ctx.GetPreset("snap.White_Ghosts"))
	agent.layer["a"] = a

	b := ctx.MakeLayer("layerB")
	b.Set("visual.resolumeport", "3335")
	b.Set("visual.shape", "square")
	b.Apply(ctx.GetPreset("snap.Concentric_Squares"))
	agent.layer["b"] = b

	c := ctx.MakeLayer("layerC")
	c.Set("visual.resolumeport", "3336")
	c.Set("visual.shape", "square")
	c.Apply(ctx.GetPreset("snap.Circular_Moire"))
	agent.layer["c"] = c

	d := ctx.MakeLayer("layerD")
	d.Set("visual.resolumeport", "3337")
	d.Set("visual.shape", "square")
	d.Apply(ctx.GetPreset("snap.Diagonal_Mirror"))
	agent.layer["d"] = d

	ctx.ApplyPreset("quad.Quick Scat_Circles")

	agent.ctx = ctx
}

func (agent *Agent_default) OnMidiEvent(me engine.MidiEvent) {

	if agent.ctx == nil {
		engine.LogError(fmt.Errorf("OnMidiEvent: Start needs to be called before this"))
		return
	}

	ctx := agent.ctx

	/*
		if r.ctx.MIDIThru {
			player.PassThruMIDI(e)
		}
		if player.MIDISetScale {
			r.handleMIDISetScaleNote(e)
		}
	*/
	ctx.Log("Agent_default.onMidiEvent", "me", me)
	phr, err := ctx.MidiEventToPhrase(me)
	if err != nil {
		engine.LogError(err)
	}
	if phr != nil {
		ctx.SchedulePhraseNow(phr)
	}
}

func (agent *Agent_default) OnCursorEvent(ce engine.CursorEvent) {

	if agent.ctx == nil {
		engine.LogError(fmt.Errorf("OnMidiEvent: Start needs to be called before this"))
		return
	}

	ctx := agent.ctx

	if ce.Ddu == "down" || ce.Ddu == "drag" {
		channel := agent.cursorToChannel(ce)
		pitch := agent.cursorToPitch(ce)
		velocity := uint8(ce.Z * 1280)
		duration := 2 * engine.QuarterNote
		ctx.ScheduleNoteNow(channel, pitch, velocity, duration)
	}
}

func (agent *Agent_default) cursorToChannel(ce engine.CursorEvent) uint8 {
	return 0
}

func (agent *Agent_default) cursorToPitch(ce engine.CursorEvent) uint8 {
	pitchmin := agent.ctx.ParamIntValue("sound.pitchmin")
	pitchmax := agent.ctx.ParamIntValue("sound.pitchmax")
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
