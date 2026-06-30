package kit

import (
	"fmt"
	"math"
	"os"
	"sort"
	"sync"

	json "github.com/goccy/go-json"
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
	noteOn := NewNoteOn(synth, pitch, velocity)
	logic.addPitchSetInfo(noteOn, ce)
	return noteOn
}

var PitchSets = map[string][]uint8{}
var PitchSetPitchNames = map[string][]string{}

type pitchSetsConfig struct {
	PitchSets []pitchSetConfig `json:"pitchsets"`
}

type pitchSetConfig struct {
	Name    string        `json:"name"`
	Pitches []pitchConfig `json:"pitches"`
}

type pitchConfig struct {
	Pitch *int   `json:"pitch"`
	Name  string `json:"name,omitempty"`
}

func InitPitchSets() {
	pitchSets, pitchNames, err := LoadPitchSetsConfig()
	if err != nil {
		LogWarn("InitPitchSets: unable to load PitchSets.json", "err", err)
		PitchSets = map[string][]uint8{}
		PitchSetPitchNames = map[string][]string{}
		return
	}
	PitchSets = pitchSets
	PitchSetPitchNames = pitchNames
	LogInfo("PitchSets loaded", "len", len(PitchSets))
}

func LoadPitchSetsConfig() (map[string][]uint8, map[string][]string, error) {
	path := ConfigFilePath("PitchSets.json")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to read %s: %w", path, err)
	}

	var config pitchSetsConfig
	if err := json.Unmarshal(bytes, &config); err != nil {
		return nil, nil, fmt.Errorf("unable to parse %s: %w", path, err)
	}

	pitchSets := make(map[string][]uint8, len(config.PitchSets))
	pitchNames := make(map[string][]string, len(config.PitchSets))
	for _, pitchSet := range config.PitchSets {
		if pitchSet.Name == "" {
			return nil, nil, fmt.Errorf("PitchSets.json contains a pitch set with an empty name")
		}
		if _, exists := pitchSets[pitchSet.Name]; exists {
			return nil, nil, fmt.Errorf("PitchSets.json contains duplicate pitch set %q", pitchSet.Name)
		}
		if len(pitchSet.Pitches) == 0 {
			return nil, nil, fmt.Errorf("PitchSets.json pitch set %q has no pitches", pitchSet.Name)
		}
		pitches := make([]uint8, 0, len(pitchSet.Pitches))
		names := make([]string, 0, len(pitchSet.Pitches))
		hasNames := false
		for _, pitch := range pitchSet.Pitches {
			if pitch.Pitch == nil {
				return nil, nil, fmt.Errorf("PitchSets.json pitch set %q contains an entry without a pitch", pitchSet.Name)
			}
			pitchValue := *pitch.Pitch
			if pitchValue < 0 || pitchValue > 127 {
				return nil, nil, fmt.Errorf("PitchSets.json pitch set %q has out-of-range pitch %d", pitchSet.Name, pitchValue)
			}
			pitches = append(pitches, uint8(pitchValue))
			name := pitch.Name
			if name != "" {
				hasNames = true
			}
			names = append(names, name)
		}
		pitchSets[pitchSet.Name] = pitches
		if hasNames {
			pitchNames[pitchSet.Name] = names
		}
	}
	return pitchSets, pitchNames, nil
}

func PitchSetNamesFromConfig() ([]string, error) {
	pitchSets, _, err := LoadPitchSetsConfig()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(pitchSets)+1)
	names = append(names, "")
	for name := range pitchSets {
		names = append(names, name)
	}
	sort.Strings(names[1:])
	return names, nil
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
		n := pitchSetIndexForX(ce.Pos.X, len(pitches))
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

func (logic *PatchLogic) addPitchSetInfo(noteOn *NoteOn, ce CursorEvent) {
	pitchSet := logic.patch.Get("sound.pitchset")
	if pitchSet == "" {
		return
	}
	pitches, ok := PitchSets[pitchSet]
	if !ok || len(pitches) == 0 {
		return
	}
	index := pitchSetIndexForX(ce.Pos.X, len(pitches))
	pitchNames := PitchSetPitchNames[pitchSet]
	pitchName := ""
	if index < len(pitchNames) {
		pitchName = pitchNames[index]
	}
	noteOn.PitchSet = pitchSet
	noteOn.PitchSetIndex = index
	noteOn.PitchSetName = pitchName
}

func pitchSetIndexForX(x float64, pitchCount int) int {
	if pitchCount <= 0 || math.IsNaN(x) {
		return 0
	}
	if x <= 0.0 {
		return 0
	}
	if x >= 1.0 {
		return pitchCount - 1
	}
	return int(x * float64(pitchCount))
}

func uint8sToInts(values []uint8) []int {
	ints := make([]int, 0, len(values))
	for _, value := range values {
		ints = append(ints, int(value))
	}
	return ints
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
	globalZ := globalPressureShape(ce.Pos.Z, "sound")

	scaledZ := globalZ.Scaled
	// LogInfo("CursorToVelocity","scaledZ",scaledZ,"ce.Z",ce.Z,"zmin",zmin,"zmax",zmax)

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
	vel := pressureToVelocity(v, velocitymin, velocitymax)
	LogOfType("pressure", "Cursor pressure to velocity",
		"patch", patch.Name(),
		"gid", ce.GID,
		"ddu", ce.Ddu,
		"rawZ", ce.Pos.Z,
		"globalZMin", globalZ.ZMin,
		"globalZMax", globalZ.ZMax,
		"globalCurve", globalZ.Curve,
		"globalScaledZ", globalZ.Scaled,
		"scaledZ", scaledZ,
		"volstyle", volstyle,
		"velocitymin", velocitymin,
		"velocitymax", velocitymax,
		"velocity", vel)
	return uint8(vel)
}

func (logic *PatchLogic) generateVisualsFromCursor(ce CursorEvent) {
	// send an OSC message to Resolume
	visualCE := logic.cursorToVisualPressure(ce)
	msg := CursorToOscMsg(visualCE)
	TheResolume().ToFreeFramePlugin(logic.patch.Name(), msg)
}

func (logic *PatchLogic) cursorToVisualPressure(ce CursorEvent) CursorEvent {
	patch := logic.patch
	globalZ := globalPressureShape(ce.Pos.Z, "visual")
	patchZMin := patch.GetFloat("misc.pressurevisualzmin")
	patchZMax := patch.GetFloat("misc.pressurevisualzmax")
	scaledZ := scalePressureRange(globalZ.Scaled, patchZMin, patchZMax)
	LogOfType("pressure", "Cursor pressure to visual",
		"patch", patch.Name(),
		"gid", ce.GID,
		"ddu", ce.Ddu,
		"rawZ", ce.Pos.Z,
		"globalZMin", globalZ.ZMin,
		"globalZMax", globalZ.ZMax,
		"globalCurve", globalZ.Curve,
		"globalScaledZ", globalZ.Scaled,
		"patchZMin", patchZMin,
		"patchZMax", patchZMax,
		"scaledZ", scaledZ)
	ce.Pos.Z = scaledZ
	return ce
}

func (logic *PatchLogic) liveStepperRoute() string {
	return NewSamplePlaybackDomain(logic.patch).route()
}

func (logic *PatchLogic) bssSampleMode() bool {
	return NewSamplePlaybackDomain(logic.patch).Enabled()
}

func (logic *PatchLogic) proSampleMode() bool {
	return proSamplePlaybackEnabled(logic.patch)
}

func (logic *PatchLogic) liveRouteIncludesBidule() bool {
	return NewSamplePlaybackDomain(logic.patch).RouteIncludesBidule()
}

func (logic *PatchLogic) liveRouteIncludesSamples() bool {
	return NewSamplePlaybackDomain(logic.patch).RouteIncludesSamples()
}

func (logic *PatchLogic) scheduleLiveNoteOn(ce CursorEvent, noteOn *NoteOn, atClick Clicks) {
	if logic.proSampleMode() {
		logic.scheduleProSampleNoteOn(ce, noteOn, atClick)
		return
	}
	scheduled := false
	if logic.liveRouteIncludesBidule() {
		ScheduleAt(atClick, ce.Tag, noteOn)
		scheduled = true
	}
	if logic.liveRouteIncludesSamples() && theStepper != nil {
		synth := theStepper.config.SamplesplitterSynthForPatch(logic.patch.Name())
		if synth == nil {
			if scheduled {
				logic.logPitchSetUsed(noteOn)
			}
			return
		}
		velocity := samplePlaybackVelocityFromPressure(logic.patch, ce.Pos.Z)
		ScheduleAt(atClick, ce.Tag, NewPitchBend(synth, theStepper.pitchBendValue(ce.Pos.Z)))
		ScheduleAt(atClick, ce.Tag, NewNoteOn(synth, noteOn.Pitch, velocity))
		ScheduleAt(atClick+1, ce.Tag, NewPitchBend(synth, MidiPitchBendCenter))
		scheduled = true
	}
	if scheduled {
		logic.logPitchSetUsed(noteOn)
	}
}

func (logic *PatchLogic) scheduleProSampleNoteOn(ce CursorEvent, noteOn *NoteOn, atClick Clicks) {
	if noteOn == nil {
		return
	}
	if err := EnsureProSamplePlaybackService(); err != nil {
		LogWarn("scheduleProSampleNoteOn: sample playback unavailable", "patch", logic.patch.Name(), "err", err)
		return
	}
	channel := SamplePlaybackChannelForPatch(logic.patch.Name())
	ScheduleAt(atClick, ce.Tag, &SamplePlaybackStart{
		Patch:          logic.patch.Name(),
		SigilChannel:   channel,
		SampleSelector: int(noteOn.Pitch),
		Velocity:       int(noteOn.Velocity),
		PitchBend:      MidiPitchBendCenter,
		VoiceKey:       samplePlaybackVoiceKey(ce),
	})
	logic.logPitchSetUsed(noteOn)
}

func (logic *PatchLogic) logPitchSetUsed(noteOn *NoteOn) {
	if noteOn == nil || noteOn.PitchSet == "" {
		return
	}
	pitches := PitchSets[noteOn.PitchSet]
	pitchNames := PitchSetPitchNames[noteOn.PitchSet]
	LogOfType("pitchset", "PitchSet used",
		"patch", logic.patch.Name(),
		"pitchset", noteOn.PitchSet,
		"index", noteOn.PitchSetIndex,
		"pitch", noteOn.Pitch,
		"pitchname", noteOn.PitchSetName,
		"pitches", uint8sToInts(pitches),
		"pitchnames", pitchNames)
}

func (logic *PatchLogic) scheduleLiveNoteOff(ce CursorEvent, noteOff *NoteOff, atClick Clicks) {
	if logic.proSampleMode() {
		if noteOff == nil {
			return
		}
		channel := SamplePlaybackChannelForPatch(logic.patch.Name())
		ScheduleAt(atClick, ce.Tag, &SamplePlaybackStop{
			Patch:          logic.patch.Name(),
			SigilChannel:   channel,
			SampleSelector: int(noteOff.Pitch),
			VoiceKey:       samplePlaybackVoiceKey(ce),
		})
		return
	}
	if logic.liveRouteIncludesBidule() {
		ScheduleAt(atClick, ce.Tag, noteOff)
	}
	if logic.liveRouteIncludesSamples() && theStepper != nil {
		synth := theStepper.config.SamplesplitterSynthForPatch(logic.patch.Name())
		if synth == nil {
			return
		}
		ScheduleAt(atClick, ce.Tag, NewNoteOff(synth, noteOff.Pitch, noteOff.Velocity))
	}
}

func (logic *PatchLogic) generateSoundFromCursor(ce CursorEvent, cursorStyle string) {

	LogOfType("gensound", "generateSoundFromCursor", "cursor", ce.GID, "ce", ce)

	if logic.bssSampleMode() {
		logic.generateBSSSampleFromCursor(ce)
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

func (logic *PatchLogic) generateBSSSampleFromCursor(ce CursorEvent) {
	logic.mutex.Lock()
	defer logic.mutex.Unlock()

	ac, ok := theCursorManager.getActiveCursorFor(ce.GID)
	if !ok {
		LogWarn("generateBSSSampleFromCursor: no active cursor", "gid", ce.GID)
		return
	}
	NewSamplePlaybackDomain(logic.patch).HandleCursor(ce, ac)
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
		if theStepper != nil && !IsBSSInitialPage() {
			theStepper.RecordNoteOn(ce.Tag, noteOn, ce.Pos.Z, atClick, quant)
		}
		noteOff := NewNoteOffFromNoteOn(noteOn)
		atClick += QuarterNote
		logic.scheduleLiveNoteOff(ce, noteOff, atClick)
		if theStepper != nil && !IsBSSInitialPage() {
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
			if theStepper != nil && !IsBSSInitialPage() {
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
		if theStepper != nil && !IsBSSInitialPage() {
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
		deltaztrignote := getGlobalPressureFloat("global.deltaztrignote", 0.2)
		deltaztrigcontroller := getGlobalPressureFloat("global.deltaztrigcontroller", 0.02)

		deltay := math.Abs(float64(ac.Previous.Pos.Y - ce.Pos.Y))
		deltaytrig := getGlobalPressureFloat("global.deltaytrig", 0.08)

		// logic.generateController(ac)
		if !logic.proSampleMode() && patch.Get("sound.controllerstyle") == "modulationonly" {
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
			if theStepper != nil && !IsBSSInitialPage() {
				theStepper.RecordNoteOff(ce.Tag, noteOff, offClick)
			}

			thisClick := logic.nextQuant(cc, c2q)
			if thisClick < offClick {
				thisClick = offClick
			}

			logic.scheduleLiveNoteOn(ce, newNoteOn, thisClick)
			if theStepper != nil && !IsBSSInitialPage() {
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
			if theStepper != nil && !IsBSSInitialPage() {
				theStepper.RecordNoteOff(ce.Tag, noteOff, offClick+1)
			}
			// delete(logic.cursorNote, ce.Gid)
		}
	}
}

func (logic *PatchLogic) nextQuant(t Clicks, q Clicks) Clicks {
	return nextQuant(t, q)
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
