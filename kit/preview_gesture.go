package kit

import (
	"math"
	"sync"
	"time"
)

// Preview gestures go directly to the visual renderer, so they cannot make
// sound, enter cursor loops, or count as real input on the pads.
type previewGestureState struct {
	mutex       sync.Mutex
	enabled     bool
	inputSerial uint64
	run         *previewGestureRun
}

type previewGestureRun struct {
	cancel chan struct{}
	emit   func(CursorEvent)
	last   CursorEvent
	active bool
}

func (s *previewGestureState) isEnabled() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.enabled
}

func (s *previewGestureState) setEnabled(enabled bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.enabled = enabled
	if !enabled {
		s.stopLocked()
	}
}

// Capture a ticket before changing parameters. Input arriving during the load
// invalidates it, rather than starting a demo over a gesture already in progress.
func (s *previewGestureState) beginLoad() uint64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.stopLocked()
	return s.inputSerial
}

func (s *previewGestureState) realInput() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.inputSerial++
	s.stopLocked()
}

func (s *previewGestureState) stopLocked() {
	if s.run == nil {
		return
	}
	close(s.run.cancel)
	if s.run.active {
		up := s.run.last
		up.Ddu = "up"
		up.CClick = CurrentClick()
		up.Pos.Z = 0
		s.run.emit(up)
	}
	s.run = nil
}

func (s *previewGestureState) start(ticket uint64, patch string, nextGID func() int, emit func(CursorEvent)) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if !s.enabled || ticket != s.inputSerial {
		return
	}
	s.stopLocked()
	run := &previewGestureRun{cancel: make(chan struct{}), emit: emit}
	s.run = run
	go s.generate(run, patch, nextGID)
}

// Emission and cancellation share a lock: once realInput returns, an old
// goroutine cannot send another drag after the real gesture has taken over.
func (s *previewGestureState) emit(run *previewGestureRun, ce CursorEvent) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.run != run {
		return false
	}
	run.last = ce
	run.active = ce.Ddu != "up"
	run.emit(ce)
	return true
}

func (s *previewGestureState) generate(run *previewGestureRun, patch string, nextGID func() int) {
	ticker := time.NewTicker(time.Second / 30)
	defer ticker.Stop()
	for stroke := 0; ; stroke++ {
		gid := nextGID()
		for step := 0; step <= 60; step++ {
			if step > 0 {
				select {
				case <-run.cancel:
					return
				case <-ticker.C:
				}
			}
			t := 2*math.Pi*float64(step)/60 + float64(stroke)*math.Pi/3
			pos := CursorPos{X: .5 + .3*math.Sin(t), Y: .5 + .25*math.Sin(2*t), Z: .6 + .15*math.Sin(t)}
			ddu := "drag"
			if step == 0 {
				ddu = "down"
			} else if step == 60 {
				ddu = "up"
				pos.Z = 0
			}
			if !s.emit(run, NewCursorEvent(gid, patch+",preview", ddu, pos)) {
				return
			}
		}
		select {
		case <-run.cancel:
			return
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func (patch *Patch) startPreviewGesture(ticket uint64) {
	if theCursorManager == nil {
		return
	}
	logic := NewPatchLogic(patch)
	patch.preview.start(ticket, patch.Name(), theCursorManager.UniqueGID, func(ce CursorEvent) {
		if patch.GetBool("misc.generatevisual") {
			LogOfType("cursor", "Preview gesture", "patch", patch.Name(), "gid", ce.GID, "ddu", ce.Ddu)
			logic.generateVisualsFromCursor(transformCursorEvent(ce))
		}
	})
}

// Called only at real input ingress, not when replaying scheduled cursors.
func stopPreviewGestureForInput(ce CursorEvent) {
	if patch := GetPatch(ce.Source()); patch != nil {
		patch.preview.realInput()
	}
}
