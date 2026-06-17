package kit

import (
	"testing"
	"time"
)

type fakeMidiOut struct {
	open bool
	sent [][]byte
}

func (f *fakeMidiOut) Open() error {
	f.open = true
	return nil
}

func (f *fakeMidiOut) Close() error {
	f.open = false
	return nil
}

func (f *fakeMidiOut) IsOpen() bool {
	return f.open
}

func (f *fakeMidiOut) Number() int {
	return 1
}

func (f *fakeMidiOut) String() string {
	return "fake MIDI out"
}

func (f *fakeMidiOut) Underlying() interface{} {
	return nil
}

func (f *fakeMidiOut) Send(data []byte) error {
	f.sent = append(f.sent, append([]byte(nil), data...))
	return nil
}

func setupSynthWatchdogTest(t *testing.T) (*Synth, *fakeMidiOut, func()) {
	t.Helper()

	oldMidiIO := theMidiIO
	oldLog := theLog
	InitLog("")

	portchannel := PortChannel{port: "fake", channel: 1}
	output := &fakeMidiOut{open: true}
	state := &MidiPortChannelState{
		channel: 1,
		output:  output,
		isopen:  true,
	}
	theMidiIO = &MidiIO{
		midiPortChannelState: map[PortChannel]*MidiPortChannelState{
			portchannel: state,
		},
	}
	synth := NewSynth("test", portchannel, 0, 0, state)

	return synth, output, func() {
		theMidiIO = oldMidiIO
		theLog = oldLog
	}
}

func TestSynthNoteTrackingKeepsOverlappingNotes(t *testing.T) {
	synth, _, cleanup := setupSynthWatchdogTest(t)
	defer cleanup()

	synth.SendNoteToMidiOutput(NewNoteOn(synth, 60, 100))
	synth.SendNoteToMidiOutput(NewNoteOn(synth, 60, 100))
	synth.SendNoteToMidiOutput(NewNoteOff(synth, 60, 100))

	if got := synth.noteDownCount[60]; got != 1 {
		t.Fatalf("noteDownCount[60] = %d, want 1", got)
	}
	if !synth.noteDown[60] {
		t.Fatal("noteDown[60] = false, want true while one overlapping note remains")
	}
}

func TestSynthNoteOnVelocityZeroClearsTracking(t *testing.T) {
	synth, _, cleanup := setupSynthWatchdogTest(t)
	defer cleanup()

	synth.SendNoteToMidiOutput(NewNoteOn(synth, 60, 100))
	synth.SendNoteToMidiOutput(NewNoteOn(synth, 60, 0))

	if got := synth.noteDownCount[60]; got != 0 {
		t.Fatalf("noteDownCount[60] = %d, want 0", got)
	}
	if synth.noteDown[60] {
		t.Fatal("noteDown[60] = true, want false after note-on velocity zero")
	}
}

func TestSynthWatchdogExpiresLongNotes(t *testing.T) {
	synth, output, cleanup := setupSynthWatchdogTest(t)
	defer cleanup()

	now := time.Unix(100, 0)
	synth.recordNoteOn(60, now.Add(-31*time.Second))
	synth.recordNoteOn(61, now.Add(-10*time.Second))

	expired := synth.SendExpiredNoteOffs(now, 30*time.Second)
	if expired != 1 {
		t.Fatalf("expired = %d, want 1", expired)
	}
	if got := synth.noteDownCount[60]; got != 0 {
		t.Fatalf("noteDownCount[60] = %d, want 0", got)
	}
	if got := synth.noteDownCount[61]; got != 1 {
		t.Fatalf("noteDownCount[61] = %d, want 1", got)
	}
	if len(output.sent) != 1 {
		t.Fatalf("sent MIDI messages = %d, want 1 (%v)", len(output.sent), output.sent)
	}
	want := []byte{NoteOffStatus, 60, 0}
	for i := range want {
		if output.sent[0][i] != want[i] {
			t.Fatalf("sent MIDI = %v, want %v", output.sent[0], want)
		}
	}

	expired = synth.SendExpiredNoteOffs(now, 30*time.Second)
	if expired != 0 {
		t.Fatalf("second expired = %d, want 0", expired)
	}
}
