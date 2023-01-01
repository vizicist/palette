package engine

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// CursorDeviceCallbackFunc xxx
type CursorCallbackFunc func(e CursorEvent)

// CursorEvent is a single Cursor event
type CursorEvent struct {
	Cid   string
	Click Clicks // absolute, I think
	// Source string
	Ddu  string // "down", "drag", "up" (sometimes "clear")
	X    float32
	Y    float32
	Z    float32
	Area float32
}

type CursorState struct {
	Current  CursorEvent
	Previous CursorEvent
	// Data     any // Plugins can use this to store data
}

type CursorManager struct {
	cursors      map[string]*CursorState
	cursorsMutex sync.RWMutex
}

func (ce CursorEvent) ToMap() map[string]string {
	return map[string]string{
		"event": "cursor",
		"cid":   ce.Cid,
		// "source": ce.Source,
		// Timestamp time.Time
		"ddu":  ce.Ddu,
		"x":    strconv.FormatFloat(float64(ce.X), 'f', 6, 32),
		"y":    strconv.FormatFloat(float64(ce.Y), 'f', 6, 32),
		"z":    strconv.FormatFloat(float64(ce.Z), 'f', 6, 32),
		"area": strconv.FormatFloat(float64(ce.Area), 'f', 6, 32),
	}
}

// Format xxx
func (ce CursorEvent) Format(f fmt.State, c rune) {
	s := fmt.Sprintf("(CursorEvent{%f,%f,%f})", ce.X, ce.Y, ce.Z)
	f.Write([]byte(s))
}

func (ce CursorEvent) Source() string {
	if ce.Cid == "" {
		LogWarn("CursorEvent.Source: empty cid", "ce", ce)
		return "dummysource"
	}
	arr := strings.Split(ce.Cid, "#")
	return arr[0]
}

func NewCursorManager() *CursorManager {
	return &CursorManager{
		cursors:      map[string]*CursorState{},
		cursorsMutex: sync.RWMutex{},
	}
}

func (cm *CursorManager) GetCursorState(cid string) *CursorState {

	cm.cursorsMutex.Lock()
	defer cm.cursorsMutex.Unlock()

	LogInfo("GetCursorState", "cid", cid, "ncursors", len(cm.cursors))
	cs, ok := cm.cursors[cid]
	if !ok {
		return nil
	}
	return cs
}

func (cm *CursorManager) clearCursors() {

	cm.cursorsMutex.RLock()

	currentClick := CurrentClick()
	cidsToDelete := []string{}
	for cid, cs := range cm.cursors {
		ce := cs.Current
		ce.Click = currentClick
		ce.Ddu = "up"

		cm.cursorsMutex.RUnlock()
		cm.HandleCursorEvent(ce)
		cm.cursorsMutex.RLock()

		DebugLogOfType("cursor", "Clearing cursor", "cid", cid)

		cidsToDelete = append(cidsToDelete, cid)
	}
	cm.cursorsMutex.RUnlock()

	cm.cursorsMutex.Lock()
	for _, cid := range cidsToDelete {
		delete(cm.cursors, cid)
	}
	cm.cursorsMutex.Unlock()
}

func (cm *CursorManager) HandleCursorEvent(ce CursorEvent) {

	switch ce.Ddu {

	case "clear":
		// Special event to clear cursors (by sending them "up" events)
		cm.clearCursors()

	case "down", "drag", "up":
		cm.handleDownDragUp(ce)
	}
}

func (cm *CursorManager) generateCursorGestureesture(cid string, noteDuration time.Duration, x0, y0, z0, x1, y1, z1 float32) {
	LogInfo("generateCursorGestureesture: start")

	ce := CursorEvent{
		Cid:   cid,
		Click: CurrentClick(),
		// Source: source,
		Ddu:  "down",
		X:    x0,
		Y:    y0,
		Z:    z0,
		Area: 0,
	}
	cm.HandleCursorEvent(ce)
	LogInfo("generateCursorGestureesture", "ddu", "down", "ce", ce)

	// secs := float32(3.0)
	secs := float32(noteDuration)
	dt := time.Duration(int(secs * float32(time.Second)))
	time.Sleep(dt)
	ce.Ddu = "up"
	ce.X = x1
	ce.Y = y1
	ce.Z = z1
	cm.HandleCursorEvent(ce)
	LogInfo("generateCursorGestureesture end", "ddu", "up", "ce", ce)
}

func (cm *CursorManager) getCursorFor(cid string) (*CursorState, bool) {
	cm.cursorsMutex.RLock()
	cursorState, ok := cm.cursors[cid]
	cm.cursorsMutex.RUnlock()
	return cursorState, ok
}

func (cm *CursorManager) handleDownDragUp(ce CursorEvent) {

	if ce.Click == 0 {
		ce.Click = CurrentClick()
	}

	cursorState, ok := cm.getCursorFor(ce.Cid)
	if !ok {
		// new CursorState
		// Make sure the first ddu is "down"
		if ce.Ddu != "down" {
			LogWarn("handleDownDragUp: first ddu is not down", "cid", ce.Cid, "ddu", ce.Ddu)
			ce.Ddu = "down"
		}
		cursorState = &CursorState{
			Current:  ce,
			Previous: ce,
		}
		cm.cursorsMutex.Lock()
		cm.cursors[ce.Cid] = cursorState
		cm.cursorsMutex.Unlock()
	} else {
		// existing CursorState
		cursorState.Previous = cursorState.Current
		cursorState.Current = ce
	}
	cursorState.Current.Click = CurrentClick()

	// See which layer wants this input, but don't hold the Lock
	ThePluginManager().HandleCursorEvent(ce)

	DebugLogOfType("cursor", "CursorManager.handleDownDragUp", "ce", ce)

	if ce.Ddu == "up" {
		DebugLogOfType("cursor", "handleDownDragUp up is deleting cid", "cid", ce.Cid, "ddu", ce.Ddu)

		cm.cursorsMutex.Lock()
		delete(cm.cursors, ce.Cid)
		cm.cursorsMutex.Unlock()
	}
}

func (cm *CursorManager) autoCursorUp(now time.Time) {

	// checkDelay is the number of CLicks that has to pass
	// before we decide a cursor is no longer present,
	// resulting in an "up" event.
	checkDelay := 8 * OneBeat

	cm.cursorsMutex.RLock()
	var cidsToDelete = []string{}
	for cid, cursorState := range cm.cursors {

		dclick := cursorState.Previous.Click - cursorState.Current.Click
		if dclick > checkDelay {
			ce := cursorState.Current
			ce.Ddu = "up"
			LogInfo("Layer.autoCursorUp: before handleDownDragUp", "cid", cid)

			cm.cursorsMutex.RUnlock()
			cm.handleDownDragUp(ce)
			cm.cursorsMutex.RUnlock()

			cidsToDelete = append(cidsToDelete, cid)
		}
	}
	cm.cursorsMutex.RUnlock()

	for _, cid := range cidsToDelete {
		cm.cursorsMutex.Lock()
		LogInfo("Layer.autoCursorUp: deleting cursor", "cid", cid)
		delete(cm.cursors, cid)
		cm.cursorsMutex.Unlock()
	}
}
