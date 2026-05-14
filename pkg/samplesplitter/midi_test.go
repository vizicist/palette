package samplesplitter

import (
	"path/filepath"
	"testing"
)

func TestPlanNoteOnUsesSigilSampleForChannel(t *testing.T) {
	state := NewState(DefaultConfig())
	state.SigilSamples["oracle"] = SampleState{
		Sigil:       "oracle",
		CurrentFile: filepath.Join("mp3s", "oracle-test.mp3"),
		CueData: &CueData{
			Duration:   4,
			Splits:     []float64{0, 1, 2, 3},
			PeakStarts: []float64{0.2, 1.25, 2.5, 3.25},
		},
	}
	state.SetPitchBend(1, 4096)

	req, err := state.PlanNoteOn(50, 110, 1)
	if err != nil {
		t.Fatal(err)
	}
	if req.Sigil != "oracle" {
		t.Fatalf("Sigil = %q, want oracle", req.Sigil)
	}
	if req.SplitIndex != 2 {
		t.Fatalf("SplitIndex = %d, want 2", req.SplitIndex)
	}
	if req.StartSec != 2.5 {
		t.Fatalf("StartSec = %v, want peak start 2.5", req.StartSec)
	}
	if req.PitchSemitones != 6 {
		t.Fatalf("PitchSemitones = %v, want 6", req.PitchSemitones)
	}
}

func TestPlanNoteOnMapsLowNotesAcrossSplits(t *testing.T) {
	state := NewState(DefaultConfig())
	state.CurrentFile = filepath.Join("mp3s", "fallback.mp3")
	state.CueData = &CueData{
		Duration: 4,
		Splits:   []float64{0, 1, 2, 3},
	}

	req, err := state.PlanNoteOn(24, 100, 9)
	if err != nil {
		t.Fatal(err)
	}
	if req.SplitIndex != 2 {
		t.Fatalf("SplitIndex = %d, want 2", req.SplitIndex)
	}
}

func TestPreviewPitchBendAffectsPlaybackRequest(t *testing.T) {
	state := NewState(DefaultConfig())
	state.SetCurrent(filepath.Join("mp3s", "voice.mp3"), CueData{
		Duration:   2,
		Splits:     []float64{0, 1},
		PeakStarts: []float64{0.25, 1.25},
	}, nil)
	state.SetPitchBendSemitones(-1, -12)

	req, err := state.PlanPreview(0, "preview", 100)
	if err != nil {
		t.Fatal(err)
	}
	if req.PitchSemitones != -12 {
		t.Fatalf("PitchSemitones = %v, want -12", req.PitchSemitones)
	}
	if req.PitchRatio != 0.5 {
		t.Fatalf("PitchRatio = %v, want 0.5", req.PitchRatio)
	}
	if state.Snapshot().PitchBendSemitones != -12 {
		t.Fatalf("snapshot pitch bend = %v, want -12", state.Snapshot().PitchBendSemitones)
	}
}

func TestMIDIManagerHandlesChannelMessages(t *testing.T) {
	state := NewState(DefaultConfig())
	state.SigilSamples["chaos"] = SampleState{
		Sigil:       "chaos",
		CurrentFile: filepath.Join("mp3s", "chaos-test.mp3"),
		CueData: &CueData{
			Duration: 2,
			Splits:   []float64{0, 1},
		},
	}
	manager := &MIDIManager{state: state}

	manager.handlePitchBend(0, 8191)
	manager.handleNoteOn(0, 49, 100)

	snap := state.Snapshot()
	if snap.MIDIActivityCount != 2 {
		t.Fatalf("MIDIActivityCount = %d, want 2", snap.MIDIActivityCount)
	}
	if snap.LastPlayback == nil {
		t.Fatal("LastPlayback is nil")
	}
	if snap.LastPlayback.Sigil != "chaos" || snap.LastPlayback.SplitIndex != 1 {
		t.Fatalf("LastPlayback = %+v, want chaos split 1", snap.LastPlayback)
	}
	if snap.LastPlayback.PitchSemitones <= 11.9 {
		t.Fatalf("PitchSemitones = %v, want close to +12", snap.LastPlayback.PitchSemitones)
	}
}

func TestMIDIManagerVelocityZeroStopsAudio(t *testing.T) {
	state := NewState(DefaultConfig())
	audio := &AudioManager{
		voices: map[string]*audioVoice{
			"midi-0-48": {pcm: []int16{1000}, loop: true},
		},
		channelVoices: map[int]string{0: "midi-0-48"},
		noteVoices:    map[[2]int]string{{0, 48}: "midi-0-48"},
	}
	manager := &MIDIManager{state: state, audio: audio}

	manager.handleNoteOn(0, 48, 0)
	if len(audio.voices) != 0 {
		t.Fatalf("voices = %v, want velocity-zero note_on to stop audio", audio.voices)
	}
	if state.Snapshot().LastPlayback.Type != "note_off" {
		t.Fatalf("LastPlayback = %+v, want note_off", state.Snapshot().LastPlayback)
	}
}
