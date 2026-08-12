package samplesplitter

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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

// blockingAnalyze returns an analyzer that reports each entry on started and
// then waits for release, so a test can hold one analysis open while other
// callers pile up behind it.
func blockingAnalyze(calls *atomic.Int64, started chan<- struct{}, release <-chan struct{}, err error) analyzeMP3Func {
	return func(path string, opts AnalyzeOptions) (CueData, []float64, error) {
		calls.Add(1)
		started <- struct{}{}
		<-release
		if err != nil {
			return CueData{}, nil, err
		}
		return CueData{File: path, Splits: []float64{0}, Duration: 1}, []float64{0.5}, nil
	}
}

// A burst of concurrent requests for one file must run the analyzer once, not
// once per caller. Analyzing spawns an ffmpeg and the cache lock can't be held
// across it, so without single-flight the whole burst misses together - which is
// exactly the shape a preset load has.
//
// The assertion is the absence of a second call while the first is still
// running, so the extra callers are given a window to produce one. Without the
// coalescing they reach the analyzer as soon as they are scheduled, which the
// WaitGroup has already waited for.
func TestAnalysisCacheSingleFlight(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.mp3")
	if err := writeTestMP3(path, 11); err != nil {
		t.Fatal(err)
	}

	var calls atomic.Int64
	started := make(chan struct{}, 8)
	release := make(chan struct{})
	analyze := newAnalysisCache().wrap(blockingAnalyze(&calls, started, release, nil))
	opts := AnalyzeOptions{Mode: DefaultSplitMode}

	// One caller gets into the analyzer and stays there.
	results := make(chan CueData, 8)
	go func() {
		cue, _, err := analyze(path, opts)
		if err != nil {
			t.Error(err)
		}
		results <- cue
	}()
	<-started

	const extra = 7
	var running sync.WaitGroup
	running.Add(extra)
	for i := 0; i < extra; i++ {
		go func() {
			running.Done() // scheduled, and about to ask for the same file
			cue, _, err := analyze(path, opts)
			if err != nil {
				t.Error(err)
			}
			results <- cue
		}()
	}
	running.Wait()

	deadline := time.Now().Add(250 * time.Millisecond)
	for time.Now().Before(deadline) {
		if n := calls.Load(); n > 1 {
			t.Fatalf("%d callers for one file started %d analyses, want 1", extra+1, n)
		}
		time.Sleep(5 * time.Millisecond)
	}

	close(release)
	for i := 0; i < extra+1; i++ {
		if cue := <-results; cue.File != path {
			t.Fatalf("caller %d got %q, want %q", i, cue.File, path)
		}
	}
	if n := calls.Load(); n != 1 {
		t.Fatalf("ran the analyzer %d times for one file, want 1", n)
	}
}

// A failed analysis must not be cached, but the callers waiting on it still
// have to be handed the error rather than left hanging.
func TestAnalysisCacheSingleFlightSharesErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.mp3")
	if err := writeTestMP3(path, 11); err != nil {
		t.Fatal(err)
	}

	var calls atomic.Int64
	started := make(chan struct{}, 4)
	release := make(chan struct{})
	boom := errors.New("analysis failed")
	analyze := newAnalysisCache().wrap(blockingAnalyze(&calls, started, release, boom))
	opts := AnalyzeOptions{Mode: DefaultSplitMode}

	errs := make(chan error, 4)
	go func() { _, _, err := analyze(path, opts); errs <- err }()
	<-started

	const extra = 3
	var running sync.WaitGroup
	running.Add(extra)
	for i := 0; i < extra; i++ {
		go func() {
			running.Done()
			_, _, err := analyze(path, opts)
			errs <- err
		}()
	}
	running.Wait()
	time.Sleep(100 * time.Millisecond) // let any duplicate analysis show itself

	close(release)
	for i := 0; i < extra+1; i++ {
		if err := <-errs; !errors.Is(err, boom) {
			t.Fatalf("caller %d got %v, want %v", i, err, boom)
		}
	}
	if n := calls.Load(); n != 1 {
		t.Fatalf("ran the analyzer %d times, want 1", n)
	}

	// The failure wasn't cached, so the next call tries again. release is
	// already closed, so this one doesn't block.
	if _, _, err := analyze(path, opts); !errors.Is(err, boom) {
		t.Fatalf("after the burst, got %v, want %v", err, boom)
	}
	if n := calls.Load(); n != 2 {
		t.Fatalf("ran the analyzer %d times, want 2 - a failure must not be cached", n)
	}
}
