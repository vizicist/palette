package kit

import (
	"fmt"
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
	synth := logic.patch.Synth()
	velocity := logic.cursorToVelocity(ce)
	pitch, err := logic.cursorToPitch(ce)

	LogInfo("cursorToNoteOn","velocity",velocity)

	if err != nil {
		LogIfError(fmt.Errorf("cursorToNoteOn: no pitch for cursor, ce=%v", ce))
		return nil
	}
	LogOfType("cursor", "cursorToNoteOn", "pitch", pitch, "velocity", velocity)
	return NewNoteOn(synth, pitch, velocity)
}

var PitchSets = map[string][]uint8{
	"stylusrmx": {36, 37, 38, 39, 42, 43, 51, 49},
}

func (logic *PatchLogic) cursorToPitch(ce CursorEvent) (uint8, error) {
	patch := logic.patch
	pitchset := patch.Get("sound.pitchset")
	if pitchset != "" {
		pitches, ok := PitchSets[pitchset]
		if !ok {
			err := fmt.Errorf("unknown value for sound.pitchset: %s", pitchset)
			LogIfError(err)
			return 0, err
		}
		// In the stylusrmx set, there are 8 pitches corresponding to the 8 parts in Stylus RMX
		n := int(ce.Pos.X*8.0) % len(pitches)
		// Note: pitchsets don't get pitchoffset
		return pitches[n], nil

	} else {
		pitchmin := patch.GetInt("sound.pitchmin")
		pitchmax := patch.GetInt("sound.pitchmax")
		if pitchmin > pitchmax {
			t := pitchmin
			pitchmin = pitchmax
			pitchmax = t
		}
		dp := pitchmax - pitchmin + 1
		p1 := int(ce.Pos.X * float32(dp))
		p := uint8(pitchmin + p1%dp)

		scaleName := patch.Get("misc.scale")

		// The engine.scale param, if not "", overrides misc.scale
		engineScaleName, err := GetParam("engine.scale")
		if err != nil {
			engineScaleName = ""
		}
		if engineScaleName != "" {
			scaleName = engineScaleName
		}

		if scaleName != "chromatic" {
			scale := GetScale(scaleName)
			closest := scale.ClosestTo(p)
			// MIDIOctaveShift might be negative
			i := int(closest) + 12*MidiOctaveShift
			for i < 0 {
				i += 12
			}
			for i > 127 {
				i -= 12
			}
			p = uint8(i)
		}
		currentOffset := int(TheScheduler.currentPitchOffset.Load())
		if currentOffset != 0 {
			newpitch := int(p) + currentOffset
			if newpitch < 0 {
				newpitch = 0
			} else if newpitch > 127 {
				newpitch = 127
			}
			// LogOfType("midi", "cursorToPitch applied pitchoffset", "newpitch", newpitch, "oldpitch", p)
			p = uint8(newpitch)
		}
		return p, nil
	}
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
		LogWarn("Hey! velocitymin == velocitymax == 0", "patch", patch.Name(), "ce", ce)
		velocitymin = 0
		velocitymax = 127
	}
	if velocitymin > velocitymax { // really?
		LogWarn("Hey! velocitymin > velocitymax", "velocitymin", velocitymin, "velocitymax", velocitymax, "patch", patch.Name(), "ce", ce)
		t := velocitymin
		velocitymin = velocitymax
		velocitymax = t
	}
	zmin := patch.GetFloat("sound._controllerzmin")
	zmax := patch.GetFloat("sound._controllerzmax")

	scaledZ := BoundAndScaleFloat(ce.Pos.Z, zmin, zmax, 0.0, 1.0)
	// LogInfo("CursorToVelocity","scaledZ",scaledZ,"ce.Z",ce.Z,"zmin",zmin,"zmax",zmax)

	// Scale cursor Z to controller zmin and zmax
	v := float32(0.8) // default and fixed value
	switch volstyle {
	case "frets":
		v = 1.0 - ce.Pos.Y
	case "pressure":
		v = scaledZ
	case "fixed":
		// do nothing
	default:
		LogWarn("Unrecognized vol value", "volstyle", volstyle)
	}

	dv := velocitymax - velocitymin + 1
	vScaledToVelocityRange := int(v * float32(dv))
	var vel uint8
	if vScaledToVelocityRange >= dv {
		vel = uint8(velocitymax)
	} else {
		vel = uint8(velocitymin) + uint8(vScaledToVelocityRange%dv)
	}

	LogInfo("cursorToVelocity","velocitymin",velocitymin,"p1",vScaledToVelocityRange,"dv",dv,"v",v)
	return uint8(vel)
}

// func (logic *PatchLogic) generateVisualsFromCursor(ce CursorEvent) {
// 	// send an OSC message to Resolume
// 	msg := CursorToOscMsg(ce)
// 	TheResolume().ToFreeFramePlugin(logic.patch.Name(), msg)
// }

func (logic *PatchLogic) generateSoundFromCursor(ce CursorEvent, cursorStyle string) {

	LogOfType("gensound", "generateSoundFromCursor", "cursor", ce.Gid, "ce", ce)

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

	// XXX - is this mutex really needed?
	logic.mutex.Lock()
	defer logic.mutex.Unlock()

	switch ce.Ddu {
	case "down":
		noteOn := logic.cursorToNoteOn(ce)
		if noteOn == nil {
			return // do nothing, assumes any errors are logged in cursorToNoteOn
		}
		atClick := logic.nextQuant(CurrentClick(), logic.patch.CursorToQuant(ce))
		// LogInfo("logic.down", "current", CurrentClick(), "atClick", atClick, "noteOn", noteOn)
		ScheduleAt(atClick, ce.Tag, noteOn)
		noteOff := NewNoteOffFromNoteOn(noteOn)
		atClick += QuarterNote
		ScheduleAt(atClick, ce.Tag, noteOff)

	case "drag":
		// do nothing

	case "up":
		// do nothing
	}
}

func (logic *PatchLogic) generateSoundFromCursorRetrigger(ce CursorEvent) {

	// XXX - is this mutex really needed?
	logic.mutex.Lock()
	defer logic.mutex.Unlock()

	patch := logic.patch
	ac, ok := TheCursorManager.getActiveCursorFor(ce.Gid)
	if !ok {
		LogWarn("generateSoundFromCursor: no active cursor", "gid", ce.Gid)
		return
	}

	switch ce.Ddu {
	case "down":
		oldNoteOn := ac.NoteOn
		if oldNoteOn != nil {
			// I don't recall the situations where this occurred,
			// but it does happen pretty regularly, I think.
			// LogWarn("generateSoundFromCursor: oldNote already exists", "gid", ce.Gid)
			noteOff := NewNoteOffFromNoteOn(oldNoteOn)
			ScheduleAt(CurrentClick(), ce.Tag, noteOff)
		}
		atClick := logic.nextQuant(CurrentClick(), patch.CursorToQuant(ce))
		noteOn := logic.cursorToNoteOn(ce)
		if noteOn == nil {
			LogWarn("Hmmm, retrigger, noteOn for down is nil?")
			return // do nothing, assumes any errors are logged in cursorToNoteOn
		}
		ScheduleAt(atClick, ce.Tag, noteOn)
		ac.NoteOn = noteOn
		ac.NoteOnY = ce.Pos.Y
		ac.NoteOnClick = atClick
	case "drag":
		oldNoteOn := ac.NoteOn
		if oldNoteOn == nil {
			// LogWarn("generateSoundFromCursor: no ActiveCursor.NoteOn", "gid", ce.Gid)
			return
		}
		newNoteOn := logic.cursorToNoteOn(ce)
		if newNoteOn == nil {
			LogWarn("Hmmm, retrigger, noteOn for drag is nil?")
			return // do nothing, assumes any errors are logged in cursorToNoteOn
		}
		oldpitch := oldNoteOn.Pitch
		newpitch := newNoteOn.Pitch
		oldvelocity := oldNoteOn.Velocity
		newvelocity := newNoteOn.Velocity
		// We only turn off the existing note (for a given Cursor ID)
		// and start the new one if the pitch changes

		// Also do this if the Z/Velocity value changes more than the trigger value

		// NOTE: this could and perhaps should use a.ce.Z now that we're
		// saving a.ce, like the deltay value

		dz := float64(int(oldvelocity) - int(newvelocity))
		deltaz := float32(math.Abs(dz) / 128.0)
		deltaztrignote := patch.GetFloat("sound._deltaztrignote")
		deltaztrigcontroller := patch.GetFloat("sound._deltaztrigcontroller")

		// deltay := float32(math.Abs(float64(ac.Previous.Pos.Y - ce.Pos.Y)))
		deltay := float32(math.Abs(float64(ac.NoteOnY - ce.Pos.Y)))
		deltaytrignote := patch.GetFloat("sound._deltaytrignote")
		// deltaytrigcontroller := patch.GetFloat("sound._deltaytrigcontroller")

		// logic.generateController(ac)
		if patch.Get("sound.controllerstyle") == "modulationonly" {
			if deltaz > deltaztrigcontroller {
				patch.Synth().SendController(1, newvelocity)
			}
			// Currently, only z sends the modulation controller.
			// There needs to be a new parameter to give a specific controller number for Y.
			// if deltay > deltaytrigcontroller {
			//	// LogWarn("Hey, deltaytrigcontroller needs work?")
			//	// patch.Synth().SendController(1, newvelocity)
			// }
		}

		cc := CurrentClick()
		c2q := patch.CursorToQuant(ce)
		// If the last NoteOn for this ActiveCursor is scheduled in the future, don't retrigger.
		if ac.NoteOnClick > cc {
			// inTheFuture := ac.NoteOnClick - cc
			// LogInfo("NOTEON IN FUTURE, NOT RETRIGGERING!!!!!", "inTheFuture", inTheFuture)
			return
		}

		deltay_triggered := false
		deltaz_triggered := false
		newpitch_triggered := false
		// LogInfo("deltay and deltaytrig", "deltay", deltay, "deltaytrignote", deltaytrignote)
		if deltay > deltaytrignote {
			deltay_triggered = true
			// LogInfo("YES!!  deltay > deltaytrig", "deltay", deltay, "deltaytrignote", deltaytrignote)
		}
		if deltaz > deltaztrignote {
			deltaz_triggered = true
			// LogInfo("YES!!  deltaz > deltaztrig", "deltaz", deltaz, "deltaztrig", deltaztrignote)
		}
		if newpitch != oldpitch {
			newpitch_triggered = true
		}

		doNewNote := newpitch_triggered || deltaz_triggered || deltay_triggered
		if doNewNote {

			LogOfType("note", "Turning note off/on due",
				"newpitch_triggered",newpitch_triggered,
				"deltaz_triggered",deltaz_triggered,
				"deltay_triggered",deltay_triggered)

			// Turn off existing note, one Click after noteOn
			noteOff := NewNoteOffFromNoteOn(oldNoteOn)
			offClick := ac.NoteOnClick + 1
			ScheduleAt(offClick, ce.Tag, noteOff)

			thisClick := logic.nextQuant(cc, c2q)
			if thisClick < offClick {
				thisClick = offClick
			}

			ScheduleAt(thisClick, ce.Tag, newNoteOn)
			ac.NoteOn = newNoteOn
			ac.NoteOnClick = thisClick
			if deltay_triggered {
				ac.NoteOnY = ce.Pos.Y
				// LogInfo("Setting ac.NoteOnY","ac.NoteOnY",ac.NoteOnY)
			}
			if deltaz_triggered {
				ac.NoteOnZ = ce.Pos.Z
			}
		}

	case "up":
		// LogInfo("CURSOR up event for cursor", "gid", ce.Gid)
		oldNoteOn := ac.NoteOn
		if oldNoteOn == nil {
			// not sure why this happens, yet
			LogWarn("Unexpected UP, no oldNoteOn", "gid", ce.Gid)
		} else {
			noteOff := NewNoteOffFromNoteOn(oldNoteOn)
			offClick := ac.NoteOnClick + 1
			ScheduleAt(offClick+1, ce.Tag, noteOff)
			// delete(logic.cursorNote, ce.Gid)
		}
	}
}

/*
func (logic *PatchLogic) generateController(ActiveCursor *ActiveCursor) {

	patch := logic.patch
	if patch.Get("sound.controllerstyle") == "modulationonly" {
		zmin := patch.GetFloat("sound._controllerzmin")
		zmax := patch.GetFloat("sound._controllerzmax")
		cmin := patch.GetInt("sound._controllermin")
		cmax := patch.GetInt("sound._controllermax")
		oldz := ActiveCursor.Previous.Z
		newz := ActiveCursor.Current.Z
		// XXX - should put the old controller value in ActiveNote so
		// it doesn't need to be computed every time
		oldzc := BoundAndScaleController(oldz, zmin, zmax, cmin, cmax)
		newzc := BoundAndScaleController(newz, zmin, zmax, cmin, cmax)

		if newzc != 0 && newzc != oldzc {
			patch.Synth().SendController(1, newzc)
		}
	}
}
*/

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
		x = TheRand.Float32()
		y = TheRand.Float32()
	case "linear":
		y = 0.5
		x = float32(pitch-pitchmin) / float32(pitchmax-pitchmin)
	case "cursor":
		x = TheRand.Float32()
		y = TheRand.Float32()
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
		x = TheRand.Float32()
		y = TheRand.Float32()
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
