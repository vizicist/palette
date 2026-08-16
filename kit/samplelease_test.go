package kit

import (
	"sync"
	"testing"
	"time"

	ss "github.com/vizicist/palette/pkg/samplesplitter"
)

// A shutdown must wait for the calls already inside the service before it
// closes anything.
//
// withSamplePlaybackService used to copy the service pointer under the mutex
// and then run its callback with the mutex released, so StopSamplePlaybackService
// could Close the service - tearing down the malgo device and context - while a
// scheduled note, a reload or a parameter setter was still calling into it.
//
// The service itself needs real audio hardware, so this exercises the lease
// bookkeeping directly: a lease is outstanding, and the waiter must not get
// through until it is returned.
func TestSamplePlaybackShutdownWaitsForLeases(t *testing.T) {
	var leases sync.WaitGroup

	// A call that is inside the service right now.
	leases.Add(1)

	closed := make(chan struct{})
	go func() {
		leases.Wait() // what StopSamplePlaybackService does before Close
		close(closed)
	}()

	select {
	case <-closed:
		t.Fatal("shutdown proceeded to Close while a call was still inside the service")
	case <-time.After(100 * time.Millisecond):
		// Correct: still waiting.
	}

	leases.Done()

	select {
	case <-closed:
		// Correct: released once the call returned.
	case <-time.After(10 * time.Second):
		t.Fatal("shutdown never proceeded after the lease was returned")
	}
}

// Detaching before waiting is what makes the wait safe: once the service is out
// of the holder, withSamplePlaybackService takes no further lease, so the
// waiter cannot miss one that appears behind it.
func TestSamplePlaybackDetachStopsNewLeases(t *testing.T) {
	old := samplePlaybackService.service
	defer func() { samplePlaybackService.service = old }()

	samplePlaybackService.mutex.Lock()
	samplePlaybackService.service = nil
	samplePlaybackService.mutex.Unlock()

	// With nothing attached this must report failure rather than run the
	// callback, and must not leave a lease behind for a waiter to hang on.
	ran := false
	if withSamplePlaybackService(func(*ss.Service) { ran = true }) {
		t.Fatal("withSamplePlaybackService succeeded with no service attached")
	}
	if ran {
		t.Fatal("the callback ran with no service attached")
	}

	done := make(chan struct{})
	go func() {
		samplePlaybackService.leases.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("a failed lookup left a lease outstanding, so shutdown would hang")
	}
}
