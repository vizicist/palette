package samplesplitter

import (
	"path/filepath"
	"testing"
)

// The configured minimum keeps too-short files away from the splitter, which
// needs something long enough to carve into words. The sampleplayer plays
// files whole, so short one-shots must get through.
func TestWholeModeIgnoresMinimumDuration(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"short1.mp3", "short2.mp3"} {
		if err := writeTestMP3(filepath.Join(dir, name), 1); err != nil {
			t.Fatal(err)
		}
	}

	s := NewState(Config{MinimumMP3DurationSeconds: 10})
	s.SetChannelPlayback(0, WholeSplitMode, false)
	analyze := func(path string, opts AnalyzeOptions) (CueData, []float64, error) {
		return CueData{File: path, Splits: []float64{0}, Duration: 1}, nil, nil
	}

	paths, err := s.loadChannelRotation(0, dir, analyze)
	if err != nil {
		t.Fatalf("loadChannelRotation: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("loaded %d one-second files (%v), want both", len(paths), paths)
	}
}

func TestSplitModeStillEnforcesMinimumDuration(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"short1.mp3", "short2.mp3"} {
		if err := writeTestMP3(filepath.Join(dir, name), 1); err != nil {
			t.Fatal(err)
		}
	}

	s := NewState(Config{MinimumMP3DurationSeconds: 10})
	s.SetChannelPlayback(0, DefaultSplitMode, true)
	analyze := func(path string, opts AnalyzeOptions) (CueData, []float64, error) {
		return CueData{File: path, Splits: []float64{0}, Duration: 1}, nil, nil
	}

	if _, err := s.loadChannelRotation(0, dir, analyze); err == nil {
		t.Fatal("splitter accepted files below the configured minimum")
	}
}

func TestMinimumDurationForChannelIsPerChannel(t *testing.T) {
	s := NewState(Config{MinimumMP3DurationSeconds: 10})
	s.SetChannelPlayback(0, WholeSplitMode, false)
	s.SetChannelPlayback(1, DefaultSplitMode, true)

	if got := s.minimumDurationForChannel(0); got != 0 {
		t.Fatalf("sampleplayer channel minimum = %v, want 0", got)
	}
	if got := s.minimumDurationForChannel(1); got != 10 {
		t.Fatalf("samplesplitter channel minimum = %v, want 10", got)
	}
	// An unconfigured channel keeps the configured minimum.
	if got := s.minimumDurationForChannel(2); got != 10 {
		t.Fatalf("unconfigured channel minimum = %v, want 10", got)
	}
}

// wholeFileState mimics what the sampleplayer produces: one split at 0, so the
// whole file is the only thing a note can trigger.
func wholeFileState(duration float64) *State {
	s := NewState(Config{BaseNote: 48})
	cue := CueData{
		Splits:     []float64{0},
		PeakStarts: []float64{0},
		Duration:   duration,
		Mode:       WholeSplitMode,
	}
	s.ChannelSamples[0] = SampleState{
		Sigil:       "channel-0",
		CurrentFile: "goat1.mp3",
		CueData:     &cue,
	}
	return s
}

func TestWholeModePlaysEntireFileForAnyNote(t *testing.T) {
	s := wholeFileState(2.5)
	s.SetChannelPlayback(0, WholeSplitMode, false)

	// Cursor X spreads notes across 0..47; every one should play the whole file.
	for _, note := range []int{0, 1, 12, 23, 47, 48, 96} {
		req, err := s.PlanNoteOn(note, 100, 0)
		if err != nil {
			t.Fatalf("PlanNoteOn(note=%d): %v", note, err)
		}
		if req.StartSec != 0 || req.EndSec != 2.5 {
			t.Fatalf("note %d played %.3f-%.3f, want 0-2.5", note, req.StartSec, req.EndSec)
		}
		if req.SplitIndex != 0 {
			t.Fatalf("note %d split index = %d, want 0", note, req.SplitIndex)
		}
	}
}

func TestWholeModeDoesNotLoop(t *testing.T) {
	s := wholeFileState(1.0)
	s.SetChannelPlayback(0, WholeSplitMode, false)

	req, err := s.PlanNoteOn(48, 100, 0)
	if err != nil {
		t.Fatalf("PlanNoteOn: %v", err)
	}
	if req.Loop {
		t.Fatal("sampleplayer note is looping, want one-shot")
	}
}

func TestPeakStartDoesNotSeekPastStartInWholeMode(t *testing.T) {
	s := wholeFileState(2.0)
	s.Config.PeakStartEnabled = true
	s.SetChannelPlayback(0, WholeSplitMode, false)

	req, err := s.PlanNoteOn(48, 100, 0)
	if err != nil {
		t.Fatalf("PlanNoteOn: %v", err)
	}
	if req.StartSec != 0 {
		t.Fatalf("StartSec = %.3f with PeakStartEnabled, want 0", req.StartSec)
	}
}

func TestSplitterChannelStillLoopsByDefault(t *testing.T) {
	s := wholeFileState(1.0) // no SetChannelPlayback call at all

	req, err := s.PlanNoteOn(48, 100, 0)
	if err != nil {
		t.Fatalf("PlanNoteOn: %v", err)
	}
	if !req.Loop {
		t.Fatal("a channel with no recorded playback settings stopped looping")
	}
}

func TestSetChannelPlaybackLoopIsPerChannel(t *testing.T) {
	s := NewState(Config{BaseNote: 48})
	cue := CueData{Splits: []float64{0}, PeakStarts: []float64{0}, Duration: 1}
	for _, ch := range []int{0, 1} {
		s.ChannelSamples[ch] = SampleState{
			Sigil:       "channel",
			CurrentFile: "x.mp3",
			CueData:     &cue,
		}
	}
	// Patch A on the sampleplayer, patch B on the samplesplitter.
	s.SetChannelPlayback(0, WholeSplitMode, false)
	s.SetChannelPlayback(1, DefaultSplitMode, true)

	a, err := s.PlanNoteOn(48, 100, 0)
	if err != nil {
		t.Fatalf("PlanNoteOn(channel 0): %v", err)
	}
	b, err := s.PlanNoteOn(48, 100, 1)
	if err != nil {
		t.Fatalf("PlanNoteOn(channel 1): %v", err)
	}
	if a.Loop {
		t.Fatal("channel 0 (sampleplayer) should not loop")
	}
	if !b.Loop {
		t.Fatal("channel 1 (samplesplitter) should loop")
	}
}

func TestAnalyzeOptionsArePerChannel(t *testing.T) {
	s := NewState(Config{DefaultWords: 2})
	s.SetChannelPlayback(0, WholeSplitMode, false)
	s.SetChannelPlayback(1, DefaultSplitMode, true)

	if got := s.analyzeOptionsForChannel(0).Mode; got != WholeSplitMode {
		t.Fatalf("channel 0 mode = %q, want %q", got, WholeSplitMode)
	}
	if got := s.analyzeOptionsForChannel(1).Mode; got != DefaultSplitMode {
		t.Fatalf("channel 1 mode = %q, want %q", got, DefaultSplitMode)
	}
	// An unconfigured channel keeps the service-wide default.
	if got := s.analyzeOptionsForChannel(2).Mode; got != DefaultSplitMode {
		t.Fatalf("channel 2 mode = %q, want the default %q", got, DefaultSplitMode)
	}
}
