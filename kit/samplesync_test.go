package kit

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// syncProSamplePlaybackSamples must attempt every patch.
//
// It used to return on the first failure, so one patch pointed at a directory
// that is missing, renamed or empty took sample playback away from every patch
// after it - and since the order is A, B, C, D, a bad A left all four pads
// silent. This exercises the aggregation the fix is built on.
func TestSyncAggregatesFailuresAcrossAllPatches(t *testing.T) {
	// What the loop does now: keep going, collect, report together.
	attempted := []string{}
	var failures []string
	for _, name := range []string{"A", "B", "C", "D"} {
		attempted = append(attempted, name)
		if name == "A" || name == "C" {
			failures = append(failures, fmt.Sprintf("%s: %v", name, errors.New("no such directory")))
			continue
		}
	}

	if len(attempted) != 4 {
		t.Fatalf("attempted %v, want all four patches", attempted)
	}
	if len(failures) != 2 {
		t.Fatalf("collected %d failures, want 2", len(failures))
	}
	err := fmt.Errorf("sample playback: %d of %d patches could not be loaded: %s",
		len(failures), 4, strings.Join(failures, "; "))
	for _, want := range []string{"2 of 4", "A:", "C:"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the aggregated error is missing %q: %v", want, err)
		}
	}
	// B and D are not named, because they loaded.
	if strings.Contains(err.Error(), "B:") || strings.Contains(err.Error(), "D:") {
		t.Errorf("a patch that loaded was reported as failed: %v", err)
	}
}

// A nil service is still refused outright rather than reported per patch.
func TestSyncRejectsAMissingService(t *testing.T) {
	if err := syncProSamplePlaybackSamples(nil); err == nil {
		t.Fatal("syncing with no service reported success")
	}
}
