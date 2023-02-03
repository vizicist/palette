package plugin

import (
	"math"
	"sync"

	"github.com/vizicist/palette/engine"
)

type PatchLogic struct {
	quadpro *QuadPro
	patch   *engine.Patch

	mutex sync.Mutex
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

func NewPatchLogic(quadpro *QuadPro, patch *engine.Patch) *PatchLogic {
	logic := &PatchLogic{
		quadpro: quadpro,
		patch:   patch,
	}
	return logic
}

func (logic *PatchLogic) cursorToNoteOn(ctx *engine.PluginContext, ce engine.CursorEvent) *engine.NoteOn {
	pitch := logic.cursorToPitch(ctx, ce)
	velocity := logic.cursorToVelocity(ctx, ce)
	synth := logic.patch.Synth()
	return engine.NewNoteOn(synth, pitch, velocity)
}

func (logic *PatchLogic) cursorToPitch(ctx *engine.PluginContext, ce engine.CursorEvent) uint8 {
	patch := logic.patch
	quadpro := logic.quadpro
	pitchmin := patch.GetInt("sound.pitchmin")
	pitchmax := patch.GetInt("sound.pitchmax")
	dp := pitchmax - pitchmin + 1
	p1 := int(ce.X * float32(dp))
	p := uint8(pitchmin + p1%dp)

	usescale := patch.GetBool("misc.usescale")
	chromatic := patch.GetBool("sound.chromatic")
	if !chromatic && usescale {
		scaleName := patch.GetString("misc.scale")
		scale := engine.GetScale(scaleName)
		p = scale.ClosestTo(p)
		// MIDIOctaveShift might be negative
		i := int(p) + 12*quadpro.MIDIOctaveShift
		for i < 0 {
			i += 12
		}
		for i > 127 {
			i -= 12
		}
		p = uint8(i + quadpro.TransposePitch)
	}
	return p
}

func (logic *PatchLogic) cursorToVelocity(ctx *engine.PluginContext, ce engine.CursorEvent) uint8 {
	patch := logic.patch
	volstyle := patch.Get("misc.volstyle")
	if volstyle == "" {
		volstyle = "pressure"
	}
	velocitymin := patch.GetInt("sound.velocitymin")
	velocitymax := patch.GetInt("sound.velocitymax")
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
	switch volstyle {
	case "frets":
		v = 1.0 - ce.Y
	case "pressure":
		v = ce.Z * 4.0
	case "fixed":
		// do nothing
	default:
		engine.LogWarn("Unrecognized vol value", "volstyle", volstyle)
	}
	dv := velocitymax - velocitymin + 1
	p1 := int(v * float32(dv))
	vel := uint8(velocitymin + p1%dv)
	return uint8(vel)
}

func (logic *PatchLogic) generateVisualsFromCursor(ce engine.CursorEvent) {
	// send an OSC message to Resolume
	msg := engine.CursorToOscMsg(ce)
	engine.TheResolume().ToFreeFramePlugin(logic.patch.Name(), msg)
}

func (logic *PatchLogic) generateSoundFromCursor(ctx *engine.PluginContext, ce engine.CursorEvent, cursorStyle string) {

	switch cursorStyle {
	case "downonly":
		logic.generateSoundFromCursorDownOnly(ctx, ce)
	case "", "retrigger":
		logic.generateSoundFromCursorRetrigger(ctx, ce)
	default:
		engine.LogWarn("Unrecognized cursorStyle", "cursorStyle", cursorStyle)
		logic.generateSoundFromCursorDownOnly(ctx, ce)
	}
}

func (logic *PatchLogic) generateSoundFromCursorDownOnly(ctx *engine.PluginContext, ce engine.CursorEvent) {

	logic.mutex.Lock()
	defer logic.mutex.Unlock()

	switch ce.Ddu {
	case "down":
		noteOn := logic.cursorToNoteOn(ctx, ce)
		atClick := logic.nextQuant(ctx.CurrentClick(), logic.patch.CursorToQuant(ce))
		// engine.LogInfo("logic.down", "current", ctx.CurrentClick(), "atClick", atClick, "noteOn", noteOn)
		ctx.ScheduleAt(noteOn, atClick)
		noteOff := engine.NewNoteOffFromNoteOn(noteOn)
		atClick += engine.QuarterNote
		ctx.ScheduleAt(noteOff, atClick)

	case "drag":
		// do nothing

	case "up":
		// do nothing
	}
}

func (logic *PatchLogic) generateSoundFromCursorRetrigger(ctx *engine.PluginContext, ce engine.CursorEvent) {

	logic.mutex.Lock()
	defer logic.mutex.Unlock()

	patch := logic.patch
	cursorState := ctx.GetCursorState(ce.Cid)
	if cursorState == nil {
		engine.LogWarn("generateSoundFromCursor: cursorState is nil", "cid", ce.Cid)
		return
	}

	switch ce.Ddu {
	case "down":
		// engine.LogInfo("CURSOR down event for cursor", "cid", ce.Cid)
		oldNoteOn := cursorState.NoteOn
		if oldNoteOn != nil {
			engine.LogWarn("generateSoundFromCursor: oldNote already exists", "cid", ce.Cid)
			noteOff := engine.NewNoteOffFromNoteOn(oldNoteOn)
			ctx.ScheduleAt(noteOff, ctx.CurrentClick())
		}
		atClick := logic.nextQuant(ctx.CurrentClick(), logic.patch.CursorToQuant(ce))
		noteOn := logic.cursorToNoteOn(ctx, ce)
		ctx.ScheduleAt(noteOn, atClick)
		cursorState.NoteOn = noteOn
		cursorState.NoteOnClick = atClick
	case "drag":
		// engine.LogInfo("CURSOR drag event for cursor", "cid", ce.Cid)
		oldNoteOn := cursorState.NoteOn
		if oldNoteOn == nil {
			engine.LogWarn("generateSoundFromCursor: no cursorState.NoteOn", "cid", ce.Cid)
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
		deltaztrig := patch.GetFloat("sound._deltaztrig")

		deltay := float32(math.Abs(float64(cursorState.Previous.Y - ce.Y)))
		deltaytrig := patch.GetFloat("sound._deltaytrig")

		logic.generateController(ctx, cursorState)

		if newpitch != oldpitch || deltaz > deltaztrig || deltay > deltaytrig {
			// Turn off existing note, one Click after noteOn
			noteOff := engine.NewNoteOffFromNoteOn(oldNoteOn)
			offClick := cursorState.NoteOnClick + 1
			ctx.ScheduleAt(noteOff, offClick)

			atClick := logic.nextQuant(ctx.CurrentClick(), logic.patch.CursorToQuant(ce))
			if atClick < offClick {
				atClick = offClick
			}
			ctx.ScheduleAt(newNoteOn, atClick)
			cursorState.NoteOn = newNoteOn
			cursorState.NoteOnClick = atClick
		}

	case "up":
		// engine.LogInfo("CURSOR up event for cursor", "cid", ce.Cid)
		oldNoteOn := cursorState.NoteOn
		if oldNoteOn == nil {
			// not sure why this happens, yet
			engine.LogWarn("Unexpected UP, no oldNoteOn", "cid", ce.Cid)
		} else {
			noteOff := engine.NewNoteOffFromNoteOn(oldNoteOn)
			offClick := cursorState.NoteOnClick + 1
			ctx.ScheduleAt(noteOff, offClick+1)
			// delete(logic.cursorNote, ce.Cid)
		}
	}
}

func (logic *PatchLogic) generateController(ctx *engine.PluginContext, cursorState *engine.CursorState) {

	if logic.patch.Get("sound.controllerstyle") == "modulationonly" {
		zmin := logic.patch.GetFloat("sound._controllerzmin")
		zmax := logic.patch.GetFloat("sound._controllerzmax")
		cmin := logic.patch.GetInt("sound._controllermin")
		cmax := logic.patch.GetInt("sound._controllermax")
		oldz := cursorState.Previous.Z
		newz := cursorState.Current.Z
		// XXX - should put the old controller value in ActiveNote so
		// it doesn't need to be computed every time
		oldzc := BoundAndScaleController(oldz, zmin, zmax, cmin, cmax)
		newzc := BoundAndScaleController(newz, zmin, zmax, cmin, cmax)

		if newzc != 0 && newzc != oldzc {
			logic.patch.Synth().SendController(1, newzc)
		}
	}
}

func (logic *PatchLogic) nextQuant(t engine.Clicks, q engine.Clicks) engine.Clicks {
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

/*
func (logic *PatchLogic) oldsendNoteOn(note any) {

	logic.patch.Synth.SendNoteToMidiOutput(note)

	// ss := logic.patch.Get("visual.spritesource")
	// if ss == "midi" {
	// 	logic.generateSpriteFromPhraseElement(ctx, pe)
	// }
}
*/

/*
func (logic *PatchLogic) SendPhraseElementToSynth(ctx *engine.PluginContext, pe *engine.PhraseElement) {

	ss := logic.patch.Get("visual.spritesource")
	if ss == "midi" {
		logic.generateSpriteFromPhraseElement(ctx, pe)
	}
	logic.patch.Synth.SendTo(pe)
}
*/

/*
func (logic *PatchLogic) generateSpriteFromPhraseElement(ctx *engine.PluginContext, pe *engine.PhraseElement) {

	patch := logic.patch

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

	pitchmin := uint8(patch.GetInt("sound.pitchmin"))
	pitchmax := uint8(patch.GetInt("sound.pitchmax"))
	if pitch < pitchmin || pitch > pitchmax {
		engine.LogWarn("Unexpected value", "pitch", pitch)
		return
	}

	var x float32
	var y float32
	switch patch.Get("visual.placement") {
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

	engine.TheResolume().ToFreeFramePlugin(patch.Name(), msg)
}
*/

// func (logic *PatchLogic) sendANO() {
// 	logic.patch.Synth.SendANO()
// }

/*
func (logic *PatchLogic) sendNoteOff(n *engine.NoteOn) {
	if n == nil {
		// Not sure why this sometimes happens
		return
	}
	noteOff := engine.NewNoteOff(n.Channel, n.Pitch, n.Velocity)
	// pe := &engine.PhraseElement{Value: noteOff}
	logic.patch.Synth.SendNoteToMidiOutput(noteOff)
	// patch.SendPhraseElementToSynth(pe)
}
*/

/*
func (logic *PatchLogic) advanceTransposeTo(newclick engine.Clicks) {

	quadpro := logic.quadpro
	quadpro.transposeNext += (quadpro.transposeClicks * engine.OneBeat)
	quadpro.transposeIndex = (quadpro.transposeIndex + 1) % len(quadpro.transposeValues)

			transposePitch := quadpro.transposeValues[quadpro.transposeIndex]
				fr _, patch := range TheRouter().patches {
					// patch.clearDown()
					LogOfType("transpose""setting transposepitch in patch","pad", patch.padName, "transposePitch",transposePitch, "nactive",len(patch.activeNotes))
					patch.TransposePitch = transposePitch
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
