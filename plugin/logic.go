package plugin

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/hypebeast/go-osc/osc"
	"github.com/vizicist/palette/engine"
)

type LayerLogic struct {
	ppro  *PalettePro
	layer *engine.Layer

	cursorNote map[string]*engine.NoteOn
	// tempoFactor            float64
	// loop            *StepLoop
	// loopLength      engine.Clicks
	// loopIsRecording bool
	// loopIsPlaying   bool
	// fadeLoop        float32

	// lastCursorStepEvent    CursorStepEvent
	// lastUnQuantizedStepNum Clicks

	// params                         map[string]any

	// paramsMutex               sync.RWMutex

	// Things moved over from Router

	// MIDINumDown      int
	// MIDIThru         bool
	// MIDIThruScadjust bool
	// MIDISetScale     bool
	// MIDIQuantized    bool
	// TransposePitch   int
}

func NewLayerLogic(ppro *PalettePro, layer *engine.Layer) *LayerLogic {
	logic := &LayerLogic{
		ppro:       ppro,
		layer:      layer,
		cursorNote: map[string]*engine.NoteOn{},
	}
	return logic
}

func (logic *LayerLogic) cursorToNoteOn(ctx *engine.PluginContext, ce engine.CursorEvent) *engine.NoteOn {
	pitch := logic.cursorToPitch(ctx, ce)
	velocity := logic.cursorToVelocity(ctx, ce)
	channel := logic.layer.MIDIChannel()
	return engine.NewNoteOn(channel, pitch, velocity)
}

func (logic *LayerLogic) cursorToPitch(ctx *engine.PluginContext, ce engine.CursorEvent) uint8 {
	layer := logic.layer
	ppro := logic.ppro
	pitchmin := layer.GetInt("sound.pitchmin")
	pitchmax := layer.GetInt("sound.pitchmax")
	dp := pitchmax - pitchmin + 1
	p1 := int(ce.X * float32(dp))
	p := uint8(pitchmin + p1%dp)

	usescale := layer.GetBool("misc.usescale")
	chromatic := layer.GetBool("sound.chromatic")
	if !chromatic && usescale {
		scaleName := layer.GetString("misc.scale")
		scale := engine.GetScale(scaleName)
		p = scale.ClosestTo(p)
		// MIDIOctaveShift might be negative
		i := int(p) + 12*ppro.MIDIOctaveShift
		for i < 0 {
			i += 12
		}
		for i > 127 {
			i -= 12
		}
		p = uint8(i + ppro.TransposePitch)
	}
	return p
}

func (logic *LayerLogic) cursorToVelocity(ctx *engine.PluginContext, ce engine.CursorEvent) uint8 {
	layer := logic.layer
	vol := layer.Get("misc.vol")
	velocitymin := layer.GetInt("sound.velocitymin")
	velocitymax := layer.GetInt("sound.velocitymax")
	// bogus, when values in json are missing
	if velocitymin == 0 && velocitymax == 0 {
		velocitymin = 0
		velocitymax = 127
	}
	if velocitymin > velocitymax {
		t := velocitymin
		velocitymin = velocitymax
		velocitymax = t
	}
	v := float32(0.8) // default and fixed value
	switch vol {
	case "frets":
		v = 1.0 - ce.Y
	case "pressure":
		v = ce.Z * 4.0
	case "fixed":
		// do nothing
	default:
		engine.LogWarn("Unrecognized vol value", "vol", vol)
	}
	dv := velocitymax - velocitymin + 1
	p1 := int(v * float32(dv))
	vel := uint8(velocitymin + p1%dv)
	return uint8(vel)
}

func (logic *LayerLogic) generateVisualsFromCursor(ce engine.CursorEvent) {
	// send an OSC message to Resolume
	msg := osc.NewMessage("/cursor")
	msg.Append(ce.Ddu)
	msg.Append(ce.Cid)
	msg.Append(float32(ce.X))
	msg.Append(float32(ce.Y))
	msg.Append(float32(ce.Z))
	logic.ppro.resolume.toFreeFramePlugin(logic.layer.Name(), msg)
}

func (logic *LayerLogic) generateSoundFromCursor(ctx *engine.PluginContext, ce engine.CursorEvent) {

	layer := logic.layer
	switch ce.Ddu {
	case "down":
		_, ok := logic.cursorNote[ce.Cid]
		if ok {
			engine.LogWarn("generateSoundFromCursor: cursorNote already exists", "cid", ce.Cid)
		}
		noteOn := logic.cursorToNoteOn(ctx, ce)
		logic.sendNoteOn(noteOn)
		logic.cursorNote[ce.Cid] = noteOn
	case "drag":
		cursorState := ctx.GetCursorState(ce.Cid)
		if cursorState == nil {
			engine.LogWarn("generateSoundFromCursor: cursorState is nil", "cid", ce.Cid)
			return
		}
		oldNoteOn, ok := logic.cursorNote[ce.Cid]
		if !ok {
			engine.LogWarn("generateSoundFromCursor: no cursorNote", "cid", ce.Cid)
			return
		}
		newNoteOn := logic.cursorToNoteOn(ctx, ce)
		oldpitch := oldNoteOn.Pitch
		newpitch := newNoteOn.Pitch
		// We only turn off the existing note (for a given Cursor ID)
		// and start the new one if the pitch changes

		// Also do this if the Z/Velocity value changes more than the trigger value

		// NOTE: this could and perhaps should use a.ce.Z now that we're
		// saving a.ce, like the deltay value

		dz := float64(int(oldNoteOn.Velocity) - int(newNoteOn.Velocity))
		deltaz := float32(math.Abs(dz) / 128.0)
		deltaztrig := layer.GetFloat("sound._deltaztrig")

		deltay := float32(math.Abs(float64(cursorState.Previous.Y - ce.Y)))
		deltaytrig := layer.GetFloat("sound._deltaytrig")

		if layer.Get("sound.controllerstyle") == "modulationonly" {
			zmin := layer.GetFloat("sound._controllerzmin")
			zmax := layer.GetFloat("sound._controllerzmax")
			cmin := layer.GetInt("sound._controllermin")
			cmax := layer.GetInt("sound._controllermax")
			oldz := cursorState.Previous.Z
			newz := ce.Z
			// XXX - should put the old controller value in ActiveNote so
			// it doesn't need to be computed every time
			oldzc := BoundAndScaleController(oldz, zmin, zmax, cmin, cmax)
			newzc := BoundAndScaleController(newz, zmin, zmax, cmin, cmax)

			if newzc != 0 && newzc != oldzc {
				logic.layer.Synth.SendController(1, newzc)
			}
		}

		if newpitch != oldpitch || deltaz > deltaztrig || deltay > deltaytrig {
			// Turn off existing note
			logic.sendNoteOff(oldNoteOn)
			engine.LogInfo("Replacing cursorNote", "cid", ce.Cid)
			logic.cursorNote[ce.Cid] = newNoteOn
			logic.sendNoteOn(newNoteOn)
		}
	case "up":
		cursorState := ctx.GetCursorState(ce.Cid)
		if cursorState == nil {
			engine.LogWarn("generateSoundFromCursor: cursorState is nil", "cid", ce.Cid)
			return
		}
		oldNoteOn, ok := logic.cursorNote[ce.Cid]
		if !ok {
			// not sure why this happens, yet
			engine.LogWarn("Unexpected UP, no cursorNote", "cid", ce.Cid)
		} else {
			logic.sendNoteOff(oldNoteOn)
			delete(logic.cursorNote, ce.Cid)
		}
	}
}

func (logic *LayerLogic) clearGraphics() {
	// send an OSC message to Resolume
	logic.ppro.resolume.toFreeFramePlugin(logic.layer.Name(), osc.NewMessage("/clear"))
}

func (logic *LayerLogic) nextQuant(t engine.Clicks, q engine.Clicks) engine.Clicks {
	// the algorithm below is the same as KeyKit's nextquant
	if q <= 1 {
		return t
	}
	tq := t
	rem := tq % q
	if (rem * 2) > q {
		tq += (q - rem)
	} else {
		tq -= rem
	}
	if tq < t {
		tq += q
	}
	return tq
}

func (logic *LayerLogic) sendNoteOn(note any) {

	logic.layer.Synth.SendTo(note)

	// ss := logic.layer.Get("visual.spritesource")
	// if ss == "midi" {
	// 	logic.generateSpriteFromPhraseElement(ctx, pe)
	// }
}

/*
func (logic *LayerLogic) SendPhraseElementToSynth(ctx *engine.PluginContext, pe *engine.PhraseElement) {

	ss := logic.layer.Get("visual.spritesource")
	if ss == "midi" {
		logic.generateSpriteFromPhraseElement(ctx, pe)
	}
	logic.layer.Synth.SendTo(pe)
}
*/

func (logic *LayerLogic) generateSpriteFromPhraseElement(ctx *engine.PluginContext, pe *engine.PhraseElement) {

	layer := logic.layer

	// var channel uint8
	var pitch uint8
	var velocity uint8

	switch v := pe.Value.(type) {
	case *engine.NoteOn:
		// channel = v.Channel
		pitch = v.Pitch
		velocity = v.Velocity
	case *engine.NoteOff:
		// channel = v.Channel
		pitch = v.Pitch
		velocity = v.Velocity
	case *engine.NoteFull:
		// channel = v.Channel
		pitch = v.Pitch
		velocity = v.Velocity
	default:
		return
	}

	pitchmin := uint8(layer.GetInt("sound.pitchmin"))
	pitchmax := uint8(layer.GetInt("sound.pitchmax"))
	if pitch < pitchmin || pitch > pitchmax {
		engine.LogWarn("Unexpected value", "pitch", pitch)
		return
	}

	var x float32
	var y float32
	switch layer.Get("visual.placement") {
	case "random", "":
		x = rand.Float32()
		y = rand.Float32()
	case "linear":
		y = 0.5
		x = float32(pitch-pitchmin) / float32(pitchmax-pitchmin)
	case "cursor":
		x = rand.Float32()
		y = rand.Float32()
	case "top":
		y = 1.0
		x = float32(pitch-pitchmin) / float32(pitchmax-pitchmin)
	case "bottom":
		y = 0.0
		x = float32(pitch-pitchmin) / float32(pitchmax-pitchmin)
	case "left":
		y = float32(pitch-pitchmin) / float32(pitchmax-pitchmin)
		x = 0.0
	case "right":
		y = float32(pitch-pitchmin) / float32(pitchmax-pitchmin)
		x = 1.0
	default:
		x = rand.Float32()
		y = rand.Float32()
	}

	// send an OSC message to Resolume
	msg := osc.NewMessage("/sprite")
	msg.Append(x)
	msg.Append(y)
	msg.Append(float32(velocity) / 127.0)

	// Someday localhost should be changed to the actual IP address.
	// XXX - Set sprite ID to pitch, is this right?
	msg.Append(fmt.Sprintf("%d@localhost", pitch))

	logic.ppro.resolume.toFreeFramePlugin(layer.Name(), msg)
}

func (logic *LayerLogic) sendNoteOff(n *engine.NoteOn) {
	if n == nil {
		// Not sure why this sometimes happens
		return
	}
	noteOff := engine.NewNoteOff(n.Channel, n.Pitch, n.Velocity)
	// pe := &engine.PhraseElement{Value: noteOff}
	logic.layer.Synth.SendTo(noteOff)
	// layer.SendPhraseElementToSynth(pe)
}

/*
func (logic *LayerLogic) advanceTransposeTo(newclick engine.Clicks) {

	ppro := logic.ppro
	ppro.transposeNext += (ppro.transposeClicks * engine.OneBeat)
	ppro.transposeIndex = (ppro.transposeIndex + 1) % len(ppro.transposeValues)

			transposePitch := ppro.transposeValues[ppro.transposeIndex]
				fr _, layer := range TheRouter().layers {
					// layer.clearDown()
					LogOfType("transpose""setting transposepitch in layer","pad", layer.padName, "transposePitch",transposePitch, "nactive",len(layer.activeNotes))
					layer.TransposePitch = transposePitch
				}
		sched.SendAllPendingNoteoffs()
}
*/

func BoundAndScaleController(v, vmin, vmax float32, cmin, cmax int) int {
	newv := BoundAndScaleFloat(v, vmin, vmax, float32(cmin), float32(cmax))
	return int(newv)
}

func BoundAndScaleFloat(v, vmin, vmax, outmin, outmax float32) float32 {
	if v < vmin {
		v = vmin
	} else if v > vmax {
		v = vmax
	}
	out := outmin + (outmax-outmin)*((v-vmin)/(vmax-vmin))
	return out
}
