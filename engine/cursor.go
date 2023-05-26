package engine

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hypebeast/go-osc/osc"
)

var TheCursorManager *CursorManager

type ActiveCursor struct {
	Current     CursorEvent
	Previous    CursorEvent
	NoteOn      *NoteOn
	NoteOnClick Clicks
	Patch       *Patch
	Button      string
	loopIt      bool
	loopBeats   int
	loopFade    float32
	maxZ        float32
}

type CursorManager struct {
	executeMutex sync.RWMutex

	activeMutex   sync.RWMutex
	activeCursors map[string]*ActiveCursor

	CidToLoopedCidMutex sync.RWMutex
	CidToLoopedCid      map[string]string
	handlers            map[string]CursorHandler
	sequenceNum         int
}

// CursorDeviceCallbackFunc xxx
type CursorCallbackFunc func(e CursorEvent)

type CursorHandler interface {
	onCursorEvent(state ActiveCursor) error
}

// NewActiveCursor - create a new ActiveCursor for a CursorEvent
// An ActiveCursor can be for a Button or a Patch area.
func NewActiveCursor(ce CursorEvent) *ActiveCursor {

	patch, button := TheQuadPro.PatchForCursorEvent(ce)
	if patch == nil && button == "" {
		LogWarn("No Patch or Button for CursorEvent", "ce", ce)
		return nil
	}
	ac := &ActiveCursor{
		Current:  ce,
		Previous: ce,
		Button:   button,
		Patch:    patch,
		maxZ:     0,
	}
	if ac.Patch != nil {
		ac.loopIt = patch.GetBool("misc.looping_on")
		ac.loopBeats = patch.GetInt("misc.looping_beats")
		ac.loopFade = patch.GetFloat("misc.looping_fade")
	}
	return ac
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
	// LogWarn("Should generateCursorGesture be using ScheduleCursorEvent or ExecuteCursorEvent?")
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

	cm.CidToLoopedCidMutex.Lock()
	defer cm.CidToLoopedCidMutex.Unlock()

	loopedCid, ok := cm.CidToLoopedCid[ce.Cid]
	if !ok {
		if ce.Ddu == "up" { // Not totally sure why this happens
			return ""
		}
		// LogWarn("adjustLoopedCid: oldCid not found in ActiveCursor", "oldCid", oldCid)
		loopedCid = cm.UniqueCid(ce.Cid)
		cm.CidToLoopedCid[ce.Cid] = loopedCid
	}
	return loopedCid
}

const LoopFadeZThreshold = 0.005

func (cm *CursorManager) ExecuteCursorEvent(ce CursorEvent) {

	TheEngine.RecordCursorEvent(ce)

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
		if ac == nil {
			LogWarn("CursorManager.ExecuteCursorEvent - unable to create ActiveCursor", "ce", ce)
			return
		}

		cm.activeMutex.Lock()
		LogOfType("cursor", "ExecuteCursorEvent: adding new ActiveCursor", "cid", ce.Cid, "ac", ac)
		cm.activeCursors[ce.Cid] = ac
		cm.activeMutex.Unlock()

	} else {

		// existing ActiveCursor
		ac.Previous = ac.Current
		ac.Current = ce

	}

	ac.Current.Click = CurrentClick()

	if ac.loopIt {

		// the looped CursorEvent starts out as a copy of the ActiveCursor's Current value
		loopce := ac.Current
		loopce.Click = CurrentClick() + OneBeat*Clicks(ac.loopBeats)

		// The looped CursorEvents should have unique cid values.
		loopce.Cid = TheCursorManager.LoopedCidFor(ac.Current)

		// Fade the Z value
		loopce.Z = loopce.Z * ac.loopFade
		if loopce.Z > ac.maxZ {
			ac.maxZ = loopce.Z
		}

		// LogInfo("looped CursorEvent", "ce.Z", ce.Z, "loopFade", ac.loopFade)
		se := NewSchedElement(loopce.Click, loopce)
		TheScheduler.insertScheduleElement(se)
	}

	TheEngine.sendToOscClients(CursorToOscMsg(ce))

	for _, handler := range cm.handlers {
		err := handler.onCursorEvent(*ac)
		LogIfError(err)
	}

	LogOfType("cursor", "CursorManager.handleDownDragUp", "ce", ce)

	if ce.Ddu == "up" {
		// LogOfType("cursor", "handleDownDragUp up is deleting cid", "cid", ce.Cid, "ddu", ce.Ddu)
		cm.DeleteActiveCursorIfZLessThan(ce.Cid, LoopFadeZThreshold)
	}
}

func (cm *CursorManager) DeleteActiveCursor(cid string) {
	// reuse this routine, but mke sure that it will
	cm.DeleteActiveCursorIfZLessThan(cid, 1.0)
}

// Set threshold to 1.0 (or greater) if you want to
func (cm *CursorManager) DeleteActiveCursorIfZLessThan(cid string, threshold float32) {

	cm.activeMutex.Lock()
	ac, ok := cm.activeCursors[cid]
	loopCid := ""
	if !ok {
		// LogWarn("DeleteActiveCursor: cid not found in ActiveCursor", "cid", cid)
	} else {
		// LogInfo("DeleteActiveCursorIfZLessThan", "cid", cid, "threshold", threshold, "ac.maxZ", ac.maxZ)
		if ac.maxZ < threshold {
			// we want to remove things that this ActiveCursor has created for looping.
			cm.activeMutex.Unlock()
			loopCid = cm.LoopedCidFor(ac.Current)
			cm.activeMutex.Lock()
			// but wait after we detete it from
		}
	}
	delete(cm.activeCursors, cid)
	cm.activeMutex.Unlock()

	cm.CidToLoopedCidMutex.Lock()
	delete(cm.CidToLoopedCid, cid)
	cm.CidToLoopedCidMutex.Unlock()

	if loopCid != "" {
		TheScheduler.DeleteEventsWhoseCidIs(loopCid)
	}
}

func (cm *CursorManager) DeleteActiveCursorsForCidPrefix(cidPrefix string) {

	// First construct the list
	cm.activeMutex.RLock()

	if len(cm.activeCursors) > 0 {
		LogInfo("DeleteActiveCursorsForCidPrefix", "cidPrefix", cidPrefix)
	}
	todelete := []string{}
	for _, ac := range cm.activeCursors {
		if strings.HasPrefix(ac.Current.Cid, cidPrefix) {
			todelete = append(todelete, ac.Current.Cid)
		}
	}
	cm.activeMutex.RUnlock()

	for _, cid := range todelete {
		cm.DeleteActiveCursor(cid)
		TheScheduler.DeleteEventsWhoseCidIs(cid)
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

	cm.activeMutex.RLock()
	var cidsToDelete = []string{}
	for cid, ac := range cm.activeCursors {

		dclick := ac.Previous.Click - ac.Current.Click
		if dclick > checkDelay {
			ce := ac.Current
			ce.Ddu = "up"
			LogInfo("autoCursorUp: before handleDownDragUp", "cid", cid)

			cm.ExecuteCursorEvent(ce)
			cidsToDelete = append(cidsToDelete, cid)
		}
	}
	cm.activeMutex.RUnlock()

	cm.deleteActiveCursors(cidsToDelete)

}

func (cm *CursorManager) deleteActiveCursors(cidsToDelete []string) {
	for _, cid := range cidsToDelete {
		cm.DeleteActiveCursor(cid)
	}
}

/*
func (cm *CursorManager) ClearAll() {
	LogInfo("CursorManager.ClearAll") {
	LogInfo("CursorManager.ClearAll")
	for ac := range cm.activeCursors {
		m.DeleteActiveCursor(cid)rAll")
	for ac := range cm.activeCursors {
	m.DeleteActiveCursor(cid)ursors {
	m.DeleteActiveCursor(cid)
	gInfo("ClearAll: deleting", "cid", ac)


/

/*
fn (cm *CusorManager)checkThrehold(ac *ActiveCurs()
/ XXX - shuld checking for "tolow Z" be done here  NO!!
	/XX - it hould be done whenthe "up" is receieve
	loopce.Z LoopFadeZThreshold
	gInfo("oopce.Z too low, shoul be deleting", "loopce",lopce
	ntinue

	}
*/
