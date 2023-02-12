package engine

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hypebeast/go-osc/osc"
)

var TheCursorManager *CursorManager

// CursorEvent is a singl Cursor event
type CursorEvent struct {
	Cid   string
	Click Clicks // XXX - I think this can be removed
	// Source string
	Ddu  string // "down", "drag", "up" (sometimes "clear")
	X    float32
	Y    float32
	Z    float32
	Area float32
}

type ActiveCursor struct {
	Current     CursorEvent
	Previous    CursorEvent
	NoteOn      *NoteOn
	NoteOnClick Clicks
}

type CursorManager struct {
	activeCursors  map[string]*ActiveCursor
	cursorsMutex   sync.RWMutex
	CidToLoopedCid map[string]string
	handlers       map[string]CursorHandler
	sequenceNum int
}

// CursorDeviceCallbackFunc xxx
type CursorCallbackFunc func(e CursorEvent)

type CursorHandler interface {
	onCursorEvent(state ActiveCursor) error
}

func NewActiveCursor(ce CursorEvent) *ActiveCursor {
	return &ActiveCursor{
		Current:  ce,
		Previous: ce,
	}
}

// Format xxx
func (ce CursorEvent) Format(f fmt.State, c rune) {
	s := fmt.Sprintf("(CursorEvent{%f,%f,%f})", ce.X, ce.Y, ce.Z)
	f.Write([]byte(s))
}

func (ce CursorEvent) IsInternal() bool {
	return strings.Contains(ce.Cid, "internal")
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
		activeCursors:  map[string]*ActiveCursor{},
		CidToLoopedCid: map[string]string{},
		cursorsMutex:   sync.RWMutex{},
		handlers:       map[string]CursorHandler{},
	}
}

func (cm *CursorManager) GetActiveCursor(cid string) *ActiveCursor {

	cm.cursorsMutex.Lock()
	defer cm.cursorsMutex.Unlock()

	cs, ok := cm.activeCursors[cid]
	if !ok {
		return nil
	}
	return cs
}

func (cm *CursorManager) AddCursorHandler(name string, handler CursorHandler, sources ...string) {
	cm.handlers[name] = handler
}

func (cm *CursorManager) ExecuteCursorEvent(ce CursorEvent) {

	switch ce.Ddu {

	case "clear":
		LogWarn("Hey, should CursorManager be seeing a clear CursorEvent?")
		// Special event to clear cursors (by sending them "up" events)
		cm.clearCursors()

	case "down", "drag", "up":
		cm.executeDownDragUp(ce)
	}
}

// cleaerCursors sends "up" events to all active cursors, then deletes them
func (cm *CursorManager) clearCursors() {

	cm.cursorsMutex.RLock()

	currentClick := CurrentClick()
	cidsToDelete := []string{}
	for cid, ac := range cm.activeCursors {
		ce := ac.Current
		ce.Click = currentClick
		ce.Ddu = "up"

		cm.cursorsMutex.RUnlock()
		cm.ExecuteCursorEvent(ce)
		cm.cursorsMutex.RLock()

		LogOfType("cursor", "Clearing cursor", "cid", cid)

		cidsToDelete = append(cidsToDelete, cid)
	}
	cm.cursorsMutex.RUnlock()

	cm.deleteActiveCursors(cidsToDelete)
}

func (cm *CursorManager) GenerateCursorGesture(cid string, noteDuration time.Duration, x0, y0, z0, x1, y1, z1 float32) {

	LogOfType("cursor", "generateCursorGesture start", "cid", cid, "noteDuration", noteDuration, "x0", x0, "y0", y0, "z0", z0, "x1", x1, "y1", y1, "z1", z1)
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
	LogWarn("Should generateCursorGesture be using ScheduleCursorEvent or ExecuteCursorEvent?")
	cm.ExecuteCursorEvent(ce)
	time.Sleep(noteDuration)
	ce.Ddu = "up"
	ce.X = x1
	ce.Y = y1
	ce.Z = z1
	cm.ExecuteCursorEvent(ce)
	LogOfType("cursor", "generateCursorGesture end", "cid", cid, "noteDuration", noteDuration, "x0", x0, "y0", y0, "z0", z0, "x1", x1, "y1", y1, "z1", z1)
}

func (cm *CursorManager) getActiveCursorFor(cid string) (*ActiveCursor, bool) {
	cm.cursorsMutex.RLock()
	ac, ok := cm.activeCursors[cid]
	cm.cursorsMutex.RUnlock()
	return ac, ok
}

func (cm *CursorManager) UniqueCid(baseCid string) string{
	base := baseCid
	arr := strings.Split(baseCid, "#")
	if len(arr) == 2 {
		base = arr[0]
	}
	// Return a unique cursor id, not already in cm.ActiveCursors
	for {
		cm.sequenceNum++
		newCid := fmt.Sprintf("%s#%d",base, cm.sequenceNum)
		_, ok := cm.activeCursors[newCid]
		if !ok {
			// Cool, found a unique cid
			LogOfType("cursor", "UniqueCid: returning", "baseCid", baseCid, "newCid", newCid)
			return newCid
		}
	}
}

/*
Keep track of unique names attached to each cursor "down",
and use those names on subsequent "drag" and "up" events.
*/
func (cm *CursorManager) LoopedCidFor(ce CursorEvent) string {

	loopedCid, ok := cm.CidToLoopedCid[ce.Cid]
	if !ok {
		// New entries should always be triggered by a "down" event
		if ce.Ddu != "down" {
			LogInfo("LoopedCidFor: first ddu is not down", "ce", ce)
			return ""
		}
		// LogWarn("adjustLoopedCid: oldCid not found in ActiveCursor", "oldCid", oldCid)
		loopedCid = cm.UniqueCid(ce.Cid)
		LogOfType("cursor","LoopedCidFor: adding new entry", "ce", ce, "loopedCid", loopedCid)
		cm.CidToLoopedCid[ce.Cid] = loopedCid
	}
	return loopedCid
}

/*
	newCid := fmt.Sprintf("%s#%d", oldCid[:i], oldNum+1)
	// Don't really need to copy the oldCursorEvent,
	// just want to make things clearer until it actually works.
	newCursorEvent := oldCursorEvent
	newCursorEvent.Cid = newCid
	state, ok := cm.activeCursors[newCid]
	if !ok {
		// If the newCid doesn't already have a ActiveCursor,
		// that means it should be a "down" event.
		if newCursorEvent.Ddu != "down" {
			LogWarn("ManageLoopingCid: first ddu is not down", "newCid", newCid, "ddu", newCursorEvent.Ddu)
			return nil
		}
		cm.activeCursors[newCid] = NewActiveCursor(newCursorEvent)
	} else {
		cm.activeCursors[newCid].Current = state.Previous
		cm.activeCursors[newCid].Current = newCursorEvent
	}
	se.Value = newCursorEvent
	return se
}
*/

func (cm *CursorManager) executeDownDragUp(ce CursorEvent) {

	if ce.Click == 0 {
		ce.Click = CurrentClick()
	}

	ac, ok := cm.getActiveCursorFor(ce.Cid)
	if !ok {
		// new ActiveCursor
		// Make sure the first ddu is "down"
		if ce.Ddu != "down" {
			// LogWarn("handleDownDragUp: first ddu is not down", "cid", ce.Cid, "ddu", ce.Ddu)
			ce.Ddu = "down"
		}
		ac = NewActiveCursor(ce)
		cm.cursorsMutex.Lock()
		cm.activeCursors[ce.Cid] = ac
		cm.cursorsMutex.Unlock()
	} else {
		// existing ActiveCursor
		ac.Previous = ac.Current
		ac.Current = ce
	}
	ac.Current.Click = CurrentClick()

	for _, handler := range cm.handlers {
		err := handler.onCursorEvent(*ac)
		LogIfError(err)
	}

	LogOfType("cursor", "CursorManager.handleDownDragUp", "ce", ce)

	if ce.Ddu == "up" {
		// LogOfType("cursor", "handleDownDragUp up is deleting cid", "cid", ce.Cid, "ddu", ce.Ddu)
		cm.cursorsMutex.Lock()
		delete(cm.activeCursors, ce.Cid)

		LogOfType("cursor","LoopedCidFor: DELETING CidToLoopedCid entry", "ce.Cid", ce.Cid)
		delete(cm.CidToLoopedCid,ce.Cid)

		cm.cursorsMutex.Unlock()
	}
}

func CursorToOscMsg(ce CursorEvent) *osc.Message {
	msg := osc.NewMessage("/cursor")
	msg.Append(ce.Ddu)
	msg.Append(ce.Cid)
	msg.Append(float32(ce.X))
	msg.Append(float32(ce.Y))
	msg.Append(float32(ce.Z))
	return msg
}

func MidiToOscMsg(me MidiEvent) *osc.Message {
	msg := osc.NewMessage("/midi")
	bytes := me.Msg.Bytes()
	s := "'x"
	for _, b := range bytes {
		s += fmt.Sprintf("%02x", b)
	}
	s += "'"
	msg.Append(s)
	return msg
}

func (cm *CursorManager) autoCursorUp(now time.Time) {

	// checkDelay is the number of CLicks that has to pass
	// before we decide a cursor is no longer present,
	// resulting in an "up" event.
	checkDelay := 8 * OneBeat

	cm.cursorsMutex.RLock()
	var cidsToDelete = []string{}
	for cid, ActiveCursor := range cm.activeCursors {

		dclick := ActiveCursor.Previous.Click - ActiveCursor.Current.Click
		if dclick > checkDelay {
			ce := ActiveCursor.Current
			ce.Ddu = "up"
			LogInfo("autoCursorUp: before handleDownDragUp", "cid", cid)

			cm.cursorsMutex.RUnlock()
			cm.executeDownDragUp(ce)
			cm.cursorsMutex.RUnlock()

			cidsToDelete = append(cidsToDelete, cid)

			cidsToDelete = append(cidsToDelete, cid)
		}
	}
	cm.cursorsMutex.RUnlock()

	cm.deleteActiveCursors(cidsToDelete)
}

func (cm *CursorManager) deleteActiveCursors(cidsToDelete []string) {
	cm.cursorsMutex.Lock()
	for _, cid := range cidsToDelete {
		// LogInfo("autoCursorUp: deleting cursor", "cid", cd)
		delete(cm.activeCursors, cid)
	}
	cm.cursorsMutex.Unlock()
}
