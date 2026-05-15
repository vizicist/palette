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
	if err != nil {
		LogError(fmt.Errorf("cursorToNoteOn: no pitch for cursor"), "ce", ce)
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
		if pitchmin > pitchmax { // really?
			LogWarn("Hey! pitchmin > pitchmax", "pitchmin", pitchmin, "pitchmax", pitchmax, "patch", patch.Name(), "ce", ce)
			t := pitchmin
			pitchmin = pitchmax
			pitchmax = t
		}
		dp := pitchmax - pitchmin + 1
		p1 := int(ce.Pos.X * float64(dp))
		p := uint8(pitchmin + p1%dp)

		scaleName := patch.Get("misc.scale")

		// The global.scale param, if not "", overrides misc.scale
		engineScaleName, err := GetParam("global.scale")
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
			i := int(closest) + 12*theRouter.midiOctaveShift
			for i < 0 {
				i += 12
			}
			for i > 127 {
				i -= 12
			}
			p = uint8(i)
		}
		currentOffset := int(theEngine.currentPitchOffset.Load())
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
	v := 0.8 // default and fixed value
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
	p1 := int(v * float64(dv))
	vel := uint8(velocitymin + p1%dv)
	return uint8(vel)
}

func samplesplitterVelocityFromPressure(patch *Patch, pressure float64) uint8 {
	const minTransmissionVelocity = 76

	scaledPressure := boundValueZeroToOne(pressure)
	if patch != nil {
		zmin := patch.GetFloat("sound._controllerzmin")
		zmax := patch.GetFloat("sound._controllerzmax")
		scaledPressure = BoundAndScaleFloat(pressure, zmin, zmax, 0.0, 1.0)
	}
	velocity := minTransmissionVelocity + int(math.Round(scaledPressure*float64(127-minTransmissionVelocity)))
	if velocity > 127 {
		velocity = 127
	}
	return uint8(velocity)
}

func (logic *PatchLogic) generateVisualsFromCursor(ce CursorEvent) {
	// send an OSC message to Resolume
	msg := CursorToOscMsg(ce)
	TheResolume().ToFreeFramePlugin(logic.patch.Name(), msg)
}

func (logic *PatchLogic) liveStepperRoute() string {
	if theStepper == nil {
		return "bidule"
	}
	return theStepper.routeForPatch(logic.patch.Name())
}

func (logic *PatchLogic) bss2SampleMode() bool {
	return IsBSS2InitialPage() && logic.liveStepperRoute() == "samplesplitter"
}

func (logic *PatchLogic) liveRouteIncludesBidule() bool {
	route := logic.liveStepperRoute()
	return route == "bidule" || route == "both"
}

func (logic *PatchLogic) liveRouteIncludesSamples() bool {
	route := logic.liveStepperRoute()
	return route == "samplesplitter" || route == "both"
}

func (logic *PatchLogic) scheduleLiveNoteOn(ce CursorEvent, noteOn *NoteOn, atClick Clicks) {
	if logic.liveRouteIncludesBidule() {
		ScheduleAt(atClick, ce.Tag, noteOn)
	}
	if logic.liveRouteIncludesSamples() && theStepper != nil {
		synth := theStepper.samplesplitterSynthForPatch(logic.patch.Name())
		if synth == nil {
			return
		}
		velocity := samplesplitterVelocityFromPressure(logic.patch, ce.Pos.Z)
		ScheduleAt(atClick, ce.Tag, NewPitchBend(synth, theStepper.pitchBendValue(ce.Pos.Z)))
		ScheduleAt(atClick, ce.Tag, NewNoteOn(synth, noteOn.Pitch, velocity))
		ScheduleAt(atClick+1, ce.Tag, NewPitchBend(synth, MidiPitchBendCenter))
	}
}

func (logic *PatchLogic) bss2SamplePlaybackStart(ce CursorEvent) *SamplePlaybackStart {
	x := boundValueZeroToOne(ce.Pos.X)
	sampleSelector := int(math.Min(47, math.Floor(x*48.0)))
	velocity := samplesplitterVelocityFromPressure(logic.patch, ce.Pos.Z)
	return &SamplePlaybackStart{
		Patch:          logic.patch.Name(),
		SigilChannel:   SamplePlaybackChannelForPatch(logic.patch.Name()),
		SampleSelector: sampleSelector,
		Velocity:       int(velocity),
		PitchBend:      logic.bss2SamplePitchBendValue(ce),
		VoiceKey:       bss2SampleVoiceKey(ce),
	}
}

func bss2SampleVoiceKey(ce CursorEvent) string {
	return fmt.Sprintf("cursor-%s-%d", ce.Tag, ce.GID)
}

func (logic *PatchLogic) bss2SamplePitchBendValue(ce CursorEvent) int {
	p := boundValueZeroToOne(ce.Pos.Y)
	return int(math.Round(p * 16383.0))
}

func (logic *PatchLogic) bss2SampleQuant() Clicks {
	quantBeats, err := GetParamFloat("global.transmissionquant")
	if err != nil {
		LogIfError(err)
		quantBeats = 0.5
	}
	if quantBeats <= 0 {
		return 1
	}
	if quantBeats > 1 {
		quantBeats = 1
	}
	factor := TempoFactor
	if factor <= 0 {
		factor = 1
	}
	return Clicks(math.Max(1, float64(OneBeat)*quantBeats/factor))
}

func (logic *PatchLogic) nextBSS2SampleQuant(t Clicks) Clicks {
	return logic.nextQuant(t, logic.bss2SampleQuant())
}

func (logic *PatchLogic) scheduleBSS2SampleStart(ac *ActiveCursor, ce CursorEvent, noteOn *SamplePlaybackStart, atClick Clicks) {
	if noteOn == nil {
		return
	}
	if ac.SamplePlayback != nil {
		ScheduleAt(atClick, ce.Tag, &SamplePlaybackStop{Patch: ac.SamplePlayback.Patch, SigilChannel: ac.SamplePlayback.SigilChannel, SampleSelector: ac.SamplePlayback.SampleSelector, VoiceKey: ac.SamplePlayback.VoiceKey})
	}
	ScheduleAt(atClick, ce.Tag, noteOn)
	ac.SamplePlayback = noteOn
}

func (logic *PatchLogic) scheduleBSS2SampleStop(ac *ActiveCursor, ce CursorEvent, atClick Clicks) {
	if ac.SamplePlayback != nil {
		ScheduleAt(atClick, ce.Tag, &SamplePlaybackStop{Patch: ac.SamplePlayback.Patch, SigilChannel: ac.SamplePlayback.SigilChannel, SampleSelector: ac.SamplePlayback.SampleSelector, VoiceKey: ac.SamplePlayback.VoiceKey})
		ScheduleAt(atClick+1, ce.Tag, &SamplePlaybackPitch{Patch: ac.SamplePlayback.Patch, SigilChannel: ac.SamplePlayback.SigilChannel, Value: MidiPitchBendCenter})
		ac.SamplePlayback = nil
	}
}

func (logic *PatchLogic) scheduleLiveNoteOff(ce CursorEvent, noteOff *NoteOff, atClick Clicks) {
	if logic.liveRouteIncludesBidule() {
		ScheduleAt(atClick, ce.Tag, noteOff)
	}
	if logic.liveRouteIncludesSamples() && theStepper != nil {
		synth := theStepper.samplesplitterSynthForPatch(logic.patch.Name())
		if synth == nil {
			return
		}
		ScheduleAt(atClick, ce.Tag, NewNoteOff(synth, noteOff.Pitch, noteOff.Velocity))
	}
}

func (logic *PatchLogic) generateSoundFromCursor(ce CursorEvent, cursorStyle string) {

	LogOfType("gensound", "generateSoundFromCursor", "cursor", ce.GID, "ce", ce)

	if logic.bss2SampleMode() {
		logic.generateBSS2SampleFromCursor(ce)
		return
	}

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

func (logic *PatchLogic) generateBSS2SampleFromCursor(ce CursorEvent) {
	logic.mutex.Lock()
	defer logic.mutex.Unlock()

	ac, ok := theCursorManager.getActiveCursorFor(ce.GID)
	if !ok {
		LogWarn("generateBSS2SampleFromCursor: no active cursor", "gid", ce.GID)
		return
	}

	switch ce.Ddu {
	case "down":
		noteOn := logic.bss2SamplePlaybackStart(ce)
		if noteOn == nil {
			return
		}
		atClick := logic.nextBSS2SampleQuant(CurrentClick())
		logic.scheduleBSS2SampleStart(ac, ce, noteOn, atClick)
		ac.NoteOnClick = atClick

	case "drag":
		oldNoteOn := ac.SamplePlayback
		if oldNoteOn == nil {
			noteOn := logic.bss2SamplePlaybackStart(ce)
			if noteOn == nil {
				return
			}
			atClick := logic.nextBSS2SampleQuant(CurrentClick())
			logic.scheduleBSS2SampleStart(ac, ce, noteOn, atClick)
			ac.NoteOnClick = atClick
			return
		}
		newNoteOn := logic.bss2SamplePlaybackStart(ce)
		if newNoteOn == nil {
			return
		}
		if newNoteOn.SampleSelector != oldNoteOn.SampleSelector {
			now := CurrentClick()
			onClick := logic.nextBSS2SampleQuant(now)
			theScheduler.DeleteSamplePlaybackStarts(ce.Tag, oldNoteOn.SigilChannel)
			logic.scheduleBSS2SampleStart(ac, ce, newNoteOn, onClick)
			ac.NoteOnClick = onClick
		} else {
			ScheduleAt(CurrentClick(), ce.Tag, &SamplePlaybackPitch{Patch: oldNoteOn.Patch, SigilChannel: oldNoteOn.SigilChannel, Value: logic.bss2SamplePitchBendValue(ce)})
		}

	case "up":
		if ac.SamplePlayback != nil {
			LogInfo("BSS2 sample cursor up", "patch", ac.SamplePlayback.Patch, "sigilChannel", ac.SamplePlayback.SigilChannel, "sampleSelector", ac.SamplePlayback.SampleSelector, "click", CurrentClick())
			theScheduler.DeleteSamplePlaybackStarts(ce.Tag, ac.SamplePlayback.SigilChannel)
			ScheduleAt(CurrentClick(), ce.Tag, &SamplePlaybackStop{Patch: ac.SamplePlayback.Patch, SigilChannel: ac.SamplePlayback.SigilChannel, SampleSelector: ac.SamplePlayback.SampleSelector, VoiceKey: ac.SamplePlayback.VoiceKey})
			ScheduleAt(CurrentClick()+1, ce.Tag, &SamplePlaybackPitch{Patch: ac.SamplePlayback.Patch, SigilChannel: ac.SamplePlayback.SigilChannel, Value: MidiPitchBendCenter})
			ac.SamplePlayback = nil
		} else {
			sigilChannel := SamplePlaybackChannelForPatch(logic.patch.Name())
			LogInfo("BSS2 sample cursor up without active sample playback", "patch", logic.patch.Name(), "sigilChannel", sigilChannel, "click", CurrentClick())
			theScheduler.DeleteSamplePlaybackStarts(ce.Tag, sigilChannel)
			ScheduleAt(CurrentClick(), ce.Tag, &SamplePlaybackStop{Patch: logic.patch.Name(), SigilChannel: sigilChannel, SampleSelector: -1})
			ScheduleAt(CurrentClick()+1, ce.Tag, &SamplePlaybackPitch{Patch: logic.patch.Name(), SigilChannel: sigilChannel, Value: MidiPitchBendCenter})
		}
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
		quant := logic.patch.CursorToQuant(ce)
		atClick := logic.nextQuant(CurrentClick(), quant)
		// LogInfo("logic.down", "current", CurrentClick(), "atClick", atClick, "noteOn", noteOn)
		logic.scheduleLiveNoteOn(ce, noteOn, atClick)
		if theStepper != nil && !IsBSS2InitialPage() {
			theStepper.RecordNoteOn(ce.Tag, noteOn, ce.Pos.Z, atClick, quant)
		}
		noteOff := NewNoteOffFromNoteOn(noteOn)
		atClick += QuarterNote
		logic.scheduleLiveNoteOff(ce, noteOff, atClick)
		if theStepper != nil && !IsBSS2InitialPage() {
			theStepper.RecordNoteOff(ce.Tag, noteOff, atClick)
		}

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
	ac, ok := theCursorManager.getActiveCursorFor(ce.GID)
	if !ok {
		LogWarn("generateSoundFromCursor: no active cursor", "gid", ce.GID)
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
			logic.scheduleLiveNoteOff(ce, noteOff, CurrentClick())
			if theStepper != nil && !IsBSS2InitialPage() {
				theStepper.RecordNoteOff(ce.Tag, noteOff, CurrentClick())
			}
		}
		quant := patch.CursorToQuant(ce)
		atClick := logic.nextQuant(CurrentClick(), quant)
		noteOn := logic.cursorToNoteOn(ce)
		if noteOn == nil {
			LogWarn("Hmmm, retrigger, noteOn for down is nil?")
			return // do nothing, assumes any errors are logged in cursorToNoteOn
		}
		logic.scheduleLiveNoteOn(ce, noteOn, atClick)
		if theStepper != nil && !IsBSS2InitialPage() {
			theStepper.RecordNoteOn(ce.Tag, noteOn, ce.Pos.Z, atClick, quant)
		}
		ac.NoteOn = noteOn
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
		deltaz := math.Abs(dz) / 128.0
		deltaztrignote := patch.GetFloat("sound._deltaztrignote")
		deltaztrigcontroller := patch.GetFloat("sound._deltaztrigcontroller")

		deltay := math.Abs(float64(ac.Previous.Pos.Y - ce.Pos.Y))
		deltaytrig := patch.GetFloat("sound._deltaytrig")

		// logic.generateController(ac)
		if patch.Get("sound.controllerstyle") == "modulationonly" {
			if deltaz > deltaztrigcontroller {
				patch.Synth().SendController(1, newvelocity)
			}
		}

		cc := CurrentClick()
		c2q := patch.CursorToQuant(ce)
		// If the last NoteOn for this ActiveCursor is scheduled in the future, don't retrigger.
		if ac.NoteOnClick > cc {
			// inTheFuture := ac.NoteOnClick - cc
			// LogInfo("NOTEON IN FUTURE, NOT RETRIGGERING!!!!!", "inTheFuture", inTheFuture)
			return
		}

		if newpitch != oldpitch || deltaz > deltaztrignote || deltay > deltaytrig {

			LogOfType("note", "Turning note off/on due to newpitch or deltaz or deltay")
			// Turn off existing note, one Click after noteOn
			noteOff := NewNoteOffFromNoteOn(oldNoteOn)
			offClick := ac.NoteOnClick + 1
			logic.scheduleLiveNoteOff(ce, noteOff, offClick)
			if theStepper != nil && !IsBSS2InitialPage() {
				theStepper.RecordNoteOff(ce.Tag, noteOff, offClick)
			}

			thisClick := logic.nextQuant(cc, c2q)
			if thisClick < offClick {
				thisClick = offClick
			}

			logic.scheduleLiveNoteOn(ce, newNoteOn, thisClick)
			if theStepper != nil && !IsBSS2InitialPage() {
				theStepper.RecordNoteOn(ce.Tag, newNoteOn, ce.Pos.Z, thisClick, c2q)
			}
			ac.NoteOn = newNoteOn
			ac.NoteOnClick = thisClick
		}

	case "up":
		// LogInfo("CURSOR up event for cursor", "gid", ce.Gid)
		oldNoteOn := ac.NoteOn
		if oldNoteOn == nil {
			// not sure why this happens, yet
			LogWarn("Unexpected UP, no oldNoteOn", "gid", ce.GID)
		} else {
			noteOff := NewNoteOffFromNoteOn(oldNoteOn)
			offClick := ac.NoteOnClick + 1
			logic.scheduleLiveNoteOff(ce, noteOff, offClick+1)
			if theStepper != nil && !IsBSS2InitialPage() {
				theStepper.RecordNoteOff(ce.Tag, noteOff, offClick+1)
			}
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

func BoundAndScaleController(v, vmin, vmax float64, cmin, cmax int) int {
	newv := BoundAndScaleFloat(v, vmin, vmax, float64(cmin), float64(cmax))
	return int(newv)
}

func BoundAndScaleFloat(v, vmin, vmax, outmin, outmax float64) float64 {
	if v < vmin {
		v = vmin
	} else if v > vmax {
		v = vmax
	}
	out := outmin + (outmax-outmin)*((v-vmin)/(vmax-vmin))
	return out
}
