package samplesplitter

import (
	"errors"
	"reflect"
	"testing"
)

func TestAnalyzeFirstUsableMP3SkipsBelowWordThreshold(t *testing.T) {
	files := []MP3File{
		{Name: "quiet.mp3", Path: "quiet.mp3"},
		{Name: "words.mp3", Path: "words.mp3"},
	}
	var calls []string
	analyze := func(path string, opts AnalyzeOptions) (CueData, []float64, error) {
		calls = append(calls, path)
		if path == "quiet.mp3" {
			return CueData{}, nil, ErrBelowWordThreshold
		}
		return CueData{File: path, MaxRMS: 0.2}, []float64{0.1, 0.2}, nil
	}

	mp3, cue, waveform, err := analyzeFirstUsableMP3(files, analyze, AnalyzeOptions{WordThreshold: 0.01})
	if err != nil {
		t.Fatalf("analyzeFirstUsableMP3 err = %v", err)
	}
	if mp3.Path != "words.mp3" || cue.File != "words.mp3" {
		t.Fatalf("selected mp3=%q cue=%q, want words.mp3", mp3.Path, cue.File)
	}
	if !reflect.DeepEqual(waveform, []float64{0.1, 0.2}) {
		t.Fatalf("waveform = %v, want [0.1 0.2]", waveform)
	}
	if !reflect.DeepEqual(calls, []string{"quiet.mp3", "words.mp3"}) {
		t.Fatalf("calls = %v, want quiet then words", calls)
	}
}

func TestAnalyzeFirstUsableMP3ReturnsNonThresholdError(t *testing.T) {
	wantErr := errors.New("decode failed")
	files := []MP3File{{Name: "bad.mp3", Path: "bad.mp3"}}
	analyze := func(path string, opts AnalyzeOptions) (CueData, []float64, error) {
		return CueData{}, nil, wantErr
	}

	mp3, _, _, err := analyzeFirstUsableMP3(files, analyze, AnalyzeOptions{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	if mp3.Path != "bad.mp3" {
		t.Fatalf("mp3.Path = %q, want bad.mp3", mp3.Path)
	}
}
