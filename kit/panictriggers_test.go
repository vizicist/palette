package kit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// An empty quad category must be an error, not a panic. rand % len(arr) with
// len 0 is an integer divide by zero, and attract mode calls this from the
// scheduler goroutine every PresetChangeInterval - so before the guard, a data
// directory with no quad presets took the whole realtime loop down with it.
func TestLoadQuadRandOnEmptyCategoryReturnsError(t *testing.T) {
	setupQuadThemeTest(t)

	// The directory has to exist and be empty. A missing one fails earlier, in
	// SavedFileList; it is the existing-but-empty case that reached the modulo.
	quadDir := filepath.Join(PaletteDataPath(), SavedDir(), "quad")
	if err := os.MkdirAll(quadDir, 0755); err != nil {
		t.Fatal(err)
	}

	quad := NewQuad()

	// Must not panic.
	name, err := quad.loadQuadRand("quad")

	if err == nil {
		t.Fatalf("loadQuadRand on an empty category returned %q and no error", name)
	}
	if !strings.Contains(err.Error(), "no presets") {
		t.Fatalf("unexpected error %v", err)
	}
}

// The MIDI-thru path looks a synth up by the name in global.midithrusynth,
// which nothing validates on set. It must get the dummy synth registered under
// "" rather than the nil a raw map lookup returns, because every branch that
// follows dereferences it - SendNoteToMidiOutput reaches synth.state through
// midiOutputEnabled, which is where the scheduler used to panic.
func TestGetSynthFallsBackForUnknownName(t *testing.T) {
	fallback, _, cleanup := setupSynthWatchdogTest(t)
	defer cleanup()

	oldSynths := Synths
	defer func() { Synths = oldSynths }()
	Synths = map[string]*Synth{"": fallback}

	got := GetSynth("no-such-synth-in-Synths.json")
	if got == nil {
		t.Fatal("GetSynth returned nil for an unknown name; the MIDI-thru path would nil-deref")
	}
	if got != fallback {
		t.Fatal(`GetSynth did not fall back to the synth registered under ""`)
	}

	// The call the scheduler makes on that synth must not panic.
	got.SendNoteToMidiOutput(NewNoteOn(got, 60, 100))
}
