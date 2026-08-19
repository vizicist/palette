package kit

import (
	"sort"
	"sync/atomic"
	"testing"
	"time"
)

// quadWithPatches builds a minimal Quad whose patches have a usable synth.
// A zero-value Synth is enough: SendANO clears its note tracking and then
// bails out because no MIDI output is configured.
func quadWithPatches(t *testing.T, names ...string) *Quad {
	t.Helper()
	InitLog("") // the silence paths log

	quad := &Quad{patch: map[string]*Patch{}}
	for _, name := range names {
		quad.patch[name] = &Patch{name: name, synth: &Synth{}}
	}
	return quad
}

// captureSampleStops swaps in a recorder for the sample-playback stop and
// returns a func giving the patches it was called for.
func captureSampleStops(t *testing.T) func() []string {
	t.Helper()
	old := stopSamplePlaybackForPatch
	t.Cleanup(func() { stopSamplePlaybackForPatch = old })

	var stopped []string
	stopSamplePlaybackForPatch = func(patch string, reason string) bool {
		stopped = append(stopped, patch)
		return true
	}
	return func() []string {
		out := append([]string(nil), stopped...)
		sort.Strings(out)
		return out
	}
}

// A patch routed to the sample player never touches a MIDI synth, so an
// all-notes-off that only sends ANO leaves its voices sounding forever.
func TestAllNotesOffStopsSamplePlaybackForEveryPatch(t *testing.T) {
	quad := quadWithPatches(t, "A", "B", "C", "D")
	stopped := captureSampleStops(t)

	quad.allNotesOff()

	got := stopped()
	want := []string{"A", "B", "C", "D"}
	if len(got) != len(want) {
		t.Fatalf("stopped sample playback for %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("stopped sample playback for %v, want %v", got, want)
		}
	}
}

// The "ANO" API is what the GUI calls when switching categories.
func TestQuadANOAPIStopsSamplePlayback(t *testing.T) {
	quad := quadWithPatches(t, "A", "B")
	stopped := captureSampleStops(t)

	if _, err := quad.API("ANO", map[string]string{}); err != nil {
		t.Fatalf("quad.API(ANO): %v", err)
	}

	if len(stopped()) != 2 {
		t.Fatalf("ANO stopped sample playback for %v, want both patches", stopped())
	}
}

// attractModeTest wires up the globals setAttractMode touches and returns the
// manager plus a way to see which patches had sample playback stopped.
func attractModeTest(t *testing.T, startOn bool, lastChange time.Time) (*AttractManager, func() []string) {
	t.Helper()
	oldQuad, oldPatchs, oldAttract := theQuad, Patchs, theAttractManager
	oldScheduler := theScheduler
	t.Cleanup(func() {
		theQuad, Patchs, theAttractManager = oldQuad, oldPatchs, oldAttract
		theScheduler = oldScheduler
	})

	quad := quadWithPatches(t, "A", "B")
	theQuad = quad
	Patchs = quad.patch
	// setAttractMode clears each patch's loop, which needs a scheduler.
	theScheduler = NewScheduler()
	stopped := captureSampleStops(t)

	am := &AttractManager{
		settings:              attractSettings{Enabled: true},
		attractModeIsOn:       &atomic.Bool{},
		lastAttractModeChange: lastChange,
	}
	am.attractModeIsOn.Store(startOn)
	theAttractManager = am
	return am, stopped
}

// Attract mode drives the pads with generated gestures, so leaving it must
// silence sample playback too, not just the synths.
func TestLeavingAttractModeStopsSamplePlayback(t *testing.T) {
	am, stopped := attractModeTest(t, true, time.Now())

	am.SetAttractMode(false)

	if len(stopped()) != 2 {
		t.Fatalf("leaving attract mode stopped sample playback for %v, want both patches",
			stopped())
	}
}

// A note left hanging as someone walks away would otherwise keep sounding
// underneath the attract screen.
func TestEnteringAttractModeStopsSamplePlayback(t *testing.T) {
	// Backdated so the throttle on turning attract mode on has elapsed.
	am, stopped := attractModeTest(t, false, time.Now().Add(-5*time.Second))

	am.SetAttractMode(true)

	if !am.AttractModeIsOn() {
		t.Fatal("attract mode did not turn on")
	}
	if len(stopped()) != 2 {
		t.Fatalf("entering attract mode stopped sample playback for %v, want both patches",
			stopped())
	}
}
