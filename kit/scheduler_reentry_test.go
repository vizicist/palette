package kit

import (
	"testing"
	"time"
)

// deleteScheduledEvents must run its onDelete callbacks with sched.mutex
// released.
//
// The callbacks re-enter the scheduler for real: DeleteEventsWithTag and
// FilterEventsWithTag delete active cursors, and DeleteActiveCursor ->
// cleanupDeletedActiveCursor -> stopActiveSamplePlayback -> stopSamplePlayback
// calls DeleteSamplePlaybackStarts, which locks the same mutex. sync.RWMutex is
// not reentrant, so doing that under the lock wedged the goroutine - and
// DeleteEventsWithTag runs on the scheduler goroutine, which takes the clock,
// all scheduled notes, cursor handling and the hung-note watchdog with it.
func TestDeleteScheduledEventsRunsCallbacksUnlocked(t *testing.T) {
	InitLog("")

	oldScheduler := theScheduler
	defer func() { theScheduler = oldScheduler }()
	sched := NewScheduler()
	theScheduler = sched

	for i := 0; i < 3; i++ {
		sched.schedList.PushBack(NewSchedElement(Clicks(i), "A", &NoteOff{}))
	}

	done := make(chan int, 1)
	go func() {
		// The callback does exactly what the real ones end up doing: reach back
		// into the scheduler while it is deleting.
		done <- sched.deleteScheduledEvents(
			func(se *SchedElement) bool { return se.Tag == "A" },
			func(se *SchedElement) {
				sched.DeleteSamplePlaybackStarts("A", 1)
			})
	}()

	select {
	case n := <-done:
		if n != 3 {
			t.Fatalf("deleted %d events, want 3", n)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("deadlocked: onDelete re-entered the scheduler while its mutex was held")
	}

	// Everything matching really is gone, not merely unlocked.
	if got := sched.schedList.Len(); got != 0 {
		t.Fatalf("%d events left in the list, want 0", got)
	}
}

// Non-matching events survive, and the callback only sees what was removed.
func TestDeleteScheduledEventsOnlyTouchesMatches(t *testing.T) {
	InitLog("")

	sched := NewScheduler()
	sched.schedList.PushBack(NewSchedElement(1, "A", &NoteOff{}))
	sched.schedList.PushBack(NewSchedElement(2, "B", &NoteOff{}))
	sched.schedList.PushBack(NewSchedElement(3, "A", &NoteOff{}))

	seen := []string{}
	n := sched.deleteScheduledEvents(
		func(se *SchedElement) bool { return se.Tag == "A" },
		func(se *SchedElement) { seen = append(seen, se.Tag) })

	if n != 2 || len(seen) != 2 {
		t.Fatalf("deleted %d and called back %d times, want 2 and 2", n, len(seen))
	}
	if sched.schedList.Len() != 1 {
		t.Fatalf("%d events left, want the one tagged B", sched.schedList.Len())
	}
	if se := sched.schedList.Front().Value.(*SchedElement); se.Tag != "B" {
		t.Fatalf("survivor is tagged %q, want B", se.Tag)
	}
}
