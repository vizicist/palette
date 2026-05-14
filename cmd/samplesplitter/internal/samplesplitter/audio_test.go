package samplesplitter

import (
	"testing"
)

func TestRenderSegmentAppliesPitchAndBounds(t *testing.T) {
	buf := &audioBuffer{samples: make([]int16, audioSampleRate)}
	for i := range buf.samples {
		buf.samples[i] = int16(i % 32767)
	}

	pcm, err := renderSegment(buf, 0, 1, 2, 127)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(pcm), audioSampleRate/2; got != want {
		t.Fatalf("len(pcm) = %d, want %d", got, want)
	}
}

func TestRenderSegmentFadesEdges(t *testing.T) {
	buf := &audioBuffer{samples: make([]int16, audioSampleRate/10)}
	for i := range buf.samples {
		buf.samples[i] = 10000
	}

	pcm, err := renderSegment(buf, 0, 0.1, 1, 127)
	if err != nil {
		t.Fatal(err)
	}
	first := pcm[0]
	middle := pcm[len(pcm)/2]
	last := pcm[len(pcm)-1]
	if first != 0 {
		t.Fatalf("first sample = %d, want fade-in start at 0", first)
	}
	if middle <= 9000 {
		t.Fatalf("middle sample = %d, want near full volume", middle)
	}
	if last != 0 {
		t.Fatalf("last sample = %d, want fade-out end at 0", last)
	}
}

func TestAudioManagerSyncActiveVoices(t *testing.T) {
	state := NewState(DefaultConfig())
	audio := &AudioManager{
		state:  state,
		voices: map[string]*audioVoice{"preview": {}, "midi-0-48": {}},
	}

	audio.syncActiveVoicesLocked()
	snapshot := state.Snapshot()
	if len(snapshot.ActiveVoices) != 2 {
		t.Fatalf("len(ActiveVoices) = %d, want 2 (%v)", len(snapshot.ActiveVoices), snapshot.ActiveVoices)
	}
	if snapshot.ActiveVoices[0] != "midi-0-48" || snapshot.ActiveVoices[1] != "preview" {
		t.Fatalf("ActiveVoices = %v, want sorted names", snapshot.ActiveVoices)
	}
}

func TestAudioManagerMixCompletesOneShotVoice(t *testing.T) {
	audio := &AudioManager{
		voices: map[string]*audioVoice{
			"short": {pcm: []int16{1000}},
		},
	}
	out := make([]byte, 4)

	completed := audio.mixIntoLocked(out)
	if len(completed) != 1 || completed[0] != "short" {
		t.Fatalf("completed = %v, want short", completed)
	}
}

func TestAudioManagerMixLoopsVoice(t *testing.T) {
	audio := &AudioManager{
		voices: map[string]*audioVoice{
			"loop": {pcm: []int16{1000, -1000}, loop: true},
		},
	}
	out := make([]byte, 8)

	completed := audio.mixIntoLocked(out)
	if len(completed) != 0 {
		t.Fatalf("completed = %v, want none", completed)
	}
	if audio.voices["loop"].position != 2 {
		t.Fatalf("position = %d, want 2", audio.voices["loop"].position)
	}
}

func TestAudioManagerStopNoteFallsBackToChannel(t *testing.T) {
	audio := &AudioManager{
		voices: map[string]*audioVoice{
			"midi-0-48": {pcm: []int16{1000}, loop: true},
		},
		channelVoices: map[int]string{0: "midi-0-48"},
		noteVoices:    map[[2]int]string{{0, 48}: "midi-0-48"},
	}

	audio.StopNote(0, 60)
	if len(audio.voices) != 0 {
		t.Fatalf("voices = %v, want stopped channel", audio.voices)
	}
}
