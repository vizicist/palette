package kit

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	midi "gitlab.com/gomidi/midi/v2"
)

var MidiOctaveShift int
var Midisetexternalscale bool
var Midithru bool
var MidiThruScadjust bool
var MidiNumDown int

var TheHost Host

func RegisterHost(h Host) {
	TheHost = h
}

func Init() error {

	InitParamDefs()
	if TheHost == nil {
		return fmt.Errorf("kit.Init called without registering a Host")
	}

	LoadEngineParams("_Current.json")
	InitLogTypes()

	TheCursorManager = NewCursorManager()
	TheScheduler = NewScheduler()
	TheAttractManager = NewAttractManager()
	TheQuadPro = NewQuadPro()

	err := TheHost.Init()
	if err != nil {
		return err
	}

	enabled, err := GetParamBool("engine.attractenabled")
	LogIfError(err)
	TheAttractManager.SetAttractEnabled(enabled)

	InitSynths()

	return nil
}

func StartEngine() {
	TheHost.Start()
	TheQuadPro.Start()
	go TheScheduler.Start()
}

func LoadEngineParams(fname string) {
	// First set the defaults
	for nm, d := range ParamDefs {
		if strings.HasPrefix(nm, "engine.") {
			err := EngineParams.Set(nm, d.Init)
			LogIfError(err)
		}
	}
	bytes, err := TheHost.GetSavedData("engine", fname)
	if err != nil {
		TheHost.LogError(err)
		return
	}
	paramsmap, err := LoadParamsMap(bytes)
	if err != nil {
		TheHost.LogError(err)
		return
	}
	EngineParams.ApplyValuesFromMap("engine", paramsmap, EngineParams.Set)
}

func ScheduleCursorEvent(ce CursorEvent) {

	// schedule CursorEvents rather than handle them right away.
	// This makes it easier to do looping, among other benefits.
	LogOfType("cursor", "ScheduleCursorEvent", "ce", ce)

	ScheduleAt(CurrentClick(), ce.Tag, ce)
}

func HandleMidiEvent(me MidiEvent) {
	TheHost.SendToOscClients(MidiToOscMsg(me))

	if Midithru {
		LogOfType("midi", "PassThruMIDI", "msg", me.Msg)
		if MidiThruScadjust {
			LogWarn("PassThruMIDI, midiThruScadjust needs work", "msg", me.Msg)
		}
		ScheduleAt(CurrentClick(), "midi", me.Msg)
	}
	if Midisetexternalscale {
		handleMIDISetScaleNote(me)
	}

	err := TheQuadPro.onMidiEvent(me)
	LogIfError(err)

	if TheErae.enabled {
		TheErae.onMidiEvent(me)
	}
}

func handleMIDISetScaleNote(me MidiEvent) {
	if !me.HasPitch() {
		return
	}
	pitch := me.Pitch()
	if me.Msg.Is(midi.NoteOnMsg) {

		if MidiNumDown < 0 {
			// may happen when there's a Read error that misses a noteon
			MidiNumDown = 0
		}

		// If there are no notes held down (i.e. this is the first), clear the scale
		if MidiNumDown == 0 {
			LogOfType("scale", "Clearing external scale")
			ClearExternalScale()
		}
		SetExternalScale(pitch, true)
		MidiNumDown++

		// adjust the octave shift based on incoming MIDI
		newshift := 0
		if pitch < 60 {
			newshift = -1
		} else if pitch > 72 {
			newshift = 1
		}
		if newshift != MidiOctaveShift {
			MidiOctaveShift = newshift
			LogOfType("midi", "MidiOctaveShift changed", "octaveshift", MidiOctaveShift)
		}
	} else if me.Msg.Is(midi.NoteOffMsg) {
		MidiNumDown--
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

var FirstTime = true
var Time0 = time.Time{}

// Uptime returns the number of seconds since the program started.
func Uptime() float64 {
	now := time.Now()
	if FirstTime {
		FirstTime = false
		Time0 = now
	}
	return now.Sub(Time0).Seconds()
}
