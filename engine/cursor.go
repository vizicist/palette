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
}

type CursorManager struct {
	cursors            map[string]*DeviceCursor
	cursorsMutex       sync.RWMutex
	lastAttractCommand time.Time
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
	defer cm.cursorsMutex.Unlock()

	for id, c := range cm.cursors {
		if !c.downed {
			Warn("Hmmm, why is a cursor not downed?")
		} else {
			cm.handleCursorEvent(CursorEvent{Source: id, Ddu: "up"})
			DebugLogOfType("cursor", "Clearing cursor", "id", id)
			delete(cm.cursors, id)
		}
	}
}

func (cm *CursorManager) handleCursorEvent(ce CursorEvent) {

	// As soon as there's any non-internal cursor event,
	// we turn attract mode off.
	if ce.Source != "internal" {
		if time.Since(cm.lastAttractCommand) > time.Second {
			go func() {
				TheEngine().Scheduler.cmdInput <- AttractModeCmd{false}
			}()
			cm.lastAttractCommand = time.Now()
		}
	}

	switch ce.Ddu {

	case "clear":
		// Special event to clear cursors (by sending them "up" events)
		cm.clearCursors()

	case "down", "drag", "up":
		cm.handleDownDragUp(ce)
	}
}

func (cm *CursorManager) generateCursorGestureesture(source string, cid string, x0, y0, z0, x1, y1, z1 float32) {
	Info("generateCursorGestureesture: start")

	ce := CursorEvent{
		Source:    source,
		Timestamp: time.Now(),
		Ddu:       "down",
		X:         x0,
		Y:         y0,
		Z:         z0,
		Area:      0,
	}
	cm.handleCursorEvent(ce)
	Info("generateCursorGestureesture", "ddu", "down", "ce", ce)

	// secs := float32(3.0)
	secs := float32(TheEngine().Scheduler.attractNoteDuration)
	dt := time.Duration(int(secs * float32(time.Second)))
	time.Sleep(dt)
	ce.Ddu = "up"
	ce.X = x1
	ce.Y = y1
	ce.Z = z1
	cm.handleCursorEvent(ce)
	Info("generateCursorGestureesture end", "ddu", "up", "ce", ce)
}

func (cm *CursorManager) handleDownDragUp(ce CursorEvent) {

	cm.cursorsMutex.Lock()
	defer func() {
		cm.cursorsMutex.Unlock()
	}()

	c, ok := cm.cursors[ce.ID]
	if !ok {
		// new DeviceCursor
		c = &DeviceCursor{}
		cm.cursors[ce.ID] = c
	}
	c.lastTouch = time.Now()

	// If it's a new (downed==false) cur"sor, make sure the first event is "down"
	if !c.downed {
		ce.Ddu = "down"
		c.downed = true
	}

	// See which player wants this input
	TheRouter().PlayerManager.handleCursorEvent(ce)

	DebugLogOfType("cursor", "CursorManager.handleDownDragUp", "id", ce.ID, "ddu", ce.Ddu, "x", ce.X, "y", ce.Y, "z", ce.Z)
	if ce.Ddu == "up" {
		delete(cm.cursors, ce.ID)
	}
}

func (cm *CursorManager) autoCursorUp(now time.Time) {

	if checkDelay == 0 {
		milli := ConfigIntWithDefault("upcheckmillisecs", 1000)
		checkDelay = time.Duration(milli) * time.Millisecond
	}

	cm.cursorsMutex.Lock()
	defer cm.cursorsMutex.Unlock()

	for id, c := range cm.cursors {
		elapsed := now.Sub(c.lastTouch)
		if elapsed > checkDelay {
			cm.handleCursorEvent(CursorEvent{Source: "checkCursorUp", Ddu: "up"})
			Info("Player.checkCursorUp: deleting cursor", "id", id, "elapsed", elapsed)
			delete(cm.cursors, id)
		}
	}
}