package engine

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hypebeast/go-osc/osc"
)

var TheCursorManager *CursorManager

// CursorEvent is a singl Cursor event
type CursorEvent struct {
	Click *atomic.Int64 `json:"Click"`
	Gid   int           `json:"Gid"`
	Tag   string        `json:"Tag"`
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

	activeMutex   sync.RWMutex
	activeCursors map[int]*ActiveCursor // map of Gid to ActiveCursor

	GidToLoopedGidMutex sync.RWMutex
	GidToLoopedGid      map[int]int
	handlers            map[string]CursorHandler
	uniqueInt           int
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
		Click: &atomic.Int64{},
		Gid:   gid,
		Tag:   tag,
		Ddu:   ddu,
		Pos:   pos,
		Area:  0,
	}
	click := int64(CurrentClick())
	if click == 0 {
		LogWarn("NewCursorEvent: click is 0?")
	}
	ce.Click.Store(int64(CurrentClick()))
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
	return &CursorManager{
		activeCursors:  map[int]*ActiveCursor{},
		GidToLoopedGid: map[int]int{},
		activeMutex:    sync.RWMutex{},
		handlers:       map[string]CursorHandler{},
		uniqueInt:      1,
	}
}

func (cm *CursorManager) UniqueGid() int {
	cm.activeMutex.Lock()
	defer cm.activeMutex.Unlock()
	unique := cm.uniqueInt
	cm.uniqueInt++
	return unique
}

func (cm *CursorManager) getActiveCursorFor(gid int) (*ActiveCursor, bool) {
	cm.activeMutex.RLock()
	ac, ok := cm.activeCursors[gid]
	cm.activeMutex.RUnlock()
	return ac, ok
}

func (cm *CursorManager) AddCursorHandler(name string, handler CursorHandler, sources ...string) {
	cm.handlers[name] = handler
}

// cleaerCursors sends "up" events to all active cursors, then deletes them
func (cm *CursorManager) clearActiveCursors(tag string) {

	currentClick := CurrentClick()
	gidsToDelete := []int{}

	cm.activeMutex.RLock()

	for gid, ac := range cm.activeCursors {
		if ac.Current.Tag != tag {
			LogInfo("clearActiveCursors is NOT clearing", "ac.Current", ac.Current)
			continue
		}
		ce := ac.Current
		ce.SetClick(currentClick)
		ce.Ddu = "up"

		cm.activeMutex.RUnlock()
		cm.ExecuteCursorEvent(ce)
		cm.activeMutex.RLock()

		LogOfType("cursor", "Clearing cursor", "gid", gid)

		gidsToDelete = append(gidsToDelete, gid)
	}

	cm.activeMutex.RUnlock()

	cm.deleteActiveCursors(gidsToDelete)
}

func (cm *CursorManager) GenerateRandomGesture(tags string, dur time.Duration) {

	pos0 := RandPos()
	pos1 := RandPos()
	// Occasionally force horizontal and vertical
	if rand.Int()%4 == 0 {
		pos1.X = pos0.X
	} else if rand.Int()%4 == 0 {
		pos1.Y = pos0.Y
	}
	cm.GenerateGesture(tags, dur, pos0, pos1)
}

func RandPos() CursorPos {
	return CursorPos{
		X: rand.Float32(),
		Y: rand.Float32(),
		Z: rand.Float32(),
	}
}

/*
func randDir() float32 {
	retVals := [3]float32{-1.0, 0.0, 1.0}
	r := retVals[rand.Int() % 3]
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
	LogOfType("cursor", "generateCursoresture start",
		"cid", gid, "noteDuration", dur, "tags", tag, "pos0", pos0, "pos1", pos1)

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
		cm.activeMutex.Lock()
		amount := float32(n) / float32(nsteps)
		pos := CursorPos{
			X: pos0.X + dpos.X*amount,
			Y: pos0.Y + dpos.Y*amount,
			Z: pos0.Z + dpos.Z*amount,
		}
		ce := NewCursorEvent(gid, tag, ddu, pos)
		// LogOfType("cursor", "generateCursoresture", "n", n, "amount", amount, "pos", pos)
		cm.activeMutex.Unlock()

		cm.ExecuteCursorEvent(ce)
		time.Sleep(time.Duration(dur.Nanoseconds() / int64(nsteps)))
	}
}

func (ce CursorEvent) SetClick(click Clicks) {
	if ce.Click == nil {
		LogWarn("Hey, click is null in CursorEvent.SetClick?")
		ce.Click = &atomic.Int64{}
	}
	ce.Click.Store(int64(click))
}
func (ce CursorEvent) GetClick() Clicks {
	if ce.Click == nil {
		LogWarn("Hey, click is null in CursorEvent.SetClick?")
		ce.Click = &atomic.Int64{}
	}
	return Clicks(ce.Click.Load())
}

/*
Keep track of names attached to each cursor "down",
and use those names on subsequent "drag" and "up" events.
*/
func (cm *CursorManager) LoopedGidFor(ce CursorEvent, warn bool) int {

	cm.GidToLoopedGidMutex.Lock()
	defer cm.GidToLoopedGidMutex.Unlock()

	loopedGid, ok := cm.GidToLoopedGid[ce.Gid]
	if !ok {
		if ce.Ddu == "up" { // Not totally sure why this happens
			if warn {
				LogWarn("Why is this happening?")
			}
			loopedGid = cm.UniqueGid()
			return loopedGid
		}
		// loopedCid = fmt.Sprintf("%s#%d", ce.Source(), cm.UniqueInt())
		loopedGid = cm.UniqueGid()
		cm.GidToLoopedGid[ce.Gid] = loopedGid
	}
	return loopedGid
}

const LoopFadeZThreshold = 0.001

func (cm *CursorManager) ExecuteCursorEvent(ce CursorEvent) {

	TheEngine.RecordCursorEvent(ce)

	if ce.Ddu == "clear" {
		if ce.Tag == "" {
			LogWarn("CursorManager.ExecuteCursorEvent: clear with empty tag?")
		}
		TheCursorManager.clearActiveCursors(ce.Tag)
		return
	}

	// Don't put lock above clearActiveCursors(), since it calls ExecuteCursorEvent recursively
	cm.executeMutex.Lock()
	defer cm.executeMutex.Unlock()

	if ce.GetClick() == 0 {
		ce.SetClick(CurrentClick())
	}

	LogOfType("cursor", "ExecuteCursorEvent", "ce", ce)

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

	ac.Current.Click.Store(int64(CurrentClick()))

	if ac.loopIt {

		// the looped CursorEvent starts out as a copy of the ActiveCursor's Current value
		loopce := ac.Current
		loopce.SetClick(CurrentClick() + OneBeat*Clicks(ac.loopBeats))

		// The looped CursorEvents should have unique gid val,ues.
		loopce.Gid = TheCursorManager.LoopedGidFor(ac.Current, true /*warn*/)

		// Fade the Z value
		loopce.Pos.Z = loopce.Pos.Z * ac.loopFade
		// LogInfo("loopcd.Z is now", "Z", loopce.Z)
		if loopce.Pos.Z > ac.maxZ {
			ac.maxZ = loopce.Pos.Z
		}

		// LogInfo("looped CursorEvent", "ce.Z", ce.Z, "loopFade", ac.loopFade)
		se := NewSchedElement(loopce.GetClick(), ce.Tag, loopce)
		TheScheduler.insertScheduleElement(se)
	}

	TheEngine.sendToOscClients(CursorToOscMsg(ce))

	LogOfType("cursor", "CursorManager.handleDownDragUp before doing handlers", "ce", ce)
	for _, handler := range cm.handlers {
		err := handler.onCursorEvent(*ac)
		LogIfError(err)
	}

	LogOfType("cursor", "CursorManager.handleDownDragUp", "ce", ce)

	if ce.Ddu == "up" {
		// LogOfType("cursor", "handleDownDragUp up is deleting cid", "cid", ce.Cid, "ddu", ce.Ddu)
		cm.DeleteActiveCursorIfZLessThan(ce.Gid, LoopFadeZThreshold)
	}
}

func (cm *CursorManager) DeleteActiveCursor(gid int) {
	// reuse this routine, but mke sure that it will
	cm.DeleteActiveCursorIfZLessThan(gid, 1.0)
}

// Set threshold to 1.0 (or greater) if you want to
func (cm *CursorManager) DeleteActiveCursorIfZLessThan(gid int, threshold float32) {

	cm.activeMutex.Lock()
	ac, ok := cm.activeCursors[gid]
	loopGid := 0
	if !ok {
		LogWarn("DeleteActiveCursor: gid not found in ActiveCursor", "gid", gid)
	} else {
		// LogInfo("DeleteActiveCursorIfZLessThan", "gid", gid, "threshold", threshold, "ac.maxZ", ac.maxZ)
		if ac.maxZ < threshold {
			// we want to remove things that this ActiveCursor has created for looping.
			cm.activeMutex.Unlock()
			loopGid = cm.LoopedGidFor(ac.Current, false /*don't warn*/)
			LogInfo("DeleteActiveCursorIfZLessThan REMOVING", "loopCid", loopGid, "ac.maxZ", ac.maxZ)
			cm.activeMutex.Lock()
			// but wait after we detete it from
		}
	}
	delete(cm.activeCursors, gid)
	cm.activeMutex.Unlock()

	cm.GidToLoopedGidMutex.Lock()
	delete(cm.GidToLoopedGid, gid)
	cm.GidToLoopedGidMutex.Unlock()

	if loopGid != 0 {
		LogInfo("Calling DeleteEventsWhoseGidIs", "loopCid", loopGid)
		TheScheduler.DeleteEventsWhoseGidIs(loopGid)
	}
}

func (cm *CursorManager) DeleteActiveCursorsForTag(tag string) {

	// First construct the list
	cm.activeMutex.RLock()

	if len(cm.activeCursors) > 0 {
		LogOfType("cursor", "DeleteActiveCursorsWithTag", "tag", tag)
	}
	todelete := []int{}
	for _, ac := range cm.activeCursors {
		if ac.Current.Tag == tag {
			todelete = append(todelete, ac.Current.Gid)
		}
	}
	cm.activeMutex.RUnlock()

	for _, cid := range todelete {
		cm.DeleteActiveCursor(cid)
		TheScheduler.DeleteEventsWhoseGidIs(cid)
	}
}

func CursorToOscMsg(ce CursorEvent) *osc.Message {
	msg := osc.NewMessage("/cursor")
	msg.Append(ce.Ddu)
	msg.Append(int32(ce.Gid))
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

func (cm *CursorManager) autoCursorUp(now time.Time) {

	// checkDelay is the number of CLicks that has to pass
	// before we decide a cursor is no longer present,
	// resulting in an "up" event.
	checkDelay := 8 * OneBeat

	cm.activeMutex.RLock()
	var gidsToDelete = []int{}
	for gid, ac := range cm.activeCursors {

		dclick := ac.Current.GetClick() - ac.Previous.GetClick()
		if dclick > checkDelay {
			ce := ac.Current
			ce.Ddu = "up"
			LogInfo("autoCursorUp: Executing autoCursorUp!", "gid", gid, "ce", ce)

			cm.ExecuteCursorEvent(ce)
			gidsToDelete = append(gidsToDelete, gid)
		}
	}
	cm.activeMutex.RUnlock()

	cm.deleteActiveCursors(gidsToDelete)

}

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
