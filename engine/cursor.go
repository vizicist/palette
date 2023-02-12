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
	executeMutex sync.RWMutex

	activeMutex   sync.RWMutex
	activeCursors map[string]*ActiveCursor

	CidToLoopedCid map[string]string
	handlers       map[string]CursorHandler
	sequenceNum    int
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
		activeMutex:    sync.RWMutex{},
		handlers:       map[string]CursorHandler{},
	}
}

func (cm *CursorManager) getActiveCursorFor(cid string) (*ActiveCursor, bool) {
	cm.activeMutex.RLock()
	ac, ok := cm.activeCursors[cid]
	cm.activeMutex.RUnlock()
	return ac, ok
}

func (cm *CursorManager) AddCursorHandler(name string, handler CursorHandler, sources ...string) {
	cm.handlers[name] = handler
}

/*
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
*/

// cleaerCursors sends "up" events to all active cursors, then deletes them
func (cm *CursorManager) clearCursors() {

	currentClick := CurrentClick()
	cidsToDelete := []string{}

	cm.activeMutex.RLock()

	for cid, ac := range cm.activeCursors {
		ce := ac.Current
		ce.Click = currentClick
		ce.Ddu = "up"

		// cm.DeleteActiveCursor(ce.Cid)
		cm.activeMutex.RUnlock()
		cm.ExecuteCursorEvent(ce)
		cm.activeMutex.RLock()

		LogOfType("cursor", "Clearing cursor", "cid", cid)

		cidsToDelete = append(cidsToDelete, cid)
	}

	cm.activeMutex.RUnlock()

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

func (cm *CursorManager) UniqueCid(baseCid string) string {
	base := baseCid
	arr := strings.Split(baseCid, "#")
	if len(arr) == 2 {
		base = arr[0]
	}
	// Return a unique cursor id, not already in cm.ActiveCursors
	for {
		cm.sequenceNum++
		newCid := fmt.Sprintf("%s#%d", base, cm.sequenceNum)

		cm.activeMutex.RLock()
		_, ok := cm.activeCursors[newCid]
		cm.activeMutex.RUnlock()

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
		LogOfType("cursor", "LoopedCidFor: adding new entry", "ce", ce, "loopedCid", loopedCid)
		cm.CidToLoopedCid[ce.Cid] = loopedCid
	}
	return loopedCid
}

func (cm *CursorManager) ExecuteCursorEvent(ce CursorEvent) {

	if ce.Ddu == "clear" {
		TheCursorManager.clearCursors()
		return
	}

	cm.executeMutex.Lock()
	defer cm.executeMutex.Unlock()

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

		cm.activeMutex.Lock()
		cm.activeCursors[ce.Cid] = ac
		cm.activeMutex.Unlock()

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
		cm.DeleteActiveCursor(ce.Cid)
	}
}

func (cm *CursorManager) DeleteActiveCursor(cid string) {
	cm.activeMutex.Lock()
	LogOfType("cursor", "DeleteActiveCursor", "cid", cid)
	delete(cm.activeCursors, cid)
	delete(cm.CidToLoopedCid, cid)
	cm.activeMutex.Unlock()
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

	cm.activeMutex.RLock()
	var cidsToDelete = []string{}
	for cid, ac := range cm.activeCursors {

		dclick := ac.Previous.Click - ac.Current.Click
		if dclick > checkDelay {
			ce := ac.Current
			ce.Ddu = "up"
			LogInfo("autoCursorUp: before handleDownDragUp", "cid", cid)

			cm.DeleteActiveCursor(ce.Cid)
			// cm.activeMutex.RUnlock()
			/////// cm.ExecuteCursorEvent(ce)
			// cm.activeMutex.RUnlock()

			cidsToDelete = append(cidsToDelete, cid)
		}
	}
	cm.activeMutex.RUnlock()

	cm.deleteActiveCursors(cidsToDelete)
}

func (cm *CursorManager) deleteActiveCursors(cidsToDelete []string) {
	cm.activeMutex.Lock()
	for _, cid := range cidsToDelete {
		// LogInfo("autoCursorUp: deleting cursor", "cid", cd)
		delete(cm.activeCursors, cid)
	}
	cm.activeMutex.Unlock()
}
