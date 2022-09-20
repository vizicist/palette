package block

import (
	"fmt"
	"log"
	"strconv"

	"github.com/vizicist/palette/engine"
)

func init() {
	engine.RegisterBlock("midigen", NewMidiGen1)
}

//////////////////////////////////////////////////////////////////////

// MidiGen1 is a trivial midification
type MidiGen1 struct {
	channel          int
	port             string
	MIDIOctaveShift  int
	TransposePitch   int
	synth            string
	useExternalScale bool // if true, scadjust uses "external" Scale
	externalScale    *engine.Scale
	MIDINumDown      int
	notesOn          map[string]*engine.Note
}

// NewMidiGen1 xxx
func NewMidiGen1(ctx *engine.EContext) engine.Block {
	return &MidiGen1{
		channel:          0,
		port:             "",
		MIDIOctaveShift:  0,
		TransposePitch:   0,
		synth:            "",
		useExternalScale: false,
		MIDINumDown:      0,
		notesOn:          make(map[string]*engine.Note),
	}
}

// AcceptEngineMsg xxx
func (alg *MidiGen1) AcceptEngineMsg(ctx *engine.EContext, cmd engine.Cmd) string {

	subj := cmd.Subj
	values := cmd.Values
	switch subj {
	case "setparam":
		name, ok := values["name"]
		if !ok {
			return engine.ErrorResult("MidiGen1: setparam missing name")
		}
		value, ok := values["value"]
		if !ok {
			return engine.ErrorResult("MidiGen1: setparam missing value")
		}
		switch name {
		case "channel":
			i, err := strconv.Atoi(value)
			if err != nil {
				e := fmt.Sprintf("MidiAlg1: Bad value of channel (%s)\n", value)
				return engine.ErrorResult(e)
			}
			alg.channel = i
		case "port":
			alg.port = value
		default:
			e := fmt.Sprintf("MidiAlg1: Unknown parameter (%s)\n", name)
			return engine.ErrorResult(e)
		}

	case "cursor3d":

		ddu := cmd.ValuesString("ddu", "")
		id := cmd.ValuesString("id", "")
		x := cmd.ValuesFloat("x", 0.0)
		y := cmd.ValuesFloat("y", 0.0)
		z := cmd.ValuesFloat("z", 0.0)

		switch ddu {
		case "down":
			noteon := alg.cursorToNoteOn(x, y, z)
			old, ok := alg.notesOn[id]
			if !ok {
				// nothing assigned to msg.ID, it's new one
			} else {
				log.Printf("MidiAlg1: got cursor down for existing cid=%s\n", id)
				// terminate old note first
				alg.terminateNote(ctx, old, id)
			}
			alg.notesOn[id] = noteon
			ctx.PublishCmd(engine.NewNoteCmd(noteon))

		case "drag":
			// Repeat notes when dragging
			noteon := alg.cursorToNoteOn(x, y, z)
			old, ok := alg.notesOn[id]
			if !ok {
				// nothing assigned to msg.ID, it's new one
				log.Printf("MidiAlg1: got cursor drag without down, cid=%s\n", id)
			} else {
				// Existing note during drag is ignored
				if old.Pitch == noteon.Pitch && old.Synth == noteon.Synth {
					// Identical note, do nothing
					// log.Printf("Identical note in drag pitch=%d\n", old.Pitch)
					noteon = nil
				} else {
					// new pitch or sound, terminate old note first
					alg.terminateNote(ctx, old, id)
				}
			}
			if noteon != nil {
				alg.notesOn[id] = noteon
				ctx.PublishCmd(engine.NewNoteCmd(noteon))
			}

		case "up":
			old, ok := alg.notesOn[id]
			if !ok {
				log.Printf("MidiAlg1: got cursor up without down, cid=%s\n", id)
			} else {
				// terminate using old note, don't use the values from the UP event
				alg.terminateNote(ctx, old, id)
			}
		default:
			log.Printf("Bad value of DownDragUp\n")
		}
	}
	return ""
}

func (alg *MidiGen1) terminateNote(ctx *engine.EContext, old *engine.Note, cid string) {
	noteoff := engine.NewNoteOff(old.Pitch, old.Velocity, old.Synth)
	ctx.PublishCmd(engine.NewNoteCmd(noteoff))
	delete(alg.notesOn, cid)
}

func (alg *MidiGen1) cursorToNoteOn(x, y, z float32) *engine.Note {
	pitch := alg.cursorToPitch(x)
	// log.Printf("MidiGen1: cursorToNoteOn: pitch=%d\n", pitch)
	pitch = uint8(int(pitch) + alg.TransposePitch)
	velocity := alg.cursorToVelocity(x, y, z)
	return engine.NewNoteOn(pitch, velocity, alg.synth)
}

func (alg *MidiGen1) cursorToPitch(x float32) uint8 {
	/*
		pitchmin := alg.params.ParamIntValue("sound.pitchmin")
		pitchmax := alg.params.ParamIntValue("sound.pitchmax")
	*/
	pitchmin := 0
	pitchmax := 127
	dp := pitchmax - pitchmin + 1
	p1 := int(float32(x) * float32(dp))
	p := uint8(pitchmin + p1%dp)
	scale := alg.getScale()
	p = scale.ClosestTo(p)
	pnew := int(p) + 12*alg.MIDIOctaveShift
	if pnew < 0 {
		p = uint8(pnew + 12)
	} else if pnew > 127 {
		p = uint8(pnew - 12)
	} else {
		p = uint8(pnew)
	}
	return p
}

func (alg *MidiGen1) cursorToVelocity(x, y, z float32) uint8 {
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
		v = 1.0 - y
	case "pressure":
		v = z * 4.0
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

// func (alg *MidiGen1) cursorToDuration(ge engine.Cursor3DMsg) int {
// 	return 92
// }

func (alg *MidiGen1) getScale() *engine.Scale {
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
func (alg *MidiGen1) ClearExternalScale() {
	alg.externalScale = engine.MakeScale()
}

// SetExternalScale xxx
func (alg *MidiGen1) SetExternalScale(pitch int, on bool) {
	s := alg.externalScale
	for p := pitch; p < 128; p += 12 {
		s.HasNote[p] = on
	}
}

/*
func (alg *MidiGen1) handleMIDISetScaleNote(e portmidi.Event) {
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
*/
