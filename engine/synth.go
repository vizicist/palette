package engine

import (
	"encoding/json"
	"os"
	// "github.com/vizicist/portmidi"
)

type PortChannel struct {
	port    string // as used by Portmidi
	channel int    // 0-15 for MIDI channels 1-16
}

var PortChannels map[PortChannel]*MIDIChannelOutput

type Synth struct {
	portchannel PortChannel
	bank        int // 0 if not set
	program     int // 0 if note set
	// midiChannelOut *MidiChannelOutput
	noteDown      []bool
	noteDownCount []int
}

var Synths map[string]*Synth

func InitSynths() {

	Synths = make(map[string]*Synth)

	filename := ConfigFilePath("synths.json")
	bytes, err := os.ReadFile(filename)
	if err != nil {
		Log.Debugf("InitSynths: ReadFile of %s failed, err=%s\n", filename, err)
		return
	}

	type synth struct {
		Name    string `json:"name"`
		Port    string `json:"port"`
		Channel int    `json:"channel"`
		Bank    int    `json:"bank"`
		Program int    `json:"program"`
	}

	var jsynths struct {
		Synths []synth `json:"synths"`
	}
	json.Unmarshal(bytes, &jsynths)

	for i := range jsynths.Synths {
		nm := jsynths.Synths[i].Name
		port := jsynths.Synths[i].Port
		channel := jsynths.Synths[i].Channel
		bank := jsynths.Synths[i].Bank
		program := jsynths.Synths[i].Program

		Synths[nm] = NewSynth(port, channel, bank, program)
	}
	Log.Debugf("Synths loaded, len=%d\n", len(Synths))
}

func (synth *Synth) ClearNoteDowns() {
	// Log.Debugf("ClearNoteDowns: clearing noteDowns for synth=%p port=%s chan=%d prog=%d bank=%d\n", synth, synth.portchannel.port, synth.portchannel.channel, synth.program, synth.bank)
	for i := range synth.noteDown {
		synth.noteDown[i] = false
		synth.noteDownCount[i] = 0
	}
}

func NewSynth(port string, channel int, bank int, program int) *Synth {

	synthoutput := ConfigBoolWithDefault("generatesound", true)

	// If there's already a Synth for this PortChannel, error

	portchannel := PortChannel{port: port, channel: channel}

	var midiChannelOut *MIDIChannelOutput
	if synthoutput {
		midiChannelOut = MIDI.openChannelOutput(portchannel)
	} else {
		midiChannelOut = MIDI.openFakeChannelOutput(port, channel)
	}

	if midiChannelOut == nil {
		Log.Debugf("InitSynths: Unable to open port=%s\n", port)
		return nil
	}
	sp := &Synth{
		portchannel: portchannel,
		bank:        bank,
		program:     program,
		// midiChannelOut: midiChannelOut,
		noteDown:      make([]bool, 128),
		noteDownCount: make([]int, 128),
	}
	sp.ClearNoteDowns() // debugging, shouldn't be needed
	return sp
}

// SendANO sends all-notes-off
func SendANOToSynth(synthName string) {
	synth, ok := Synths[synthName]
	if !ok {
		Log.Debugf("SendANOToSynth: no such synth - %s\n", synthName)
		return
	}
	if synth == nil {
		// We don't complain, we assume the inability to open the
		// synth named synthName has already been logged.
		return
	}
	mc := MIDI.GetMidiChannelOutput(synth.portchannel)
	if mc == nil {
		// Assumes errs are logged in GetMidiChannelOutput
		return
	}

	// This only sends the bank and/or program if they change
	mc.SendBankProgram(synth.bank, synth.program)

	synth.ClearNoteDowns()
	// for i := range synth.noteDown {
	// 	synth.noteDown[i] = false
	// }
	// Log.Debugf("SendANOToSynth: synth=%s\n", synthName)
	Log.Debugf("SendANOToSynth needs work\n")
	/*
		status := 0xb0 | (synth.portchannel.channel - 1)
		mc.midiDeviceOutput.stream.WriteShort(int64(status), int64(0x7b), int64(0x00))
	*/
}

func SendControllerToSynth(synthName string, cnum int, cval int) {
	synth, ok := Synths[synthName]
	if !ok {
		Log.Debugf("SendControllerToSynth: no such synth - %s\n", synthName)
		return
	}
	mc := MIDI.GetMidiChannelOutput(synth.portchannel)
	if mc == nil {
		// Assumes errs are logged in GetMidiChannelOutput
		return
	}

	// This only sends the bank and/or program if they change
	mc.SendBankProgram(synth.bank, synth.program)

	Log.Debugf("SendControlToSynth needs work\n")
	/*
		e := portmidi.Event{
			Timestamp: time.Now(),
			Status:    int64(synth.portchannel.channel - 1),
			Data1:     int64(cnum),
			Data2:     int64(cval),
		}
		e.Status |= 0xb0
		if Debug.MIDI {
			Log.Debugf("SendControllerToSynth: synth=%s status=0x%02x data1=%d data2=%d\n", mc.midiDeviceOutput.Name(), e.Status, e.Data1, e.Data2)
		}
		if e.Data2 > 0x7f {
			Log.Debugf("SendControllerToSynth: Hey! Data2 shouldn't be > 0x7f\n")
		} else {
			mc.midiDeviceOutput.stream.WriteShort(e.Status, e.Data1, e.Data2)
		}
	*/
}

// SendNote sends MIDI output for a Note
func SendNoteToSynth(note *Note) {
	synthName := note.Synth
	if synthName == "" {
		synthName = "default"
	}
	synth, ok := Synths[synthName]
	if !ok {
		Log.Debugf("SendNoteToSynth: no such synth - %s\n", synthName)
		return
	}
	if synth == nil {
		// We don't complain, we assume the inability to open the
		// synth named synthName has already been logged.
		return
	}
	mc := MIDI.GetMidiChannelOutput(synth.portchannel)
	if mc == nil {
		// Assumes errs are logged in GetMidiChannelOutput
		return
	}

	// This only sends the bank and/or program if they change
	mc.SendBankProgram(synth.bank, synth.program)

	status := byte(synth.portchannel.channel - 1)
	data1 := byte(note.Pitch)
	data2 := byte(note.Velocity)
	if note.TypeOf == "noteon" && note.Velocity == 0 {
		Log.Debugf("MIDIIO.SendNote: noteon with velocity==0 is changed to a noteoff\n")
		note.TypeOf = "noteoff"
	}
	switch note.TypeOf {
	case "noteon":
		status |= 0x90

		// We now allow multiple notes with the same pitch,
		// which assumes the synth handles it okay.
		// There might need to be an option to
		// automatically send a noteOff before sending the noteOn.
		// if synth.noteDown[note.Pitch] {
		//     Log.Debugf("SendNoteToSynth: Ignoring second noteon for synth=%p synth=%s chan=%d pitch=%d\n", synth, synthName, synth.portchannel.channel, note.Pitch)
		// }

		synth.noteDown[note.Pitch] = true
		synth.noteDownCount[note.Pitch]++
		if Debug.MIDI {
			Log.Debugf("SendNoteToSynth: synth=%p noteon noteCount>0 chan=%d pitch=%d downcount=%d\n", synth, synth.portchannel.channel, note.Pitch, synth.noteDownCount[note.Pitch])
		}
	case "noteoff":
		status |= 0x80
		data2 = 0
		synth.noteDown[note.Pitch] = false
		synth.noteDownCount[note.Pitch]--
		if Debug.MIDI {
			Log.Debugf("SendNoteToSynth: synth=%p noteoff pitch=%d downcount is now %d\n", synth, note.Pitch, synth.noteDownCount[note.Pitch])
		}
	case "controller":
		status |= 0xB0
	case "progchange":
		status |= 0xC0
	case "chanpressure":
		status |= 0xD0
	case "pitchbend":
		status |= 0xE0
	default:
		Log.Debugf("SendNoteToSynth: can't handle Note TypeOf=%v\n", note.TypeOf)
		return
	}

	if Debug.MIDI {
		Log.Debugf("SendNoteToSynth: synth=%s status=0x%02x data1=%d data2=%d\n", mc.output.String(), status, data1, data2)
	}

	err := mc.output.Send([]byte{status, data1, data2})
	if err != nil {
		Log.Debugf("output.Send: err=%s\n", err)
	}
	// mc.midiDeviceOutput.stream.WriteShort(e.Status, e.Data1, e.Data2)
}
