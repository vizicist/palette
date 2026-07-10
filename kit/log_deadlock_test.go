package kit

import (
	"testing"
	"time"
)

// Verifies that querying/setting an unrecognized logtype doesn't deadlock
// (Log* calls re-enter logMutex via appendExtraValues -> IsLogging).
func TestUnknownLogtypeDoesNotDeadlock(t *testing.T) {
	InitLog("test")
	done := make(chan struct{})
	go func() {
		IsLogging("definitely_not_a_logtype")
		SetLogTypes("also_bogus")
		close(done)
	}()
	select {
	case <-done:
		// ok
	case <-time.After(5 * time.Second):
		t.Fatal("deadlock: unknown logtype hung IsLogging/SetLogTypes")
	}
}

func TestSamplePlaybackLogtypeCanBeEnabled(t *testing.T) {
	InitLog("test")
	defer SetLogTypes("")

	SetLogTypes("sampleplayback")
	if !IsLogging("sampleplayback") {
		t.Fatal("sampleplayback logtype was not enabled")
	}
}
