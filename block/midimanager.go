package block

// THIS CODE WAS AN EXPERIMENT, NOT SURE IF IT'S THE RIGHT WAY TO GO

import (
	"fmt"
	"log"

	"github.com/vizicist/palette/engine"
)

func init() {
	engine.RegisterBlock("midimanager", NewMidiManager)
}

//////////////////////////////////////////////////////////////////////

// MidiManager is a trivial visualization, for development
type MidiManager struct {
	// context          engine.EContext
	channel     int
	port        string
	synthOutput *engine.Synth
	activeNotes map[string]*ActiveNote
	// activeNotesMutex *sync.RWMutex
	lastActiveID     int
	MIDIOctaveShift  int
	TransposePitch   int
	synth            string
	useExternalScale bool // if true, scadjust uses "external" Scale
	// externalScale    *engine.Scale
	MIDINumDown int
}

// ActiveNote is a currently active MIDI note
type ActiveNote struct {
	// id     int
	// noteOn *engine.Note
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

/*
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
*/

/*
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
*/

/*
func (alg *MidiManager) sendNoteOn(a *ActiveNote) {

	engine.SendNoteToSynth(a.noteOn)

	// ss := alg.params.ParamStringValue("visual.spritesource", "")
	// if ss == "midi" {
	// alg.generateSpriteFromNote(a)
	// }
}
*/

/*
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
*/

/*
func (alg *MidiManager) sendANO() {
	log.Printf("Stepper.sendANO: needs implementation!\n")
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
}
*/
