package kit

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/hypebeast/go-osc/osc"
)

var TheCursorManager *CursorManager

// CursorEvent is a single Cursor event.
// NOTE: it's assumed that we can copy a CursorEvent by copying the value.
// Do note add pointers to this struct.
type CursorEvent struct {
	// Named CClick to catch everyone that accesses it
	CClick Clicks `json:"Click"`
	GID    int    `json:"Gid"`
	Tag    string `json:"Tag"`
	// Source string
	Ddu  string    `json:"Ddu"` // "down", "drag", "up" (sometimes "clear")
	Pos  CursorPos `json:"Pos"`
	Area float64   `json:"Area"`
}

// ActiveCursor is an OSC message
type ActiveCursor struct {
	Current     CursorEvent
	Previous    CursorEvent
	NoteOn      *NoteOn
	NoteOnClick Clicks
	Patch       *Patch
	Button      string
	loopIt      bool
	loopBeats   int
	loopFade    float64
	maxZ        float64
	pitchOffset int32
}

type CursorManager struct {
	executeMutex sync.RWMutex
	ClickMutex   sync.RWMutex

	// activeMutex   sync.RWMutex
	activeMutex   sync.Mutex
	activeCursors map[int]*ActiveCursor // map of Gid to ActiveCursor

	GIDToLoopedGIDMutex sync.RWMutex
	GIDToLoopedGID      map[int]int
	handlers            map[string]CursorHandler
	uniqueInt           int
	uniqueMutex         sync.Mutex
	LoopThreshold       float64
	cursorRand          *rand.Rand
	cursorRandMutex     sync.Mutex
}

type CursorPos struct {
	X, Y, Z float64
}

type CursorCallbackFunc func(e CursorEvent)

type CursorHandler interface {
	onCursorEvent(state ActiveCursor) error
}

func NewCursorEvent(gid int, tag string, ddu string, pos CursorPos) CursorEvent {
	ce := CursorEvent{
		CClick: CurrentClick(),
		GID:    gid,
		Tag:    tag,
		Ddu:    ddu,
		Pos:    pos,
		Area:   0,
	}
	return ce
}

func NewCursorClearEvent() CursorEvent {
	gid := TheCursorManager.UniqueGID()
	return NewCursorEvent(gid, "", "clear", CursorPos{})
}

// NewActiveCursor - create a new ActiveCursor for a CursorEvent
// An ActiveCursor can be for a Button or a Patch area.
func NewActiveCursor(ce CursorEvent) *ActiveCursor {

	patch, button := TheQuad.PatchForCursorEvent(ce)
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
		if !ac.loopIt {
			ac.loopFade = 0.0
		}
	}

	forceOverride, _ := GetParamBool("global.looping_override")
	if forceOverride {
		forcebeats, _ := GetParamInt("global.looping_beats")
		forcefade, _ := GetParamFloat("global.looping_fade")
		ac.loopIt = forceOverride
		ac.loopBeats = forcebeats
		ac.loopFade = forcefade
	}

	// LogInfo("NewactiveCursor", "ac", ac)

	ac.pitchOffset = TheEngine.currentPitchOffset.Load()

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
	cm := &CursorManager{
		activeCursors:  map[int]*ActiveCursor{},
		GIDToLoopedGID: map[int]int{},
		activeMutex:    sync.Mutex{},
		handlers:       map[string]CursorHandler{},
		uniqueInt:      1,
		LoopThreshold:  0.01,
		cursorRand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	return cm
}

func (cm *CursorManager) UniqueGID() int {
	cm.uniqueMutex.Lock()
	defer cm.uniqueMutex.Unlock()
	unique := cm.uniqueInt
	cm.uniqueInt++
	return unique
}

func (cm *CursorManager) getActiveCursorFor(gid int) (*ActiveCursor, bool) {
	cm.activeMutex.Lock()
	ac, ok := cm.activeCursors[gid]
	cm.activeMutex.Unlock()
	return ac, ok
}

func (cm *CursorManager) AddCursorHandler(name string, handler CursorHandler, sources ...string) {
	cm.handlers[name] = handler
}

func (cm *CursorManager) clearActiveCursors(tag string, checkDelay Clicks) {

	currentClick := CurrentClick()

	gidsToDelete := []int{}
	cursorUpEvents := []CursorEvent{}

	cm.activeMutex.Lock()

	for gid, ac := range cm.activeCursors {

		if tag != "*" && ac.Current.Tag != tag {
			continue
		}

		TheCursorManager.ClickMutex.Lock()
		acCurrentCe := ac.Current
		TheCursorManager.ClickMutex.Unlock()

		currClick := acCurrentCe.GetClick()

		acPreviousCe := ac.Previous

		prevClick := acPreviousCe.GetClick()

		dclick := currClick - prevClick
		// Not sure the reason why dclick is sometimes -1 or -2, I should look at it later.
		// if dclick < 0 {
		// 	LogWarn("clearActiveCursors: Unexpected negative dclick?", "currClick", currClick, "prevClick", prevClick)
		// }

		if dclick < checkDelay {
			continue
		}

		ceUp := ac.Current
		ceUp.SetClick(currentClick)
		ceUp.Ddu = "up"

		LogOfType("cursor", "Clearing Activecursor", "gid", gid)

		cursorUpEvents = append(cursorUpEvents, ceUp)
		gidsToDelete = append(gidsToDelete, gid)
	}

	cm.activeMutex.Unlock()

	for _, ce := range cursorUpEvents {
		cm.ExecuteCursorEvent(ce)
	}

	cm.deleteActiveCursors(gidsToDelete)
}

/*
func (cm *CursorManager) RandPos() CursorPos {
	randZFactor := 0.5
	return CursorPos{
		X: cm.cursorRand.Float64(),
		Y: cm.cursorRand.Float64(),
		Z: cm.cursorRand.Float64() * randZFactor,
	}
}
*/

func (cm *CursorManager) GenerateCenterGesture(tag string, dur time.Duration) {

	pos0 := CursorPos{
		X: 0.5,
		Y: 0.5,
		Z: 0.5,
	}
	pos1 := pos0

	cm.GenerateGesture(tag, 1, dur, pos0, pos1)
}

func lineLength(pos0 CursorPos, pos1 CursorPos) float64 {
	return math.Sqrt(math.Pow(pos1.X-pos0.X, 2) + math.Pow(pos1.Y-pos0.Y, 2))
}

func (cm *CursorManager) zRand(zmin float64, zmax float64) float64 {
	z := 0.0
	for z == 0.0 {
		z = zmin + cm.cursorRand.Float64()*(zmax-zmin)
		if z < zmin || z > zmax {
			z = 0.0
		}
	}
	return z
}

func (cm *CursorManager) GenerateRandomGesture(tag string, numsteps int, dur time.Duration) {

	cm.cursorRandMutex.Lock()

	pos0 := CursorPos{
		X: cm.cursorRand.Float64(),
		Y: cm.cursorRand.Float64(),
	}
	pos1 := CursorPos{}
	am := TheAttractManager
	for pos1.X == 0.0 {
		pos1.X = cm.cursorRand.Float64()
		pos1.Y = cm.cursorRand.Float64()
		// Keep going until desired length is achieved
		leng := lineLength(pos0, pos1)
		if leng < am.GestureMinLength || leng > am.GestureMaxLength {
			pos1.X = 0.0 // try again
		}
	}
	pos0.Z = cm.zRand(am.GestureZMin, am.GestureZMax)
	pos1.Z = cm.zRand(am.GestureZMin, am.GestureZMax)

	// Occasionally force exactly horizontal (since pitch goes horizontal)
	if cm.cursorRand.Int()%4 == 0 {
		pos1.Y = pos0.Y
	}

	cm.cursorRandMutex.Unlock()

	cm.GenerateGesture(tag, numsteps, dur, pos0, pos1)
}

func (cm *CursorManager) GenerateGesture(tag string, numsteps int, dur time.Duration, pos0 CursorPos, pos1 CursorPos) {

	gid := cm.UniqueGID()

	LogOfType("gesture", "generateCursorgesture start",
		"gid", gid, "noteDuration", dur, "durNano", dur.Nanoseconds(), "tags", tag, "pos0", pos0, "pos1", pos1)

	dpos := CursorPos{
		X: pos1.X - pos0.X,
		Y: pos1.Y - pos0.Y,
		Z: pos1.Z - pos0.Z,
	}

	for n := 0; n <= numsteps; n++ {
		var ddu string
		if n == 0 {
			ddu = "down"
		} else if n < numsteps {
			ddu = "drag"
		} else {
			ddu = "up"
		}

		// Not sure about this Lock
		// cm.activeMutex.Lock()
		amount := float64(n) / float64(numsteps)
		pos := CursorPos{
			X: pos0.X + dpos.X*amount,
			Y: pos0.Y + dpos.Y*amount,
			Z: pos0.Z + dpos.Z*amount,
		}
		ce := NewCursorEvent(gid, tag, ddu, pos)
		LogOfType("gesture", "generateCursorEvent", "n", n, "amount", amount, "pos", pos, "tag", tag)
		// cm.activeMutex.Unlock()

		cm.ExecuteCursorEvent(ce)
		sleepDur := time.Duration(dur.Nanoseconds() / int64(numsteps))
		LogOfType("gesture", "generateGesture sleep", "sleepDur", sleepDur, "tag", tag)
		time.Sleep(sleepDur)
	}
}

func (ce *CursorEvent) Source() string {
	// Assumes the source is the first (and often only) thing in the tag
	words := strings.Split(ce.Tag, ",")
	return words[0]
}

func (ce *CursorEvent) SetClick(click Clicks) {
	TheCursorManager.ClickMutex.Lock()
	ce.CClick = click
	TheCursorManager.ClickMutex.Unlock()
}

func (ce CursorEvent) GetClick() Clicks {
	TheCursorManager.ClickMutex.RLock()
	clk := ce.CClick
	TheCursorManager.ClickMutex.RUnlock()
	return clk
}

// LoopedGIDFor - Keep track of names attached to each cursor "down",
// and use those names on subsequent "drag" and "up" events.
func (cm *CursorManager) LoopedGIDFor(ce CursorEvent, warn bool) int {

	cm.GIDToLoopedGIDMutex.Lock()
	defer cm.GIDToLoopedGIDMutex.Unlock()

	loopedGID, ok := cm.GIDToLoopedGID[ce.GID]
	if ok {
		// LogInfo("LoopedGidFor FOUND", "ce.Gid", ce.Gid, "loopedGid", loopedGid)
		return loopedGID
	}
	switch ce.Ddu {
	case "down":
		loopedGID = cm.UniqueGID()
		cm.GIDToLoopedGID[ce.GID] = loopedGID
		// LogInfo("CursorManager.LoopedGidFor down", "original gid", ce.Gid, "loopedGid", loopedGid)
	case "drag":
		loopedGID = cm.UniqueGID()
		cm.GIDToLoopedGID[ce.GID] = loopedGID
		// LogInfo("CursorManager.LoopedGidFor drag", "original gid", ce.Gid, "loopedGid", loopedGid)
	case "up":
		// Not totally sure why this happens
		if warn {
			LogWarn("Why is this happening?")
		}
		// loopedGid = 0
		loopedGID = cm.UniqueGID()
		cm.GIDToLoopedGID[ce.GID] = loopedGID
	}
	// LogInfo("CursorManager.LoopedGidFor up", "original gid", ce.Gid, "loopedGid", loopedGid)
	return loopedGID
}

var BugFixWarningCount = 0

func (cm *CursorManager) ExecuteCursorEvent(ce CursorEvent) {

	TheEngine.RecordCursorEvent(ce)

	fadeThreshold, err := GetParamFloat("global.looping_fadethreshold")
	if err != nil {
		LogIfError(err)
	} else {
		cm.executeMutex.Lock()
		TheCursorManager.LoopThreshold = fadeThreshold
		cm.executeMutex.Unlock()
	}

	cursorscaley, err := GetParamFloat("global.cursorscaley")
	if err == nil && cursorscaley > 0.0 {
		ce.Pos.Y *= cursorscaley
	}
	cursoroffsety, err := GetParamFloat("global.cursoroffsety")
	if err == nil {
		ce.Pos.Y += cursoroffsety
	}

	cursorscalex, err := GetParamFloat("global.cursorscalex")
	if err == nil && cursorscalex > 0.0 {
		ce.Pos.X *= cursorscalex
	}
	cursoroffsetx, err := GetParamFloat("global.cursoroffsetx")
	if err == nil {
		ce.Pos.X += cursoroffsetx
	}

	if ce.Ddu == "clear" {
		if ce.Tag == "" {
			LogWarn("CursorManager.ExecuteCursorEvent: clear with empty tag?")
		}
		TheCursorManager.ClearAllActiveCursors(ce.Tag)
		return
	}

	// Don't put lock above clearActiveCursors(), since it calls ExecuteCursorEvent recursively
	cm.executeMutex.Lock()
	defer cm.executeMutex.Unlock()

	if ce.GetClick() == 0 {
		ce.SetClick(CurrentClick())
	}

	LogOfType("cursor", "ExecuteCursorEvent", "gid", ce.GID, "ddu", ce.Ddu, "x", ce.Pos.X, "y", ce.Pos.Y, "z", ce.Pos.Z)

	ac, ok := cm.getActiveCursorFor(ce.GID)
	if !ok {

		// new ActiveCursor

		// If it's an "up" event for an unknown gid, don't do anything.  This shouldn't happen,
		// but there's a bug in looping where this happens occasionally.  After the CFNM show,
		// this should be investigated.  It probably has to do with when looping events
		// in the middle of a gesture are deleted.  The whole looping strategy should be rethought.
		// I.e. probably, no looped events in the middle of a gesture should be deleted until
		// the maxz is less than the threshold, and then then entire gesture should be deleted at once.
		if ce.Ddu == "up" {
			BugFixWarningCount++
			if BugFixWarningCount < 10 {
				LogWarn("CursorManager.ExecuteCursorEvent - NEW BUG FIX, ignoring up cursor event", "ce", ce)
			}
			return
		}

		// Make sure the first ddu is "down", if we get drag events for an unknown gid
		if ce.Ddu == "drag" {
			// LogWarn("handleDownDragUp: first ddu is not down", "gid", ce.Gid, "ddu", ce.Ddu)
			ce.Ddu = "down"
		} else if ce.Ddu != "down" {
			// Just in case, shouldn't happen
			LogWarn("CursorManager.ExecuteCursorEvent - unexpected Ddu", "ce", ce)
			return

		}

		ac = NewActiveCursor(ce)
		if ac == nil {
			LogWarn("CursorManager.ExecuteCursorEvent - unable to create ActiveCursor", "ce", ce)
			return
		}

		cm.activeMutex.Lock()
		LogOfType("cursor", "ExecuteCursorEvent: adding new ActiveCursor", "gid", ce.GID, "ac", ac)
		cm.activeCursors[ce.GID] = ac
		cm.activeMutex.Unlock()

	} else {

		// LogOfType("cursor", "ExecuteCursorEvent: using existing ActiveCursor", "gid", ce.Gid, "ac", ac)
		// existing ActiveCursor
		cm.activeMutex.Lock()
		ac.Previous = ac.Current
		ac.Current = ce
		cm.activeMutex.Unlock()

	}

	currClick := CurrentClick()
	ac.Current.SetClick(currClick)

	if ac.loopIt {
		se := cm.LoopCursorEvent(ac)
		if se != nil {
			if ac.Current.Ddu == "up" {
				LogOfType("cursor", "UP cursor", "maxZ", ac.maxZ)
			}
			TheScheduler.insertScheduleElement(se)
		}
	}

	TheEngine.sendToOscClients(CursorToOscMsg(ce))

	// LogOfType("cursor", "CursorManager.handleDownDragUp before doing handlers", "ce", ce)
	for _, handler := range cm.handlers {
		err := handler.onCursorEvent(*ac)
		LogIfError(err)
	}

	// LogOfType("cursor", "CursorManager.handleDownDragUp", "ce", ce)

	if ce.Ddu == "up" {
		// LogInfo("ExecuteCursorEvent UP")
		LogOfType("cursor", "handleDownDragUp up is deleting gid", "gid", ce.GID, "ddu", ce.Ddu)
		cm.DeleteActiveCursorIfZLessThan(ce.GID, cm.LoopThreshold)
	}
}

func (cm *CursorManager) LoopCursorEvent(ac *ActiveCursor) *SchedElement {
	loopce := ac.Current
	// the looped CursorEvent starts out as a copy of the ActiveCursor's Current value
	loopce.SetClick(CurrentClick() + OneBeat*Clicks(ac.loopBeats))

	// The looped CursorEvents should have unique gid val,ues.
	loopce.GID = TheCursorManager.LoopedGIDFor(ac.Current, true /*warn*/)
	if loopce.GID == 0 {
		LogWarn("HEY!!! loopIt LoopedGidFor returns 0?")
		return nil
	}
	// LogInfo("ac.loopIt LoopedGidFor", "loopce.Gid", loopce.Gid)

	// Fade the Z value
	fade := ac.loopFade
	if ac.Patch != nil {
		// ac.loopIt = patch.GetBool("misc.looping_on")
		// ac.loopBeats = patch.GetInt("misc.looping_beats")
		fade = ac.Patch.GetFloat("misc.looping_fade")
	}
	forceOverride, _ := GetParamBool("global.looping_override")
	if forceOverride {
		fade, _ = GetParamFloat("global.looping_fade")
	}

	newZ := loopce.Pos.Z * fade
	LogOfType("loop", "loopcd.Z faded", "origZ", loopce.Pos.Z, "newZ", newZ, "fade", fade)
	loopce.Pos.Z = newZ

	if loopce.Pos.Z < cm.LoopThreshold && loopce.Ddu != "up" {
		LogOfType("loop", "loopce.Z is small, NOT LOOPING IT", "loopce", loopce)
		return nil
	}

	if loopce.Pos.Z > ac.maxZ {
		ac.maxZ = loopce.Pos.Z
	}

	// LogInfo("looped CursorEvent", "ce.Z", ce.Z, "loopFade", ac.loopFade)
	return NewSchedElement(loopce.GetClick(), loopce.Tag, loopce)
}

func (cm *CursorManager) DeleteActiveCursor(gid int) {
	cm.activeMutex.Lock()
	// LogInfo("NEW DELETEACTIVECURSOR gid","gid",gid)
	ac, ok := cm.activeCursors[gid]
	if !ok {
		// LogWarn("DeleteActiveCursor: gid not found in ActiveCursor", "gid", gid)
	} else {
		// LogInfo("DeleteActiveCursor: gid found","noteon",ac.NoteOn,"gid",gid)
		if ac.NoteOn != nil {
			// LogInfo("DeleteActiveCursor: gid found SENDING NOTEOFF!","noteon",ac.NoteOn,"gid",gid)
			noteOff := NewNoteOffFromNoteOn(ac.NoteOn)
			ac.NoteOn.Synth.SendNoteToMidiOutput(noteOff)
			// LogInfo("DeleteActiveCursor: gid found AFTER NOTEOFF!","noteoff",noteOff,"gid",gid)
		} else {
			LogWarn("DeleteActiveCursor: gid found, NO NOTEON?", "gid", gid)
		}

	}
	delete(cm.activeCursors, gid)
	cm.activeMutex.Unlock()
}

// DeleteActiveCursorIfZLessThan deletes the ActiveCursor if its Z value is less than threshold.
func (cm *CursorManager) DeleteActiveCursorIfZLessThan(gid int, threshold float64) {

	// LogInfo("BEGIN DeleteActiveCursorIfZ 1", "gid", gid,"sched", TheScheduler.ToString())

	cm.activeMutex.Lock()
	ac, ok := cm.activeCursors[gid]
	loopGID := 0
	if !ok {
		// LogWarn("DeleteActiveCursor: gid not found in ActiveCursor", "gid", gid)
	} else {
		// LogInfo("DeleteActiveCursorIfZLessThan", "gid", gid, "threshold", threshold, "ac.maxZ", ac.maxZ)
		LogOfType("cursor", "DeleteActiveCursorIfZLessThan", "maxZ", ac.maxZ, "threshold", threshold, "gid", ac.Current.GID)
		if ac.maxZ < threshold {
			// we want to remove things that this ActiveCursor has created for looping.
			loopGID = cm.LoopedGIDFor(ac.Current, false /*don't warn*/)
			if loopGID == 0 {
				LogWarn("HEY!!! in DeleteActiveCursorIfZLessThan LoopedGidFor returns 0?")
			} else {
				LogOfType("cursor", "DeleteActiveCursorIfZLessThan deleting!!", "loopGid", loopGID)
				delete(cm.activeCursors, gid)
				delete(cm.activeCursors, loopGID)
				// LogInfo("DeleteActiveCursorIfZLessThan REMOVING", "loopGid", loopGid, "gid", gid, "ac.maxZ", ac.maxZ, "gid", gid)
			}
		}
	}
	cm.activeMutex.Unlock()

	if loopGID == 0 {
		return
	}

	// LogInfo("SHOULD BE SENDING NOTEOFF FOR THIS ACTIVE CURSOR?", "gid", gid, "ac", ac, "ac.NoteOn", ac.NoteOn)

	cm.GIDToLoopedGIDMutex.Lock()
	delete(cm.GIDToLoopedGID, gid)
	cm.GIDToLoopedGIDMutex.Unlock()

	if loopGID != 0 {
		TheScheduler.DeleteCursorEventsWhoseGIDIs(loopGID)
		TheScheduler.DeleteCursorEventsWhoseGIDIs(gid)
	}
}

func (cm *CursorManager) DeleteActiveCursorsForTag(tag string) {

	cm.activeMutex.Lock()
	defer cm.activeMutex.Unlock()

	for gid, ac := range cm.activeCursors {
		if ac.Current.Tag == tag {
			delete(cm.activeCursors, gid)
		}
	}
}

func CursorToOscMsg(ce CursorEvent) *osc.Message {
	msg := osc.NewMessage("/cursor")
	msg.Append(ce.Ddu)
	// FFGL code expects a string
	msg.Append(fmt.Sprintf("%d", ce.GID))
	// Note: OSC messages must use 32-bit floats
	msg.Append(float32(ce.Pos.X))
	msg.Append(float32(ce.Pos.Y))
	msg.Append(float32(ce.Pos.Z))
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

func (cm *CursorManager) CheckAutoCursorUp() {
	// checkDelay is the number of CLicks that has to pass
	// before we decide a cursor is no longer present,
	// resulting in an "up" event.
	checkDelay := 8 * OneBeat
	cm.clearActiveCursors("*", checkDelay)
}

func (cm *CursorManager) ClearAllActiveCursors(tag string) {
	cm.clearActiveCursors(tag, 0)
}

/*
func (cm *CursorManager) oldautoCursorUp() {

	currentClick := CurrentClick()

	gidsToDelete := []int{}
	cursorUpEvents := []CursorEvent{}

	cm.activeMutex.Lock()

	for gid, ac := range cm.activeCursors {

		dclick := ac.Current.GetClick() - ac.Previous.GetClick()
		if dclick > checkDelay {

			cdUp := ac.Current
			cdUp.SetClick(currentClick)
			cdUp.Ddu = "up"

			cursorUpEvents = append(cursorUpEvents, cdUp)
			gidsToDelete = append(gidsToDelete, gid)
		}
	}
	cm.activeMutex.Unlock()

	for _, ce := range cursorUpEvents {
		cm.ExecuteCursorEvent(ce)
	}
	cm.deleteActiveCursors(gidsToDelete)
}
*/

func (cm *CursorManager) deleteActiveCursors(gidsToDelete []int) {
	for _, gid := range gidsToDelete {
		cm.DeleteActiveCursor(gid)
	}
}

func (cm *CursorManager) PlayCursor(tag string, dur time.Duration, pos CursorPos) {
	gid := cm.UniqueGID()
	ce := NewCursorEvent(gid, tag, "down", pos)
	cm.ExecuteCursorEvent(ce)
	// Send the cursor up, but don't block the loop
	go func(ce CursorEvent) {
		time.Sleep(dur)
		ce.Ddu = "up"
		ce.SetClick(CurrentClick())
		cm.ExecuteCursorEvent(ce)
	}(ce)
}
