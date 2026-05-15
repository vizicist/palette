package samplesplitter

import (
	"encoding/binary"
	"testing"
)

func TestRenderSegmentAppliesPitchAndBounds(t *testing.T) {
	buf := &audioBuffer{samples: make([]int16, audioSampleRate)}
	for i := range buf.samples {
		buf.samples[i] = int16(i % 32767)
	}

	pcm, err := renderSegment(buf, 0, 1, 2, 127, false)
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

	pcm, err := renderSegment(buf, 0, 0.1, 1, 127, false)
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
			"loop": {pcm: []int16{1000, -1000}, loop: true, channel: -1},
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

func TestAudioManagerStopChannelStopsHeldNotes(t *testing.T) {
	audio := &AudioManager{
		voices: map[string]*audioVoice{
			"midi-0-48": {pcm: []int16{1000}, loop: true, channel: 0, note: 48, active: true},
			"midi-0-60": {pcm: []int16{2000}, loop: true, channel: 0, note: 60},
		},
		channelVoices: map[int]string{0: "midi-0-48"},
		channelOrder:  map[int][]string{0: {"midi-0-48", "midi-0-60"}},
		channelActive: map[int]string{0: "midi-0-48"},
		noteVoices:    map[[2]int]string{{0, 48}: "midi-0-48", {0, 60}: "midi-0-60"},
	}

	audio.StopNote(0, -1)
	if len(audio.voices) != 0 {
		t.Fatalf("voices = %v, want stopped channel", audio.voices)
	}
}

func TestAudioManagerStopNoteKeepsOtherHeldNotes(t *testing.T) {
	audio := &AudioManager{
		voices: map[string]*audioVoice{
			"midi-0-48": {pcm: []int16{1000}, loop: true, channel: 0, note: 48, active: true},
			"midi-0-60": {pcm: []int16{2000}, loop: true, channel: 0, note: 60},
		},
		channelVoices: map[int]string{0: "midi-0-48"},
		channelOrder:  map[int][]string{0: {"midi-0-48", "midi-0-60"}},
		channelActive: map[int]string{0: "midi-0-48"},
		noteVoices:    map[[2]int]string{{0, 48}: "midi-0-48", {0, 60}: "midi-0-60"},
	}

	audio.StopNote(0, 48)
	if _, ok := audio.voices["midi-0-48"]; ok {
		t.Fatalf("midi-0-48 still active after note stop")
	}
	if _, ok := audio.voices["midi-0-60"]; !ok {
		t.Fatalf("voices = %v, want other held note kept", audio.voices)
	}
	if audio.channelActive[0] != "midi-0-60" {
		t.Fatalf("channelActive = %v, want midi-0-60", audio.channelActive)
	}
	if !audio.voices["midi-0-60"].active {
		t.Fatalf("midi-0-60 should become the active held voice")
	}
}

func TestAudioManagerPlayKeepsChannelMappings(t *testing.T) {
	state := NewState(DefaultConfig())
	audio := &AudioManager{
		state:         state,
		cache:         map[string]*audioBuffer{"sample.mp3": {samples: []int16{1000, -1000}}},
		voices:        map[string]*audioVoice{},
		channelVoices: map[int]string{},
		channelOrder:  map[int][]string{},
		channelActive: map[int]string{},
		noteVoices:    map[[2]int]string{},
	}
	req := &PlaybackRequest{
		FilePath:   "sample.mp3",
		VoiceKey:   "midi-0-23",
		Channel:    0,
		Note:       23,
		EndSec:     0.00005,
		PitchRatio: 1,
		Velocity:   110,
		Loop:       true,
	}

	if err := audio.Play(req); err != nil {
		t.Fatalf("Play err = %v", err)
	}
	if audio.channelVoices[0] != "midi-0-23" {
		t.Fatalf("channelVoices = %v, want channel 0 mapped", audio.channelVoices)
	}
	if audio.noteVoices[[2]int{0, 23}] != "midi-0-23" {
		t.Fatalf("noteVoices = %v, want note mapped", audio.noteVoices)
	}
	if got := audio.channelOrder[0]; len(got) != 1 || got[0] != "midi-0-23" {
		t.Fatalf("channelOrder = %v, want voice queued", audio.channelOrder)
	}
	if audio.channelActive[0] != "midi-0-23" {
		t.Fatalf("channelActive = %v, want active voice mapped", audio.channelActive)
	}
	audio.StopNote(0, 23)
	if len(audio.voices) != 0 {
		t.Fatalf("voices = %v, want note stop to clear voice", audio.voices)
	}
}

func TestAudioManagerPlayKeepsMultipleHeldNotesPerChannel(t *testing.T) {
	state := NewState(DefaultConfig())
	audio := &AudioManager{
		state:         state,
		cache:         map[string]*audioBuffer{"sample.mp3": {samples: []int16{1000, -1000, 2000, -2000}}},
		voices:        map[string]*audioVoice{},
		channelVoices: map[int]string{},
		channelOrder:  map[int][]string{},
		channelActive: map[int]string{},
		noteVoices:    map[[2]int]string{},
	}
	baseReq := PlaybackRequest{
		FilePath:   "sample.mp3",
		Channel:    0,
		EndSec:     0.00009,
		PitchRatio: 1,
		Velocity:   110,
		Loop:       true,
	}

	reqA := baseReq
	reqA.VoiceKey = "midi-0-23"
	reqA.Note = 23
	if err := audio.Play(&reqA); err != nil {
		t.Fatalf("Play A err = %v", err)
	}
	reqB := baseReq
	reqB.VoiceKey = "midi-0-24"
	reqB.Note = 24
	if err := audio.Play(&reqB); err != nil {
		t.Fatalf("Play B err = %v", err)
	}

	if len(audio.voices) != 2 {
		t.Fatalf("voices = %v, want both held notes", audio.voices)
	}
	if got := audio.channelOrder[0]; len(got) != 2 || got[0] != "midi-0-23" || got[1] != "midi-0-24" {
		t.Fatalf("channelOrder = %v, want both notes in order", audio.channelOrder)
	}
	if audio.channelActive[0] != "midi-0-24" {
		t.Fatalf("channelActive = %v, want latest note active", audio.channelActive)
	}
	if audio.voices["midi-0-23"].active {
		t.Fatalf("older held voice should be queued, not active")
	}
}

func TestAudioManagerCyclesHeldLoopingNotes(t *testing.T) {
	audio := &AudioManager{
		voices: map[string]*audioVoice{
			"midi-0-48": {pcm: []int16{1000}, loop: true, channel: 0, note: 48, active: true},
			"midi-0-60": {pcm: []int16{2000}, loop: true, channel: 0, note: 60},
		},
		channelVoices: map[int]string{0: "midi-0-48"},
		channelOrder:  map[int][]string{0: {"midi-0-48", "midi-0-60"}},
		channelActive: map[int]string{0: "midi-0-48"},
		noteVoices:    map[[2]int]string{{0, 48}: "midi-0-48", {0, 60}: "midi-0-60"},
	}
	out := make([]byte, 8)

	completed := audio.mixIntoLocked(out)
	if len(completed) != 0 {
		t.Fatalf("completed = %v, want none", completed)
	}
	got := []int16{
		int16(binary.LittleEndian.Uint16(out[0:2])),
		int16(binary.LittleEndian.Uint16(out[2:4])),
		int16(binary.LittleEndian.Uint16(out[4:6])),
		int16(binary.LittleEndian.Uint16(out[6:8])),
	}
	want := []int16{1000, 2000, 1000, 2000}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("samples = %v, want %v", got, want)
		}
	}
}
