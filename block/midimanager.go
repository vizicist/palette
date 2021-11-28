package block

import (
	"fmt"
	"log"
	"sync"

	"github.com/vizicist/palette/engine"
	"github.com/vizicist/portmidi"
)

func init() {
	engine.RegisterBlock("midimanager", NewMidiManager)
}

//////////////////////////////////////////////////////////////////////

// MidiManager is a trivial visualization, for development
type MidiManager struct {
	context          engine.EContext
	channel          int
	port             string
	synthOutput      *engine.Synth
	activeNotes      map[string]*ActiveNote
	activeNotesMutex *sync.RWMutex
	lastActiveID     int
	MIDIOctaveShift  int
	TransposePitch   int
	synth            string
	useExternalScale bool // if true, scadjust uses "external" Scale
	externalScale    *engine.Scale
	MIDINumDown      int
}

// ActiveNote is a currently active MIDI note
type ActiveNote struct {
	id     int
	noteOn *engine.Note
}

// NewMidiManager xxx
func NewMidiManager(ctx *engine.EContext) engine.Block {
	mgr := &MidiManager{
		channel:          0,
		port:             "",
		synthOutput:      nil,
		activeNotes:      make(map[string]*ActiveNote),
		lastActiveID:     0,
		MIDIOctaveShift:  0,
		TransposePitch:   0,
		synth:            "",
		useExternalScale: false,
		MIDINumDown:      0,
	}
	mgr.ClearExternalScale()
	mgr.SetExternalScale(60%12, true) // Middle C
	return mgr
}

// AcceptEngineMsg xxx
func (alg *MidiManager) AcceptEngineMsg(ctx *engine.EContext, cmd engine.Cmd) string {
	switch cmd.Subj {
	case "Setparam":
		name := cmd.ValuesString("name", "")
		value := cmd.ValuesString("value", "")
		switch name {
		case "port":
			alg.port = value
		default:
			e := fmt.Sprintf("SpriteAlg1: Unknown parameter (%s)\n", name)
			return engine.ErrorResult(e)
		}

	case "note":
		log.Printf("MidiManager: XXX DEBUG should be sending note!?\n")
		// engine.SendNoteToSynth(values)

	case "midievent":
		log.Printf("MidiManager: MidiEventMsg\n")

	case "ano":
		log.Printf("MidiManager: got AnoMsg\n")

	default:
		log.Printf("MidiManager: Unknown message? subj=%v\n", cmd.Subj)

	}
	return ""
}

func (alg *MidiManager) getActiveNote(id string) *ActiveNote {
	alg.activeNotesMutex.RLock()
	a, ok := alg.activeNotes[id]
	alg.activeNotesMutex.RUnlock()
	if !ok {
		alg.lastActiveID++
		a = &ActiveNote{id: alg.lastActiveID}
		alg.activeNotesMutex.Lock()
		alg.activeNotes[id] = a
		alg.activeNotesMutex.Unlock()
	}
	return a
}

func (alg *MidiManager) terminateActiveNotes() {
	alg.activeNotesMutex.RLock()
	for id, a := range alg.activeNotes {
		// log.Printf("terminateActiveNotes n=%v\n", a.currentNoteOn)
		if a != nil {
			alg.sendNoteOff(a)
		} else {
			log.Printf("Hey, activeNotes entry for id=%s\n", id)
		}
	}
	alg.activeNotesMutex.RUnlock()
}

func (alg *MidiManager) generateSoundFromCursor3D(ge engine.Cursor3DMsg) {
	log.Printf("Stepper.generateSoundFromCursor3D: ce=%v\n", ge)
	a := alg.getActiveNote(ge.ID)
	switch ge.DownDragUp {
	case "down":
		// Send NOTEOFF for current note
		if a.noteOn != nil {
			// I think this happens if we get things coming in
			// faster than the checkDelay can generate the UP event.
			log.Printf("Unexpected down when currentNoteOn is non-nil!? currentNoteOn=%+v\n", a)
			// log.Printf("generateMIDI sending NoteOff before down note\n")
			alg.sendNoteOff(a)
		}
		a.noteOn = alg.cursorToNoteOn(ge)
		// log.Printf("r=%s down Setting currentNoteOn to %v!\n", alg.padName, *(a.currentNoteOn))
		// log.Printf("generateMIDI sending NoteOn for down\n")
		alg.sendNoteOn(a)
	case "drag":
		if a.noteOn == nil {
			// if we turn on playing in the middle of an existing loop,
			// we may see some drag events without a down.
			// Also, I'm seeing this pretty commonly in other situations,
			// not really sure what the underlying reason is,
			// but it seems to be harmless at the moment.
			log.Printf("drag event, a.currentNoteOn == nil?\n")
		} else {
			// log.Printf("generateMIDI sending NoteOff for previous note\n")
			alg.sendNoteOff(a)
		}
		a.noteOn = alg.cursorToNoteOn(ge)
		// log.Printf("r=%s drag Setting currentNoteOn to %v!\n", alg.padName, *(a.currentNoteOn))
		// log.Printf("generateMIDI sending NoteOn\n")
		alg.sendNoteOn(a)
	case "up":
		if a.noteOn == nil {
			// not sure why this happens, yet
			log.Printf("Unexpected UP when currentNoteOn is nil?\n")
		} else {
			alg.sendNoteOff(a)

			// XXX - the comment here is old, might have been fixed
			// HACK! There are hanging notes, seems to occur when sending multiple noteons of
			// the same pitch to the same synth, but I tried fixing it and failed. So.
			engine.SendANOToSynth(a.noteOn.Sound)

			a.noteOn = nil
			// log.Printf("r=%s UP Setting currentNoteOn to nil!\n", alg.padName)
		}
		alg.activeNotesMutex.Lock()
		delete(alg.activeNotes, ge.ID)
		alg.activeNotesMutex.Unlock()
	}
}

func (alg *MidiManager) sendNoteOn(a *ActiveNote) {

	engine.SendNoteToSynth(a.noteOn)

	/*
		ss := alg.params.ParamStringValue("visual.spritesource", "")
		if ss == "midi" {
			alg.generateSpriteFromNote(a)
		}
	*/
}

func (alg *MidiManager) sendNoteOff(a *ActiveNote) {
	n := a.noteOn
	if n == nil {
		// Not sure why this sometimes happens
		// log.Printf("HEY! sendNoteOff got a nil?\n")
		return
	}
	// the Note coming in should be a noteon
	if n.TypeOf != "noteon" {
		log.Printf("HEY! sendNoteOff expects a noteon!?")
	} else {
		noteOff := engine.NewNoteOff(n.Pitch, n.Velocity, n.Sound)
		engine.SendNoteToSynth(noteOff)
	}
}

func (alg *MidiManager) sendANO() {
	log.Printf("Stepper.sendANO: needs implementation!\n")
	/*
		if !TheRouter().generateSound {
			return
		}
		synth := r.params.ParamStringValue("sound.synth", defaultSynth)
		if synth != "" {
			if engine.Debug.MIDI {
				log.Printf("MIDI.SendANO: synth=%s\n", synth)
			}
			MIDI.SendANO(synth)
		} else {
			log.Printf("MIDI.SendANO: synth is empty?\n")
		}
	*/
}

func (alg *MidiManager) cursorToNoteOn(ge engine.Cursor3DMsg) *engine.Note {
	pitch := alg.cursorToPitch(ge)
	log.Printf("MidiAlt1: cursorToNoteOn: pitch=%d\n", pitch)
	pitch = uint8(int(pitch) + alg.TransposePitch)
	velocity := alg.cursorToVelocity(ge)
	return engine.NewNoteOn(pitch, velocity, alg.synth)
}

func (alg *MidiManager) cursorToPitch(ce engine.Cursor3DMsg) uint8 {
	/*
		pitchmin := alg.params.ParamIntValue("sound.pitchmin")
		pitchmax := alg.params.ParamIntValue("sound.pitchmax")
	*/
	pitchmin := 0
	pitchmax := 127
	dp := pitchmax - pitchmin + 1
	p1 := int(ce.X * float32(dp))
	p := uint8(pitchmin + p1%dp)
	scale := alg.getScale()
	p = scale.ClosestTo(p)
	pnew := p + uint8(12*alg.MIDIOctaveShift)
	if pnew < 0 {
		p = pnew + 12
	} else if pnew > 127 {
		p = pnew - 12
	} else {
		p = uint8(pnew)
	}
	return p
}

func (alg *MidiManager) cursorToVelocity(ce engine.Cursor3DMsg) uint8 {
	/*
		vol := alg.params.ParamStringValue("misc.vol", "fixed")
		velocitymin := alg.params.ParamIntValue("sound.velocitymin")
		velocitymax := alg.params.ParamIntValue("sound.velocitymax")
	*/
	vol := "fixed"
	velocitymin := 10
	velocitymax := 120
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
		log.Printf("Unrecognized vol value: %s, assuming %f\n", vol, v)
	}
	dv := velocitymax - velocitymin + 1
	p1 := int(v * float32(dv))
	vel := uint8(velocitymin + p1%dv)
	return uint8(vel)
}

func (alg *MidiManager) cursorToDuration(ge engine.Cursor3DMsg) int {
	return 92
}

func (alg *MidiManager) getScale() *engine.Scale {
	var scaleName string
	var scale *engine.Scale
	if alg.useExternalScale {
		scale = alg.externalScale
	} else {
		scaleName = "newage"
		scale = engine.GlobalScale(scaleName)
	}
	return scale
}

// ClearExternalScale xxx
func (alg *MidiManager) ClearExternalScale() {
	alg.externalScale = engine.MakeScale()
}

// SetExternalScale xxx
func (alg *MidiManager) SetExternalScale(pitch int, on bool) {
	s := alg.externalScale
	for p := pitch; p < 128; p += 12 {
		s.HasNote[p] = on
	}
}

func (alg *MidiManager) handleMIDISetScaleNote(e portmidi.Event) {
	status := e.Status & 0xf0
	pitch := int(e.Data1)
	if status == 0x90 {
		// If there are no notes held down (i.e. this is the first), clear the scale
		if alg.MIDINumDown < 0 {
			// this can happen when there's a Read error that misses a NOTEON
			alg.MIDINumDown = 0
		}
		if alg.MIDINumDown == 0 {
			alg.ClearExternalScale()
		}
		alg.SetExternalScale(pitch%12, true)
		alg.MIDINumDown++
		if pitch < 60 {
			alg.MIDIOctaveShift = -1
		} else if pitch > 72 {
			alg.MIDIOctaveShift = 1
		} else {
			alg.MIDIOctaveShift = 0
		}
	} else if status == 0x80 {
		alg.MIDINumDown--
	}
}
