package engine

import (
	"math"
	"sync"
)

type PatchLogic struct {
	patch *Patch

	mutex sync.Mutex
	// tempoFactor            float64
	// loop            *StepLoop
	// loopLength      Clicks
	// loopIsRecording bool
	// loopIsPlaying   bool
	// fadeLoop        float32

	// lastCursorStepEvent    CursorStepEvent
	// lastUnQuantizedStepNum Clicks
}

func NewPatchLogic(patch *Patch) *PatchLogic {
	logic := &PatchLogic{
		patch: patch,
	}
	return logic
}

func (logic *PatchLogic) cursorToNoteOn(ce CursorEvent) *NoteOn {
	pitch := logic.cursorToPitch(ce)
	velocity := logic.cursorToVelocity(ce)
	synth := logic.patch.Synth()
	return NewNoteOn(synth, pitch, velocity)
}

func (logic *PatchLogic) cursorToPitch(ce CursorEvent) uint8 {
	patch := logic.patch
	pitchmin := patch.GetInt("sound.pitchmin")
	pitchmax := patch.GetInt("sound.pitchmax")
	dp := pitchmax - pitchmin + 1
	p1 := int(ce.X * float32(dp))
	p := uint8(pitchmin + p1%dp)

	externalscaleOn := patch.GetBool("misc.externalscale")
	chromatic := patch.GetBool("sound.chromatic")
	if !chromatic && externalscaleOn {
		scaleName := TheEngine.params.Get("engine.scale")
		scale := GetScale(scaleName)
		p = scale.ClosestTo(p)
		// MIDIOctaveShift might be negative
		i := int(p) + 12*TheRouter.MIDIOctaveShift
		for i < 0 {
			i += 12
		}
		for i > 127 {
			i -= 12
		}
		p = uint8(i)
	}
	return p
}

func (logic *PatchLogic) cursorToVelocity(ce CursorEvent) uint8 {
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
		LogWarn("Unrecognized vol value", "volstyle", volstyle)
	}
	dv := velocitymax - velocitymin + 1
	p1 := int(v * float32(dv))
	vel := uint8(velocitymin + p1%dv)
	return uint8(vel)
}

func (logic *PatchLogic) generateVisualsFromCursor(ce CursorEvent) {
	// send an OSC message to Resolume
	msg := CursorToOscMsg(ce)
	TheResolume().ToFreeFramePlugin(logic.patch.Name(), msg)
}

func (logic *PatchLogic) generateSoundFromCursor(ce CursorEvent, cursorStyle string) {

	switch cursorStyle {
	case "downonly":
		logic.generateSoundFromCursorDownOnly(ce)
	case "", "retrigger":
		logic.generateSoundFromCursorRetrigger(ce)
	default:
		LogWarn("Unrecognized cursorStyle", "cursorStyle", cursorStyle)
		logic.generateSoundFromCursorDownOnly(ce)
	}
}

func (logic *PatchLogic) generateSoundFromCursorDownOnly(ce CursorEvent) {

	logic.mutex.Lock()
	defer logic.mutex.Unlock()

	switch ce.Ddu {
	case "down":
		noteOn := logic.cursorToNoteOn(ce)
		atClick := logic.nextQuant(CurrentClick(), logic.patch.CursorToQuant(ce))
		// LogInfo("logic.down", "current", CurrentClick(), "atClick", atClick, "noteOn", noteOn)
		ScheduleAt(noteOn, atClick)
		noteOff := NewNoteOffFromNoteOn(noteOn)
		atClick += QuarterNote
		ScheduleAt(noteOff, atClick)

	case "drag":
		// do nothing

	case "up":
		// do nothing
	}
}

func (logic *PatchLogic) generateSoundFromCursorRetrigger(ce CursorEvent) {

	logic.mutex.Lock()
	defer logic.mutex.Unlock()

	patch := logic.patch
	cursorState := GetCursorState(ce.Cid)
	if cursorState == nil {
		LogWarn("generateSoundFromCursor: cursorState is nil", "cid", ce.Cid)
		return
	}

	switch ce.Ddu {
	case "down":
		// LogInfo("CURSOR down event for cursor", "cid", ce.Cid)
		oldNoteOn := cursorState.NoteOn
		if oldNoteOn != nil {
			LogWarn("generateSoundFromCursor: oldNote already exists", "cid", ce.Cid)
			noteOff := NewNoteOffFromNoteOn(oldNoteOn)
			ScheduleAt(noteOff, CurrentClick())
		}
		atClick := logic.nextQuant(CurrentClick(), logic.patch.CursorToQuant(ce))
		noteOn := logic.cursorToNoteOn(ce)
		ScheduleAt(noteOn, atClick)
		cursorState.NoteOn = noteOn
		cursorState.NoteOnClick = atClick
	case "drag":
		// LogInfo("CURSOR drag event for cursor", "cid", ce.Cid)
		oldNoteOn := cursorState.NoteOn
		if oldNoteOn == nil {
			LogWarn("generateSoundFromCursor: no cursorState.NoteOn", "cid", ce.Cid)
			return
		}
		newNoteOn := logic.cursorToNoteOn(ce)
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

		logic.generateController(cursorState)

		if newpitch != oldpitch || deltaz > deltaztrig || deltay > deltaytrig {
			// Turn off existing note, one Click after noteOn
			noteOff := NewNoteOffFromNoteOn(oldNoteOn)
			offClick := cursorState.NoteOnClick + 1
			ScheduleAt(noteOff, offClick)

			atClick := logic.nextQuant(CurrentClick(), logic.patch.CursorToQuant(ce))
			if atClick < offClick {
				atClick = offClick
			}
			ScheduleAt(newNoteOn, atClick)
			cursorState.NoteOn = newNoteOn
			cursorState.NoteOnClick = atClick
		}

	case "up":
		// LogInfo("CURSOR up event for cursor", "cid", ce.Cid)
		oldNoteOn := cursorState.NoteOn
		if oldNoteOn == nil {
			// not sure why this happens, yet
			LogWarn("Unexpected UP, no oldNoteOn", "cid", ce.Cid)
		} else {
			noteOff := NewNoteOffFromNoteOn(oldNoteOn)
			offClick := cursorState.NoteOnClick + 1
			ScheduleAt(noteOff, offClick+1)
			// delete(logic.cursorNote, ce.Cid)
		}
	}
}

func (logic *PatchLogic) generateController(cursorState *CursorState) {

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

func (logic *PatchLogic) nextQuant(t Clicks, q Clicks) Clicks {
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
func (logic *PatchLogic) generateSpriteFromPhraseElement(pe *PhraseElement) {

	patch := logic.patch

	// var channel uint8
	var pitch uint8
	var velocity uint8

	switch v := pe.Value.(type) {
	case *NoteOn:
		// channel = v.Channel
		pitch = v.Pitch
		velocity = v.Velocity
	case *NoteOff:
		// channel = v.Channel
		pitch = v.Pitch
		velocity = v.Velocity
	case *NoteFull:
		// channel = v.Channel
		pitch = v.Pitch
		velocity = v.Velocity
	default:
		return
	}

	pitchmin := uint8(patch.GetInt("sound.pitchmin"))
	pitchmax := uint8(patch.GetInt("sound.pitchmax"))
	if pitch < pitchmin || pitch > pitchmax {
		LogWarn("Unexpected value", "pitch", pitch)
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

	TheResolume().ToFreeFramePlugin(patch.Name(), msg)
}
*/

// func (logic *PatchLogic) sendANO() {
// 	logic.patch.Synth.SendANO()
// }

/*
func (logic *PatchLogic) sendNoteOff(n *NoteOn) {
	if n == nil {
		// Not sure why this sometimes happens
		return
	}
	noteOff := NewNoteOff(n.Channel, n.Pitch, n.Velocity)
	// pe := &PhraseElement{Value: noteOff}
	logic.patch.Synth.SendNoteToMidiOutput(noteOff)
	// patch.SendPhraseElementToSynth(pe)
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
