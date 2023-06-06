package engine

import (
	"fmt"
	"math/rand"
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
	pitchOffset int
}

type CursorManager struct {
	executeMutex sync.RWMutex

	activeMutex   sync.RWMutex
	activeCursors map[string]*ActiveCursor

	CidToLoopedCidMutex sync.RWMutex
	CidToLoopedCid      map[string]string
	handlers            map[string]CursorHandler
	uniqueInt           int
}

type CursorPos struct {
	x, y, z float32
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

	ac.pitchOffset = TheEngine.currentPitchOffset

	/*
		// Hardcoded, channel 10 is usually drums, doesn't get transposed
		// Should probably be an attribute of the Synth.
		const drumChannel = 10
		if TheEngine.autoTransposeOn && synth.portchannel.channel != drumChannel {
			pitchOffset += TheEngine.autoTransposeValues[TheEngine.autoTransposeIndex]
		}
		newpitch := int(pitch) + pitchOffset
		if newpitch < 0 {
			newpitch = 0
		} else if newpitch > 127 {
			newpitch = 127
		}
	*/

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

func (cm *CursorManager) UniqueInt() int {
	n := cm.uniqueInt
	cm.uniqueInt++
	return n
}

func (cm *CursorManager) UniqueCid(source string) string {
	cm.activeMutex.Lock()
	defer cm.activeMutex.Unlock()
	return fmt.Sprintf("%s#%d", source, TheCursorManager.UniqueInt())
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

func (cm *CursorManager) GenerateRandomGesture(source string, attrib string, dur time.Duration) {
	pos0 := CursorPos{
		rand.Float32(),
		rand.Float32(),
		rand.Float32() / 2.0,
	}
	pos1 := CursorPos{
		rand.Float32(),
		rand.Float32(),
		rand.Float32() / 2.0,
	}
	cm.GenerateGesture(source, attrib, dur, pos0, pos1)
}

func (cm *CursorManager) GenerateGesture(source string, attrib string, dur time.Duration, pos0 CursorPos, pos1 CursorPos) {

	cid := cm.UniqueCid(source)
	if attrib != "" {
		cid = cid + "," + attrib
	}
	LogOfType("cursor", "generateCursoresture start",
		"cid", cid, "noteDuration", dur, "attrib", attrib, "pos0", pos0, "pos1", pos1)

	nsteps, err := GetParamInt("engine.gesturesteps")
	if err != nil {
		LogIfError(err)
		return
	}

	for n := 0; n <= nsteps; n++ {
		var ddu string
		if n == 0 {
			ddu = "down"
		} else if n < nsteps {
			ddu = "drag"
		} else {
			ddu = "up"
		}
		ce := CursorEvent{
			Cid:   cid,
			Click: CurrentClick(),
			Ddu:   ddu,
			X:     pos0.x + pos1.x*float32(n)/float32(nsteps),
			Y:     pos0.x + pos1.y*float32(n)/float32(nsteps),
			Z:     pos0.x + pos1.z*float32(n)/float32(nsteps),
			Area:  0,
		}
		cm.ExecuteCursorEvent(ce)
		time.Sleep(time.Duration(dur.Nanoseconds() / int64(nsteps)))
	}
}

/*
Keep track of names attached to each cursor "down",
and use those names on subsequent "drag" and "up" events.
*/
func (cm *CursorManager) LoopedCidFor(ce CursorEvent, warn bool) string {

	cm.CidToLoopedCidMutex.Lock()
	defer cm.CidToLoopedCidMutex.Unlock()

	loopedCid, ok := cm.CidToLoopedCid[ce.Cid]
	if !ok {
		if ce.Ddu == "up" { // Not totally sure why this happens
			if warn {
				LogWarn("Why is this happening?")
			}
			return ""
		}
		loopedCid = fmt.Sprintf("%s#%d", ce.Source(), cm.UniqueInt())
		cm.CidToLoopedCid[ce.Cid] = loopedCid
	}
	return loopedCid
}

const LoopFadeZThreshold = 0.001

func (cm *CursorManager) ExecuteCursorEvent(ce CursorEvent) {

	TheEngine.RecordCursorEvent(ce)

	if ce.Ddu == "clear" {
		TheCursorManager.clearCursors()
		return
	}

	// Don't put lock above clearCursors(), since it calls ExecuteCursorEvent recursively
	cm.executeMutex.Lock()
	defer cm.executeMutex.Unlock()

	if ce.Click == 0 {
		ce.Click = CurrentClick()
	}

	LogOfType("cursor", "ExecuteCursorEvent", "ce", ce)

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

		LogOfType("cursor", "ExecuteCursorEvent: using existing ActiveCursor", "cid", ce.Cid, "ac", ac)
		// existing ActiveCursor
		ac.Previous = ac.Current
		ac.Current = ce

	}

	ac.Current.Click = CurrentClick()

	if ac.loopIt {

		// the looped CursorEvent starts out as a copy of the ActiveCursor's Current value
		loopce := ac.Current
		loopce.Click = CurrentClick() + OneBeat*Clicks(ac.loopBeats)

		// The looped CursorEvents should have unique cid val,ues.
		loopce.Cid = TheCursorManager.LoopedCidFor(ac.Current, true /*warn*/)

		// Fade the Z value
		loopce.Z = loopce.Z * ac.loopFade
		// LogInfo("loopcd.Z is now", "Z", loopce.Z)
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
			// LogInfo("DeleteActiveCursorIfZLessThan REMOVING!")
			loopCid = cm.LoopedCidFor(ac.Current, false /*warn*/)
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

func (cm *CursorManager) PlayCursor(source string, dur time.Duration, x, y, z float32) {
	cid := cm.UniqueCid(source)
	ce := CursorEvent{
		Cid:   cid,
		Click: CurrentClick(),
		Ddu:   "down",
		X:     x,
		Y:     y,
		Z:     z,
	}
	cm.ExecuteCursorEvent(ce)
	// Send the cursor up, but don't block the loop
	go func(ce CursorEvent) {
		time.Sleep(dur)
		ce.Ddu = "up"
		ce.Click = CurrentClick()
		cm.ExecuteCursorEvent(ce)
	}(ce)
}
