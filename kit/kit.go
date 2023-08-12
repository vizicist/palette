package kit

import (
	"fmt"
	"runtime"
	"strconv"
	"bytes"
	midi "gitlab.com/gomidi/midi/v2"
)

type Kit struct {
	MidiOctaveShift      int
	Midisetexternalscale bool
	Midithru             bool
	MidiThruScadjust     bool
	MidiNumDown          int
}

func (k *Kit) Init() {

	TheCursorManager = NewCursorManager()
	TheScheduler = NewScheduler()
	TheAttractManager = NewAttractManager()
	TheQuadPro = NewQuadPro()

	enabled, err := GetParamBool("engine.attractenabled")
	LogIfError(err)
	TheAttractManager.SetAttractEnabled(enabled)

	InitSynths()

	TheQuadPro.Start()
	go TheScheduler.Start()

}

var TheKit *Kit

func NewKit() *Kit {
	TheKit = &Kit{ }
	return TheKit
}

func (k *Kit) ScheduleCursorEvent(ce CursorEvent) {

	// schedule CursorEvents rather than handle them right away.
	// This makes it easier to do looping, among other benefits.

	ScheduleAt(CurrentClick(), ce.Tag, ce)
}

func (k *Kit) HandleMidiEvent(me MidiEvent) {
	TheHost.SendToOscClients(MidiToOscMsg(me))

	if k.Midithru {
		LogOfType("midi","PassThruMIDI", "msg", me.Msg)
		if k.MidiThruScadjust {
			LogWarn("PassThruMIDI, midiThruScadjust needs work", "msg", me.Msg)
		}
		ScheduleAt(CurrentClick(), "midi", me.Msg)
	}
	if k.Midisetexternalscale {
		k.handleMIDISetScaleNote(me)
	}

	err := TheQuadPro.onMidiEvent(me)
	LogIfError(err)

	if TheErae.enabled {
		TheErae.onMidiEvent(me)
	}
}

func (k *Kit) handleMIDISetScaleNote(me MidiEvent) {
	if !me.HasPitch() {
		return
	}
	pitch := me.Pitch()
	if me.Msg.Is(midi.NoteOnMsg) {

		if k.MidiNumDown < 0 {
			// may happen when there's a Read error that misses a noteon
			k.MidiNumDown = 0
		}

		// If there are no notes held down (i.e. this is the first), clear the scale
		if k.MidiNumDown == 0 {
			LogOfType("scale", "Clearing external scale")
			ClearExternalScale()
		}
		SetExternalScale(pitch, true)
		k.MidiNumDown++

		// adjust the octave shift based on incoming MIDI
		newshift := 0
		if pitch < 60 {
			newshift = -1
		} else if pitch > 72 {
			newshift = 1
		}
		if newshift != k.MidiOctaveShift {
			k.MidiOctaveShift = newshift
			LogOfType("midi","MidiOctaveShift changed", "octaveshift", k.MidiOctaveShift)
		}
	} else if me.Msg.Is(midi.NoteOffMsg) {
		k.MidiNumDown--
	}
}


func GoroutineID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

func JsonObject(args ...string) string {
	s := JsonString(args...)
	return "{ " + s + " }"
}

func JsonString(args ...string) string {
	if len(args)%2 != 0 {
		LogWarn("ApiParams: odd number of arguments", "args", args)
		return ""
	}
	params := ""
	sep := ""
	for n := range args {
		if n%2 == 0 {
			params = params + sep + "\"" + args[n] + "\": \"" + args[n+1] + "\""
		}
		sep = ", "
	}
	return params
}

func BoundValueZeroToOne(v float64) float32 {
	if v < 0.0 {
		return 0.0
	}
	if v > 1.0 {
		return 1.0
	}
	return float32(v)
}

func GetNameValue(apiargs map[string]string) (name string, value string, err error) {
	name, ok := apiargs["name"]
	if !ok {
		err = fmt.Errorf("missing name argument")
		return
	}
	value, ok = apiargs["value"]
	if !ok {
		err = fmt.Errorf("missing value argument")
		return
	}
	return
}
