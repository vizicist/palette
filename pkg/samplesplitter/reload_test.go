package samplesplitter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A reload of a directory with nothing usable in it must leave the samples that
// are already loaded alone, and say so.
//
// Reload used to stop audio and clear the decoded cache before finding out
// whether the directory held anything, so an empty or below-threshold directory
// tore down what was playing and left the service silent - while still
// reporting audio healthy, because preloading an empty list returns no error.
func TestReloadRefusesAnEmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	state := NewState(Config{MP3Dir: dir, MinimumMP3DurationSeconds: 1})

	// Something is "already loaded".
	state.mu.Lock()
	state.CurrentFile = "previously-loaded.mp3"
	state.mu.Unlock()

	err := ReloadSigilSamples(state, Analyzer{}, nil, nil)
	if err == nil {
		t.Fatal("reloading an empty directory reported success")
	}
	if !strings.Contains(err.Error(), "left alone") {
		t.Errorf("the error does not say the loaded samples were kept: %v", err)
	}
	if got := state.Snapshot().CurrentFile; got != "previously-loaded.mp3" {
		t.Errorf("the previously loaded sample was discarded (CurrentFile = %q)", got)
	}
}

// A directory that cannot be read is also a refusal, not a teardown.
func TestReloadRefusesAnUnreadableDirectory(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "not-there")
	state := NewState(Config{MP3Dir: missing, MinimumMP3DurationSeconds: 1})

	if err := ReloadSigilSamples(state, Analyzer{}, nil, nil); err == nil {
		t.Fatal("reloading a missing directory reported success")
	}
}

// Files below the minimum duration do not count as usable, which is the case
// that used to commit error-only state and still claim to be healthy.
func TestReloadRefusesWhenEverythingIsBelowThreshold(t *testing.T) {
	dir := t.TempDir()
	// writeTestMP3 makes a file whose duration is derived from its size.
	if err := writeTestMP3(filepath.Join(dir, "tiny.mp3"), 1); err != nil {
		t.Fatal(err)
	}
	state := NewState(Config{MP3Dir: dir, MinimumMP3DurationSeconds: 3600})

	err := ReloadSigilSamples(state, Analyzer{}, nil, nil)
	if err == nil {
		t.Fatal("reload accepted a directory where nothing met the minimum duration")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "tiny.mp3")); statErr != nil {
		t.Fatal("the test file vanished")
	}
}
