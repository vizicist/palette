package samplesplitter

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func countingAnalyze(calls *int) analyzeMP3Func {
	return func(path string, opts AnalyzeOptions) (CueData, []float64, error) {
		*calls++
		return CueData{File: path, Splits: []float64{0}, Duration: 1}, []float64{0.5}, nil
	}
}

func TestAnalysisCacheServesRepeatsWithoutReanalyzing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.mp3")
	if err := writeTestMP3(path, 11); err != nil {
		t.Fatal(err)
	}

	calls := 0
	analyze := newAnalysisCache().wrap(countingAnalyze(&calls))
	opts := AnalyzeOptions{Mode: DefaultSplitMode, WordsPerSplit: 2}

	for i := 0; i < 20; i++ {
		if _, _, err := analyze(path, opts); err != nil {
			t.Fatalf("analyze: %v", err)
		}
	}
	if calls != 1 {
		t.Fatalf("ran the analyzer %d times, want 1", calls)
	}
}

func TestAnalysisCacheReturnsTheSameResult(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.mp3")
	if err := writeTestMP3(path, 11); err != nil {
		t.Fatal(err)
	}

	calls := 0
	analyze := newAnalysisCache().wrap(countingAnalyze(&calls))
	opts := AnalyzeOptions{Mode: DefaultSplitMode}

	first, wave1, _ := analyze(path, opts)
	second, wave2, _ := analyze(path, opts)
	if first.File != second.File || first.Duration != second.Duration {
		t.Fatalf("cached cue %+v differs from original %+v", second, first)
	}
	if len(wave1) != len(wave2) {
		t.Fatal("cached waveform differs from the original")
	}
}

// A different split mode must not reuse the previous mode's cue.
func TestAnalysisCacheKeyedOnMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.mp3")
	if err := writeTestMP3(path, 11); err != nil {
		t.Fatal(err)
	}

	calls := 0
	analyze := newAnalysisCache().wrap(countingAnalyze(&calls))

	analyze(path, AnalyzeOptions{Mode: DefaultSplitMode})
	analyze(path, AnalyzeOptions{Mode: WholeSplitMode})
	analyze(path, AnalyzeOptions{Mode: DefaultSplitMode})
	if calls != 2 {
		t.Fatalf("ran the analyzer %d times, want 2 (one per mode)", calls)
	}
}

func TestAnalysisCacheInvalidatesWhenFileChanges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.mp3")
	if err := writeTestMP3(path, 11); err != nil {
		t.Fatal(err)
	}

	calls := 0
	analyze := newAnalysisCache().wrap(countingAnalyze(&calls))
	opts := AnalyzeOptions{Mode: DefaultSplitMode}

	analyze(path, opts)
	// Rewrite it longer, so both size and mtime change.
	if err := writeTestMP3(path, 20); err != nil {
		t.Fatal(err)
	}
	analyze(path, opts)

	if calls != 2 {
		t.Fatalf("ran the analyzer %d times, want 2 (the file changed)", calls)
	}
}

// A failed analysis might be transient, so it must not be remembered.
func TestAnalysisCacheDoesNotCacheErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.mp3")
	if err := writeTestMP3(path, 11); err != nil {
		t.Fatal(err)
	}

	calls := 0
	failing := func(p string, o AnalyzeOptions) (CueData, []float64, error) {
		calls++
		return CueData{}, nil, errors.New("decode failed")
	}
	analyze := newAnalysisCache().wrap(failing)

	analyze(path, AnalyzeOptions{})
	analyze(path, AnalyzeOptions{})
	if calls != 2 {
		t.Fatalf("ran the analyzer %d times, want 2 (errors are not cached)", calls)
	}
}

func TestAnalysisCacheFallsThroughForMissingFile(t *testing.T) {
	calls := 0
	analyze := newAnalysisCache().wrap(countingAnalyze(&calls))
	missing := filepath.Join(t.TempDir(), "nope.mp3")

	analyze(missing, AnalyzeOptions{})
	analyze(missing, AnalyzeOptions{})
	// No stat means no key, so it just passes through every time.
	if calls != 2 {
		t.Fatalf("ran the analyzer %d times, want 2", calls)
	}
}

func TestAnalysisCacheClear(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.mp3")
	if err := writeTestMP3(path, 11); err != nil {
		t.Fatal(err)
	}

	calls := 0
	cache := newAnalysisCache()
	analyze := cache.wrap(countingAnalyze(&calls))

	analyze(path, AnalyzeOptions{})
	cache.clear()
	analyze(path, AnalyzeOptions{})
	if calls != 2 {
		t.Fatalf("ran the analyzer %d times, want 2 after a clear", calls)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
