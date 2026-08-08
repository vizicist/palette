package kit

import "testing"

// resetSamplePlaybackSync clears the coalescing state between tests.
func resetSamplePlaybackSync() {
	samplePlaybackSync.mutex.Lock()
	samplePlaybackSync.suspended = 0
	samplePlaybackSync.pending = false
	samplePlaybackSync.mutex.Unlock()
}

func syncPending() bool {
	samplePlaybackSync.mutex.Lock()
	defer samplePlaybackSync.mutex.Unlock()
	return samplePlaybackSync.pending
}

func syncSuspended() int {
	samplePlaybackSync.mutex.Lock()
	defer samplePlaybackSync.mutex.Unlock()
	return samplePlaybackSync.suspended
}

func TestSamplePlaybackSyncCoalescesWhileSuspended(t *testing.T) {
	resetSamplePlaybackSync()
	defer resetSamplePlaybackSync()

	resume := SuspendSamplePlaybackSync()
	// A quad load asks for a resync once per sample parameter per patch.
	for i := 0; i < 20; i++ {
		if err := RequestSamplePlaybackSync(); err != nil {
			t.Fatalf("RequestSamplePlaybackSync: %v", err)
		}
	}
	if !syncPending() {
		t.Fatal("requests while suspended did not record a pending resync")
	}
	resume()

	if syncSuspended() != 0 {
		t.Fatalf("suspend count = %d after resume, want 0", syncSuspended())
	}
	if syncPending() {
		t.Fatal("pending resync survived the resume")
	}
}

// quad.Load suspends, then each patch.load suspends again - only the outermost
// resume may flush.
func TestSamplePlaybackSyncNests(t *testing.T) {
	resetSamplePlaybackSync()
	defer resetSamplePlaybackSync()

	outer := SuspendSamplePlaybackSync()
	inner := SuspendSamplePlaybackSync()
	if syncSuspended() != 2 {
		t.Fatalf("suspend count = %d, want 2", syncSuspended())
	}

	_ = RequestSamplePlaybackSync()
	inner()
	if syncSuspended() != 1 {
		t.Fatalf("suspend count = %d after inner resume, want 1", syncSuspended())
	}
	if !syncPending() {
		t.Fatal("the inner resume consumed the pending resync; only the outermost should")
	}

	outer()
	if syncPending() || syncSuspended() != 0 {
		t.Fatalf("after outer resume: pending=%v suspended=%d, want false/0",
			syncPending(), syncSuspended())
	}
}

func TestSamplePlaybackSyncResumeWithNoRequestsDoesNothing(t *testing.T) {
	resetSamplePlaybackSync()
	defer resetSamplePlaybackSync()

	resume := SuspendSamplePlaybackSync()
	resume()
	if syncPending() {
		t.Fatal("a resume with no requests left a pending resync")
	}
}
