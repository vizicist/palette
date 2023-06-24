package engine

import (
	"fmt"
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
	Gid    int    `json:"Gid"`
	Tag    string `json:"Tag"`
	// Source string
	Ddu  string    `json:"Ddu"` // "down", "drag", "up" (sometimes "clear")
	Pos  CursorPos `json:"Pos"`
	Area float32   `json:"Area"`
}

// OscEvent is an OSC message
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
	pitchOffset int32
}

type CursorManager struct {
	executeMutex sync.RWMutex
	ClickMutex   sync.Mutex

	// activeMutex   sync.RWMutex
	activeMutex   sync.Mutex
	activeCursors map[int]*ActiveCursor // map of Gid to ActiveCursor

	GidToLoopedGidMutex sync.RWMutex
	GidToLoopedGid      map[int]int
	handlers            map[string]CursorHandler
	uniqueInt           int
	uniqueMutex         sync.Mutex
	LoopThreshold       float32
}

type CursorPos struct {
	X, Y, Z float32
}

// CursorDeviceCallbackFunc xxx
type CursorCallbackFunc func(e CursorEvent)

type CursorHandler interface {
	onCursorEvent(state ActiveCursor) error
}

func NewCursorEvent(gid int, tag string, ddu string, pos CursorPos) CursorEvent {
	ce := CursorEvent{
		CClick: CurrentClick(),
		Gid:    gid,
		Tag:    tag,
		Ddu:    ddu,
		Pos:    pos,
		Area:   0,
	}
	return ce
}

func NewCursorClearEvent() CursorEvent {
	gid := TheCursorManager.UniqueGid()
	return NewCursorEvent(gid, "", "clear", CursorPos{})
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

	forceOverride, _ := GetParamBool("engine.looping_override")
	if forceOverride {
		forcebeats, _ := GetParamInt("engine.looping_beats")
		ac.loopBeats = forcebeats
		forcefade, _ := GetParamFloat("engine.looping_fade")
		ac.loopFade = float32(forcefade)
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
		GidToLoopedGid: map[int]int{},
		activeMutex:    sync.Mutex{},
		handlers:       map[string]CursorHandler{},
		uniqueInt:      1,
		LoopThreshold:  float32(0.01),
	}
	return cm
}

func (cm *CursorManager) UniqueGid() int {
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

		currClick := ac.Current.GetClick()
		prevClick := ac.Previous.GetClick()
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

func (cm *CursorManager) GenerateRandomGesture(tags string, dur time.Duration) {

	pos0 := RandPos()
	pos1 := RandPos()
	// Occasionally force horizontal and vertical
	if TheRand.Int()%4 == 0 {
		pos1.X = pos0.X
	} else if TheRand.Int()%4 == 0 {
		pos1.Y = pos0.Y
	}
	cm.GenerateGesture(tags, dur, pos0, pos1)
}

func RandPos() CursorPos {
	return CursorPos{
		X: TheRand.Float32(),
		Y: TheRand.Float32(),
		Z: TheRand.Float32(),
	}
}

/*
func randDir() float32 {
	retVals := [3]float32{-1.0, 0.0, 1.0}
	r := retVals[TheRand.Int() % 3]
	return r
}

func dirFrom(x0, x1 float32) float32 {
	d := x1 - x0
	if d > 0 {
		return 1
	} else if d < 0 {
		return -1
	} else {
		return 0
	}
}
*/

func (cm *CursorManager) GenerateGesture(tag string, dur time.Duration, pos0 CursorPos, pos1 CursorPos) {

	gid := cm.UniqueGid()
	LogOfType("generategesture", "generateCursoresture start",
		"gid", gid, "noteDuration", dur, "tags", tag, "pos0", pos0, "pos1", pos1)

	nsteps := 1
	/*
		nsteps, err := GetParamInt("engine.gesturesteps")
		if err != nil {
			LogIfError(err)
			return
		}
	*/

	dpos := CursorPos{
		X: pos1.X - pos0.X,
		Y: pos1.Y - pos0.Y,
		Z: pos1.Z - pos0.Z,
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

		// Not sure about this Lock
		// cm.activeMutex.Lock()
		amount := float32(n) / float32(nsteps)
		pos := CursorPos{
			X: pos0.X + dpos.X*amount,
			Y: pos0.Y + dpos.Y*amount,
			Z: pos0.Z + dpos.Z*amount,
		}
		ce := NewCursorEvent(gid, tag, ddu, pos)
		// LogOfType("cursor", "generateCursoresture", "n", n, "amount", amount, "pos", pos)
		// cm.activeMutex.Unlock()

		cm.ExecuteCursorEvent(ce)
		time.Sleep(time.Duration(dur.Nanoseconds() / int64(nsteps)))
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
	TheCursorManager.ClickMutex.Lock()
	clk := ce.CClick
	TheCursorManager.ClickMutex.Unlock()
	return clk
}

/*
Keep track of names attached to each cursor "down",
and use those names on subsequent "drag" and "up" events.
*/
func (cm *CursorManager) LoopedGidFor(ce CursorEvent, warn bool) int {

	cm.GidToLoopedGidMutex.Lock()
	defer cm.GidToLoopedGidMutex.Unlock()

	loopedGid, ok := cm.GidToLoopedGid[ce.Gid]
	if ok {
		// LogInfo("LoopedGidFor FOUND", "ce.Gid", ce.Gid, "loopedGid", loopedGid)
		return loopedGid
	}
	switch ce.Ddu {
	case "down":
		loopedGid = cm.UniqueGid()
		cm.GidToLoopedGid[ce.Gid] = loopedGid
		// LogInfo("CursorManager.LoopedGidFor down", "original gid", ce.Gid, "loopedGid", loopedGid)
	case "drag":
		loopedGid = cm.UniqueGid()
		cm.GidToLoopedGid[ce.Gid] = loopedGid
		// LogInfo("CursorManager.LoopedGidFor drag", "original gid", ce.Gid, "loopedGid", loopedGid)
	case "up":
		// Not totally sure why this happens
		if warn {
			LogWarn("Why is this happening?")
		}
		// loopedGid = 0
		loopedGid = cm.UniqueGid()
		cm.GidToLoopedGid[ce.Gid] = loopedGid
	}
	// LogInfo("CursorManager.LoopedGidFor up", "original gid", ce.Gid, "loopedGid", loopedGid)
	return loopedGid
}

func (cm *CursorManager) ExecuteCursorEvent(ce CursorEvent) {

	TheEngine.RecordCursorEvent(ce)

	fadeThreshold, err := GetParamFloat("engine.looping_fadethreshold")
	if err != nil {
		LogIfError(err)
	} else {
		TheCursorManager.LoopThreshold = float32(fadeThreshold)
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

	LogOfType("cursor", "ExecuteCursorEvent", "gid", ce.Gid, "ddu", ce.Ddu, "x", ce.Pos.X, "y", ce.Pos.Y, "z", ce.Pos.Z)

	ac, ok := cm.getActiveCursorFor(ce.Gid)
	if !ok {
		// new ActiveCursor
		// Make sure the first ddu is "down"
		if ce.Ddu != "down" {
			// LogWarn("handleDownDragUp: first ddu is not down", "gid", ce.Gid, "ddu", ce.Ddu)
			ce.Ddu = "down"
		}
		ac = NewActiveCursor(ce)
		if ac == nil {
			LogWarn("CursorManager.ExecuteCursorEvent - unable to create ActiveCursor", "ce", ce)
			return
		}

		cm.activeMutex.Lock()
		LogOfType("cursor", "ExecuteCursorEvent: adding new ActiveCursor", "gid", ce.Gid, "ac", ac)
		cm.activeCursors[ce.Gid] = ac
		cm.activeMutex.Unlock()

	} else {

		// LogOfType("cursor", "ExecuteCursorEvent: using existing ActiveCursor", "gid", ce.Gid, "ac", ac)
		// existing ActiveCursor
		cm.activeMutex.Lock()
		ac.Previous = ac.Current
		ac.Current = ce
		cm.activeMutex.Unlock()

	}

	ac.Current.SetClick(CurrentClick())

	if ac.loopIt {
		se := cm.LoopCursorEvent(ac)
		if se != nil {
			if ac.Current.Ddu == "up" {
				LogOfType("cursor","UP cursor", "maxZ", ac.maxZ)
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
		LogOfType("cursor", "handleDownDragUp up is deleting gid", "gid", ce.Gid, "ddu", ce.Ddu)
		cm.DeleteActiveCursorIfZLessThan(ce.Gid, cm.LoopThreshold)
	}
}

func (cm *CursorManager) LoopCursorEvent(ac *ActiveCursor) *SchedElement {
	loopce := ac.Current
	// the looped CursorEvent starts out as a copy of the ActiveCursor's Current value
	loopce.SetClick(CurrentClick() + OneBeat*Clicks(ac.loopBeats))

	// The looped CursorEvents should have unique gid val,ues.
	loopce.Gid = TheCursorManager.LoopedGidFor(ac.Current, true /*warn*/)
	if loopce.Gid == 0 {
		LogWarn("HEY!!! loopIt LoopedGidFor returns 0?")
		return nil
	}
	// LogInfo("ac.loopIt LoopedGidFor", "loopce.Gid", loopce.Gid)

	// Fade the Z value
	newZ := loopce.Pos.Z * ac.loopFade
	LogOfType("loop","loopcd.Z faded", "origZ", loopce.Pos.Z, "newZ", newZ, "loopFade", ac.loopFade)
	loopce.Pos.Z = newZ

	if loopce.Pos.Z < cm.LoopThreshold && loopce.Ddu != "up" {
		LogOfType("loop","loopce.Z is small, NOT LOOPING IT", "loopce", loopce)
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
func (cm *CursorManager) DeleteActiveCursorIfZLessThan(gid int, threshold float32) {

	// LogInfo("BEGIN DeleteActiveCursorIfZ 1", "gid", gid,"sched", TheScheduler.ToString())

	cm.activeMutex.Lock()
	ac, ok := cm.activeCursors[gid]
	loopGid := 0
	if !ok {
		// LogWarn("DeleteActiveCursor: gid not found in ActiveCursor", "gid", gid)
	} else {
		// LogInfo("DeleteActiveCursorIfZLessThan", "gid", gid, "threshold", threshold, "ac.maxZ", ac.maxZ)
		LogOfType("cursor","DeleteActiveCursorIfZLessThan", "maxZ", ac.maxZ, "threshold", threshold, "gid", ac.Current.Gid)
		if ac.maxZ < threshold {
			// we want to remove things that this ActiveCursor has created for looping.
			loopGid = cm.LoopedGidFor(ac.Current, false /*don't warn*/)
			if loopGid == 0 {
				LogWarn("HEY!!! in DeleteActiveCursorIfZLessThan LoopedGidFor returns 0?")
			} else {
				LogOfType("cursor","DeleteActiveCursorIfZLessThan deleting!!", "loopGid", loopGid)
				delete(cm.activeCursors, gid)
				delete(cm.activeCursors, loopGid)
				// LogInfo("DeleteActiveCursorIfZLessThan REMOVING", "loopGid", loopGid, "gid", gid, "ac.maxZ", ac.maxZ, "gid", gid)
			}
		}
	}
	cm.activeMutex.Unlock()

	if loopGid == 0 {
		return
	}

	// LogInfo("SHOULD BE SENDING NOTEOFF FOR THIS ACTIVE CURSOR?", "gid", gid, "ac", ac, "ac.NoteOn", ac.NoteOn)

	cm.GidToLoopedGidMutex.Lock()
	delete(cm.GidToLoopedGid, gid)
	cm.GidToLoopedGidMutex.Unlock()

	if loopGid != 0 {
		TheScheduler.DeleteCursorEventsWhoseGidIs(loopGid)
		TheScheduler.DeleteCursorEventsWhoseGidIs(gid)
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
	msg.Append(fmt.Sprintf("%d", ce.Gid))
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
	gid := cm.UniqueGid()
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
