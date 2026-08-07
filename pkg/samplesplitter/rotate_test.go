package samplesplitter

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"
)

// channelWithRotation builds a State whose channel 0 has n rotation candidates.
func channelWithRotation(n int) *State {
	s := NewState(Config{BaseNote: 0})
	samples := make([]SampleState, 0, n)
	for i := 0; i < n; i++ {
		cue := CueData{Splits: []float64{0}, Duration: 1}
		samples = append(samples, SampleState{
			Sigil:       "channel-0",
			CurrentFile: fmt.Sprintf("goat%d.mp3", i),
			CueData:     &cue,
		})
	}
	s.ChannelRotation[0] = samples
	s.ChannelSamples[0] = samples[0]
	s.ChannelRotate[0] = true
	return s
}

func TestPlanNoteOnRotatesAcrossDirectory(t *testing.T) {
	s := channelWithRotation(4)

	seen := map[string]int{}
	for i := 0; i < 200; i++ {
		req, err := s.PlanNoteOn(0, 100, 0)
		if err != nil {
			t.Fatalf("PlanNoteOn: %v", err)
		}
		seen[req.File]++
	}

	if len(seen) != 4 {
		t.Fatalf("played %d distinct files %v, want all 4", len(seen), seen)
	}
}

func TestPlanNoteOnRotationAvoidsImmediateRepeat(t *testing.T) {
	s := channelWithRotation(3)

	prev := ""
	for i := 0; i < 200; i++ {
		req, err := s.PlanNoteOn(0, 100, 0)
		if err != nil {
			t.Fatalf("PlanNoteOn: %v", err)
		}
		if req.File == prev {
			t.Fatalf("iteration %d repeated %q back to back", i, req.File)
		}
		prev = req.File
	}
}

func TestPlanNoteOnSingleRotationCandidateRepeats(t *testing.T) {
	s := channelWithRotation(1)

	for i := 0; i < 5; i++ {
		req, err := s.PlanNoteOn(0, 100, 0)
		if err != nil {
			t.Fatalf("PlanNoteOn: %v", err)
		}
		if req.File != "goat0.mp3" {
			t.Fatalf("file = %q, want goat0.mp3", req.File)
		}
	}
}

func TestPlanNoteOnWithoutRotateUsesChannelSample(t *testing.T) {
	s := channelWithRotation(4)
	s.SetChannelRotate(0, false)

	for i := 0; i < 20; i++ {
		req, err := s.PlanNoteOn(0, 100, 0)
		if err != nil {
			t.Fatalf("PlanNoteOn: %v", err)
		}
		if req.File != "goat0.mp3" {
			t.Fatalf("file = %q, want the channel's single sample goat0.mp3", req.File)
		}
	}
}

func TestSetChannelRotateOffDropsRotationSet(t *testing.T) {
	s := channelWithRotation(3)
	s.SetChannelRotate(0, false)

	if got := len(s.ChannelRotation[0]); got != 0 {
		t.Fatalf("rotation set still has %d entries after disabling", got)
	}
}

func TestClearChannelSampleClearsRotation(t *testing.T) {
	s := channelWithRotation(3)
	s.ClearChannelSample(0)

	if len(s.ChannelRotation[0]) != 0 || s.ChannelRotate[0] {
		t.Fatal("ClearChannelSample left rotation state behind")
	}
}

func TestChannelSampleInventoryListsRotationWithLoudness(t *testing.T) {
	s := NewState(Config{BaseNote: 0})
	for i, rms := range []float64{0.4, 0.002, 0.35} {
		cue := CueData{Splits: []float64{0}, Duration: 1, MaxRMS: rms, NumSplits: 1}
		s.ChannelRotation[0] = append(s.ChannelRotation[0], SampleState{
			CurrentFile: fmt.Sprintf("goat%d.mp3", i),
			CueData:     &cue,
		})
	}

	inv := s.ChannelSampleInventory(0)
	if len(inv) != 3 {
		t.Fatalf("inventory = %d entries, want 3", len(inv))
	}
	// The quiet one must be identifiable from the inventory alone.
	if inv[1].MaxRMS != 0.002 || inv[1].Path != "goat1.mp3" {
		t.Fatalf("entry 1 = %+v, want goat1.mp3 at 0.002", inv[1])
	}
}

func TestChannelSampleInventoryFallsBackToSingleSample(t *testing.T) {
	s := NewState(Config{BaseNote: 0})
	cue := CueData{Splits: []float64{0}, Duration: 2, MaxRMS: 0.5, NumSplits: 1}
	s.ChannelSamples[0] = SampleState{CurrentFile: "only.mp3", CueData: &cue}

	inv := s.ChannelSampleInventory(0)
	if len(inv) != 1 || inv[0].Path != "only.mp3" || inv[0].MaxRMS != 0.5 {
		t.Fatalf("inventory = %+v, want the single loaded sample", inv)
	}
}

func TestChannelSampleInventoryEmptyForUnknownChannel(t *testing.T) {
	s := NewState(Config{BaseNote: 0})
	if inv := s.ChannelSampleInventory(7); len(inv) != 0 {
		t.Fatalf("inventory = %+v, want empty", inv)
	}
}

func TestPlanNoteOnReportsSourceLoudness(t *testing.T) {
	s := NewState(Config{BaseNote: 0})
	cue := CueData{Splits: []float64{0}, Duration: 1, MaxRMS: 0.0031}
	s.ChannelSamples[0] = SampleState{CurrentFile: "quiet.mp3", CueData: &cue}

	req, err := s.PlanNoteOn(0, 100, 0)
	if err != nil {
		t.Fatalf("PlanNoteOn: %v", err)
	}
	if req.MaxRMS != 0.0031 {
		t.Fatalf("MaxRMS = %v, want 0.0031", req.MaxRMS)
	}
}

func TestLoadChannelRotationKeepsEveryAnalyzableFile(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.mp3", "b.mp3", "c.mp3"} {
		if err := writeTestMP3(filepath.Join(dir, name), 11); err != nil {
			t.Fatal(err)
		}
	}

	s := NewState(Config{MinimumMP3DurationSeconds: 10})
	analyze := func(path string, opts AnalyzeOptions) (CueData, []float64, error) {
		return CueData{File: path, Splits: []float64{0}, Duration: 1}, nil, nil
	}

	paths, err := s.loadChannelRotation(0, dir, analyze)
	if err != nil {
		t.Fatalf("loadChannelRotation: %v", err)
	}
	if len(paths) != 3 {
		t.Fatalf("preload paths = %d (%v), want 3", len(paths), paths)
	}
	if got := len(s.ChannelRotation[0]); got != 3 {
		t.Fatalf("rotation set = %d, want 3", got)
	}
}

// One unreadable MP3 shouldn't take the whole directory down with it.
func TestLoadChannelRotationSkipsUnanalyzableFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.mp3", "b.mp3", "c.mp3"} {
		if err := writeTestMP3(filepath.Join(dir, name), 11); err != nil {
			t.Fatal(err)
		}
	}

	s := NewState(Config{MinimumMP3DurationSeconds: 10})
	analyze := func(path string, opts AnalyzeOptions) (CueData, []float64, error) {
		if filepath.Base(path) == "b.mp3" {
			return CueData{}, nil, errors.New("decode failed")
		}
		return CueData{File: path, Splits: []float64{0}, Duration: 1}, nil, nil
	}

	paths, err := s.loadChannelRotation(0, dir, analyze)
	if err != nil {
		t.Fatalf("loadChannelRotation: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("preload paths = %d (%v), want 2 with the failing file skipped", len(paths), paths)
	}
	for _, p := range paths {
		if filepath.Base(p) == "b.mp3" {
			t.Fatal("the unanalyzable file made it into the rotation")
		}
	}
}

func TestLoadChannelRotationFailsWhenNothingAnalyzes(t *testing.T) {
	dir := t.TempDir()
	if err := writeTestMP3(filepath.Join(dir, "a.mp3"), 11); err != nil {
		t.Fatal(err)
	}

	s := NewState(Config{MinimumMP3DurationSeconds: 10})
	analyze := func(path string, opts AnalyzeOptions) (CueData, []float64, error) {
		return CueData{}, nil, errors.New("decode failed")
	}

	if _, err := s.loadChannelRotation(0, dir, analyze); err == nil {
		t.Fatal("expected an error when no file analyzes")
	}
}
