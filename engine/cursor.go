package engine

import (
	"fmt"
	"sync"
	"time"
)

// CursorDeviceCallbackFunc xxx
type CursorCallbackFunc func(e CursorEvent)

// CursorEvent is a single Cursor event
type CursorEvent struct {
	ID        string
	Source    string
	Timestamp time.Time
	Ddu       string // "down", "drag", "up" (sometimes "clear")
	X         float32
	Y         float32
	Z         float32
	Area      float32
	internal  bool
}

type CursorManager struct {
	cursors      map[string]*DeviceCursor
	cursorsMutex sync.RWMutex
}

// Format xxx
func (ce CursorEvent) Format(f fmt.State, c rune) {
	s := fmt.Sprintf("(CursorEvent{%f,%f,%f})", ce.X, ce.Y, ce.Z)
	f.Write([]byte(s))
}

func NewCursorManager() *CursorManager {
	return &CursorManager{
		cursors:      map[string]*DeviceCursor{},
		cursorsMutex: sync.RWMutex{},
	}
}

func (cm *CursorManager) clearCursors() {

	cm.cursorsMutex.Lock()
	oldCursors := cm.cursors
	cm.cursors = map[string]*DeviceCursor{}
	cm.cursorsMutex.Unlock()

	for id, c := range oldCursors {
		if !c.downed {
			Warn("Hmmm, why is a cursor not downed?")
		} else {
			cm.handleCursorEvent(CursorEvent{Source: id, Ddu: "up"})
			DebugLogOfType("cursor", "Clearing cursor", "id", id)
			delete(oldCursors, id)
		}
	}
}

func (cm *CursorManager) handleCursorEvent(ce CursorEvent) {

	// As soon as there's any non-internal cursor event,
	// we turn attract mode off.
	if !ce.internal {
		go func() {
			TheEngine().Scheduler.Control <- Command{"attractmode", false}
		}()
	}

	DebugLogOfType("cursor", "CursorManager.handleCursorEvent", "id", ce.ID, "ddu", ce.Ddu, "x", ce.X, "y", ce.Y, "z", ce.Z)

	switch ce.Ddu {

	case "clear":
		// Special event to clear cursors (by sending them "up" events)
		// XXX - should this go in cursor.go?  yes.
		cm.clearCursors()
		return

	case "down", "drag", "up":

		cm.cursorsMutex.RLock()
		c, ok := cm.cursors[ce.ID]
		cm.cursorsMutex.RUnlock()

		if !ok {
			// new DeviceCursor
			c = &DeviceCursor{}
			cm.cursorsMutex.Lock()
			cm.cursors[ce.ID] = c
			cm.cursorsMutex.Unlock()
		}
		c.lastTouch = time.Now()

		// If it's a new (downed==false) cur"sor, make sure the first event is "down"
		if !c.downed {
			ce.Ddu = "down"
			c.downed = true
		}
		if ce.Ddu == "up" {
			cm.cursorsMutex.Lock()
			delete(cm.cursors, ce.ID)
			cm.cursorsMutex.Unlock()
		}
	}
}

func (cm *CursorManager) doCursorGesture(source string, cid string, x0, y0, z0, x1, y1, z1 float32) {

	ce := CursorEvent{
		Source:    source,
		Timestamp: time.Now(),
		Ddu:       "down",
		X:         x0,
		Y:         y0,
		Z:         z0,
		Area:      0,
		internal:  true,
	}

	DebugLogOfType("cursor", "doCursorGesture start", "ce", ce)

	cm.handleCursorEvent(ce)

	time.Sleep(TheEngine().Scheduler.attractNoteDuration)
	ce.Ddu = "up"
	ce.X = x1
	ce.Y = y1
	ce.Z = z1

	cm.handleCursorEvent(ce)

	DebugLogOfType("cursor", "doCursorGesture end", "ce", ce)
}

func (cm *CursorManager) checkCursorUp(now time.Time) {

	if checkDelay == 0 {
		milli := ConfigIntWithDefault("upcheckmillisecs", 1000)
		checkDelay = time.Duration(milli) * time.Millisecond
	}

	cm.cursorsMutex.RLock()
	oldCursors := cm.cursors
	cm.cursorsMutex.RUnlock()

	for id, c := range oldCursors {
		elapsed := now.Sub(c.lastTouch)
		if elapsed > checkDelay {
			cm.handleCursorEvent(CursorEvent{Source: "checkCursorUp", Ddu: "up"})
			Info("Player.checkCursorUp: deleting cursor", "id", id, "elapsed", elapsed)
			delete(oldCursors, id)
		}
	}
}
