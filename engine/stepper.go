package engine

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hypebeast/go-osc/osc"
	"github.com/vizicist/portmidi"
)

const defaultClicksPerSecond = 192
const minClicksPerSecond = (defaultClicksPerSecond / 16)
const maxClicksPerSecond = (defaultClicksPerSecond * 16)

var defaultSynth = "P_01_C_01"
var loopForever = 999999
var uniqueIndex = 0

// CurrentMilli is the time from the start, in milliseconds
var CurrentMilli int

var currentMilliOffset int
var currentClickOffset Clicks
var clicksPerSecond int
var currentClick Clicks
var oneBeat Clicks

// TempoFactor xxx
var TempoFactor = float64(1.0)

// ActiveNote is a currently active MIDI note
type ActiveNote struct {
	id     int
	noteOn *Note
}

// Stepper is an entity that that reacts to things (cursor events, apis) and generates output (midi, graphics)
type Stepper struct {
	padName         string
	resolumeLayer   int // 1,2,3,4
	freeframeClient *osc.Client
	resolumeClient  *osc.Client
	guiClient       *osc.Client
	lastActiveID    int

	// tempoFactor            float64
	loop                   *StepLoop
	loopIsRecording        bool
	loopIsPlaying          bool
	fadeLoop               float32
	lastCursorStepEvent    CursorStepEvent
	lastUnQuantizedStepNum Clicks

	// params                         map[string]interface{}
	params *ParamValues

	paramsMutex               sync.RWMutex
	activeNotes               map[string]*ActiveNote
	activeNotesMutex          sync.RWMutex
	activeCursors             map[string]*ActiveStepCursor
	activeCursorsMutex        sync.RWMutex
	permInstanceIDMutex       sync.RWMutex
	permInstanceIDQuantized   map[string]string // map incoming IDs to permInstanceIDs in the steploop
	permInstanceIDUnquantized map[string]string // map incoming IDs to permInstanceIDs in the steploop
	permInstanceIDDownClick   map[string]Clicks // map permInstanceIDs to quantized stepnum of the down event
	permInstanceIDDownQuant   map[string]Clicks // map permInstanceIDs to quantize value of the "down" event
	permInstanceIDDragOK      map[string]bool
	deviceCursors             map[string]*DeviceCursor
	deviceCursorsMutex        sync.RWMutex

	activePhrasesManager *ActivePhrasesManager

	// Things moved over from Router
	MIDINumDown      int
	MIDIOctaveShift  int
	MIDIThru         string // "disabled", "thru", etc
	MIDIQuantized    bool
	useExternalScale bool // if true, scadjust uses "external" Scale
	TransposePitch   int
	midiInputMutex   sync.RWMutex
	externalScale    *Scale
}

// NewStepper makes a new Stepper
func NewStepper(pad string, resolumeLayer int, freeframeClient *osc.Client, resolumeClient *osc.Client, guiClient *osc.Client) *Stepper {
	r := &Stepper{
		padName:         pad,
		resolumeLayer:   resolumeLayer,
		freeframeClient: freeframeClient,
		resolumeClient:  resolumeClient,
		guiClient:       guiClient,
		// tempoFactor:         1.0,
		// params:                         make(map[string]interface{}),
		params:                    NewParamValues(),
		activeNotes:               make(map[string]*ActiveNote),
		activeCursors:             make(map[string]*ActiveStepCursor),
		permInstanceIDQuantized:   make(map[string]string),
		permInstanceIDUnquantized: make(map[string]string),
		permInstanceIDDownClick:   make(map[string]Clicks),
		permInstanceIDDownQuant:   make(map[string]Clicks),
		permInstanceIDDragOK:      make(map[string]bool),
		fadeLoop:                  0.5,
		loop:                      NewLoop(oneBeat * 4),
		deviceCursors:             make(map[string]*DeviceCursor),
		activePhrasesManager:      NewActivePhrasesManager(),

		MIDIOctaveShift:  0,
		MIDIThru:         "thru",
		useExternalScale: false,
		MIDIQuantized:    false,
		TransposePitch:   0,
	}
	r.params.SetDefaultValues()
	r.ClearExternalScale()
	r.SetExternalScale(60%12, true) // Middle C

	log.Printf("NewStepper: pad=%s resolumeLayer=%d\n", pad, resolumeLayer)
	return r
}

// Time returns the current time
func (r *Stepper) Time() time.Time {
	return time.Now()
}

func (r *Stepper) handleCursorDeviceEvent(e CursorDeviceEvent) {

	id := e.NUID + "." + e.CID

	if e.DownDragUp == "drag" {
		// log.Printf("handleCursorDeviceEvent: Ignoring drag\n")
		return
	}
	r.deviceCursorsMutex.Lock()
	defer r.deviceCursorsMutex.Unlock()

	tc, ok := r.deviceCursors[id]
	if !ok {
		// new DeviceCursor
		tc = &DeviceCursor{}
		r.deviceCursors[id] = tc
	}
	tc.lastTouch = r.Time()

	// If it's a new (downed==false) cursor, make sure the first step event is "down
	if !tc.downed {
		e.DownDragUp = "down"
		tc.downed = true
	}
	cse := CursorStepEvent{
		ID:         id,
		X:          e.X,
		Y:          e.Y,
		Z:          e.Z,
		Downdragup: e.DownDragUp,
	}
	if DebugUtil.Cursor {
		log.Printf("Stepper.handleCursorDeviceEvent: pad=%s e=%+v cse=%+v\n", r.padName, e, cse)
	}

	r.executeIncomingCursor(cse)

	if e.DownDragUp == "up" {
		if DebugUtil.Cursor {
			log.Printf("Router.handleCursorDeviceEvent: deleting cursor id=%s\n", id)
		}
		delete(r.deviceCursors, id)
	}
}

// checkDelay is the Duration that has to pass
// before we decide a cursor is no longer present,
// resulting in a cursor UP event.
const checkDelay time.Duration = 2 * time.Second

func (r *Stepper) checkCursorUp() {
	now := r.Time()

	r.deviceCursorsMutex.Lock()
	defer r.deviceCursorsMutex.Unlock()

	for id, c := range r.deviceCursors {
		elapsed := now.Sub(c.lastTouch)
		if elapsed > checkDelay {
			r.executeIncomingCursor(CursorStepEvent{ID: id, Downdragup: "up"})
			if DebugUtil.Cursor {
				log.Printf("Stepper.checkCursorUp: deleting cursor id=%s elapsed=%v\n", id, elapsed)
			}
			delete(r.deviceCursors, id)
		}
	}
}

func (r *Stepper) getActiveNote(id string) *ActiveNote {
	r.activeNotesMutex.RLock()
	a, ok := r.activeNotes[id]
	r.activeNotesMutex.RUnlock()
	if !ok {
		r.lastActiveID++
		a = &ActiveNote{id: r.lastActiveID}
		r.activeNotesMutex.Lock()
		r.activeNotes[id] = a
		r.activeNotesMutex.Unlock()
	}
	return a
}

func (r *Stepper) getActiveStepCursor(ce CursorStepEvent) *ActiveStepCursor {
	sid := ce.ID
	r.activeCursorsMutex.RLock()
	ac, ok := r.activeCursors[sid]
	r.activeCursorsMutex.RUnlock()
	if !ok {
		ac = &ActiveStepCursor{downEvent: ce}
		r.activeCursorsMutex.Lock()
		r.activeCursors[sid] = ac
		r.activeCursorsMutex.Unlock()
	}
	return ac
}

func (r *Stepper) terminateActiveNotes() {
	r.activeNotesMutex.RLock()
	for id, a := range r.activeNotes {
		// log.Printf("terminateActiveNotes n=%v\n", a.currentNoteOn)
		if a != nil {
			r.sendNoteOff(a)
		} else {
			log.Printf("Hey, activeNotes entry for id=%s\n", id)
		}
	}
	r.activeNotesMutex.RUnlock()
}

func (r *Stepper) clearGraphics() {
	// send an OSC message to Resolume
	r.toFreeFramePluginForLayer(osc.NewMessage("/clear"))
}

func (r *Stepper) publishSprite(id string, x, y, z float32) {
	err := PublishSpriteEvent(x, y, z)
	if err != nil {
		log.Printf("publishSprite: err=%s\n", err)
	}
}

func (r *Stepper) generateSprite(id string, x, y, z float32) {
	if !TheRouter().generateVisuals {
		return
	}
	// send an OSC message to Resolume
	msg := osc.NewMessage("/sprite")
	msg.Append(x)
	msg.Append(y)
	msg.Append(z)
	msg.Append(id)
	r.toFreeFramePluginForLayer(msg)
}

func (r *Stepper) generateVisualsFromCursor(ce CursorStepEvent) {
	if !TheRouter().generateVisuals {
		return
	}
	// send an OSC message to Resolume
	msg := osc.NewMessage("/cursor")
	msg.Append(ce.Downdragup)
	msg.Append(ce.ID)
	msg.Append(float32(ce.X))
	msg.Append(float32(ce.Y))
	msg.Append(float32(ce.Z))
	if DebugUtil.GenVisual {
		log.Printf("Stepper.generateVisuals: pad=%s click=%d stepnum=%d OSC message = %+v\n", r.padName, currentClick, r.loop.currentStep, msg)
	}
	r.toFreeFramePluginForLayer(msg)
}

func (r *Stepper) generateSpriteFromNote(a *ActiveNote) {

	n := a.noteOn
	if n.TypeOf != NOTEON {
		return
	}

	// send an OSC message to Resolume
	oscaddr := "/sprite"

	// The first argument is a relative pitch amoun (0.0 to 1.0) within its range
	pitchmin := uint8(r.params.ParamIntValue("sound.pitchmin"))
	pitchmax := uint8(r.params.ParamIntValue("sound.pitchmax"))
	if n.Pitch < pitchmin || n.Pitch > pitchmax {
		log.Printf("Unexpected value of n.Pitch=%d, not between %d and %d\n", n.Pitch, pitchmin, pitchmax)
		return
	}

	var x float32
	var y float32
	switch r.params.ParamStringValue("visual.placement", "random") {
	case "random":
		x = rand.Float32()
		y = rand.Float32()
	case "linear":
		y = 0.5
		x = float32(n.Pitch-pitchmin) / float32(pitchmax-pitchmin)
	case "cursor":
		x = rand.Float32()
		y = rand.Float32()
	default:
		x = rand.Float32()
		y = rand.Float32()
	}

	msg := osc.NewMessage(oscaddr)
	msg.Append(x)
	msg.Append(y)
	msg.Append(float32(n.Velocity) / 127.0)
	// Someday localhost should be changed to the actual IP address
	msg.Append(fmt.Sprintf("%d@localhost", a.id))
	// log.Printf("generateSprite msg=%+v\n", msg)
	r.toFreeFramePluginForLayer(msg)
}

func (r *Stepper) notifyGUI(ce CursorStepEvent, wasFresh bool) {
	if !ConfigBool("notifygui") {
		return
	}
	// send an OSC message to GUI
	msg := osc.NewMessage("/notify")
	msg.Append(ce.Downdragup)
	msg.Append(ce.ID)
	msg.Append(float32(ce.X))
	msg.Append(float32(ce.Y))
	msg.Append(float32(ce.Z))
	msg.Append(int32(r.resolumeLayer))
	msg.Append(wasFresh)
	r.guiClient.Send(msg)
	if DebugUtil.Notify {
		log.Printf("Stepper.notifyGUI: msg=%v\n", msg)
	}
}

func (r *Stepper) toFreeFramePluginForLayer(msg *osc.Message) {
	r.freeframeClient.Send(msg)
	if DebugUtil.OSC {
		log.Printf("Stepper.toFreeFramePlugin: layer=%d msg=%v\n", r.resolumeLayer, msg)
	}
}

func (r *Stepper) toResolume(msg *osc.Message) {
	r.resolumeClient.Send(msg)
	if DebugUtil.OSC || DebugUtil.Resolume {
		log.Printf("Stepper.toResolume: msg=%v\n", msg)
	}
}

// ClearExternalScale xxx
func (r *Stepper) ClearExternalScale() {
	r.externalScale = makeScale()
}

// SetExternalScale xxx
func (r *Stepper) SetExternalScale(pitch int, on bool) {
	s := r.externalScale
	for p := pitch; p < 128; p += 12 {
		s.hasNote[p] = on
	}
}

func (r *Stepper) handleMIDISetScaleNote(e portmidi.Event) {
	status := e.Status & 0xf0
	pitch := int(e.Data1)
	if status == 0x90 {
		// If there are no notes held down (i.e. this is the first), clear the scale
		if r.MIDINumDown < 0 {
			// this can happen when there's a Read error that misses a NOTEON
			r.MIDINumDown = 0
		}
		if r.MIDINumDown == 0 {
			r.ClearExternalScale()
		}
		r.SetExternalScale(pitch%12, true)
		r.MIDINumDown++
		if pitch < 60 {
			r.MIDIOctaveShift = -1
		} else if pitch > 72 {
			r.MIDIOctaveShift = 1
		} else {
			r.MIDIOctaveShift = 0
		}
	} else if status == 0x80 {
		r.MIDINumDown--
	}
}

// HandleMIDITimeReset xxx
func (r *Stepper) HandleMIDITimeReset() {
	log.Printf("HandleMIDITimeReset!! needs implementation\n")
}

// HandleMIDIDeviceInput xxx
func (r *Stepper) HandleMIDIDeviceInput(e portmidi.Event) {

	r.midiInputMutex.Lock()
	defer r.midiInputMutex.Unlock()

	if DebugUtil.MIDI {
		log.Printf("Router.HandleMIDIDeviceInput: MIDIInput event=%+v\n", e)
	}
	switch r.MIDIThru {
	case "":
		// do nothing
	case "disabled":
		// do nothing
	case "setscale":
		r.handleMIDISetScaleNote(e)
	case "thru":
		r.PassThruMIDI(e, false)
	case "thruscadjust":
		r.PassThruMIDI(e, true)
	default:
		log.Printf("Router.HandleMIDIDeviceInput: unknown MIDIThru value=%s\n", r.MIDIThru)
	}
}

// getScale xxx
func (r *Stepper) getScale() *Scale {
	var scaleName string
	var scale *Scale
	if r.useExternalScale {
		scale = r.externalScale
	} else {
		scaleName = r.params.ParamStringValue("misc.scale", "newage")
		scale = GlobalScale(scaleName)
	}
	return scale
}

// PassThruMIDI xxx
func (r *Stepper) PassThruMIDI(e portmidi.Event, scadjust bool) {

	// log.Printf("Stepper.PassThruMIDI e=%+v\n", e)

	// channel on incoming MIDI is ignored
	// it uses whatever sound the Stepper is using
	status := e.Status & 0xf0

	data1 := uint8(e.Data1)
	data2 := uint8(e.Data2)
	pitch := data1

	synth := r.params.ParamStringValue("sound.synth", defaultSynth)
	var n *Note
	if (status == 0x90 || status == 0x80) && scadjust == true {
		scale := r.getScale()
		pitch = scale.ClosestTo(pitch)
	}
	switch status {
	case 0x90:
		n = NewNoteOn(pitch, data2, synth)
	case 0x80:
		n = NewNoteOff(pitch, data2, synth)
	case 0xB0:
		n = NewController(data1, data2, synth)
	case 0xC0:
		n = NewProgChange(data1, data2, synth)
	case 0xD0:
		n = NewChanPressure(data1, data2, synth)
	case 0xE0:
		n = NewPitchBend(data1, data2, synth)
	default:
		log.Printf("PassThruMIDI unable to handle status=%02x\n", status)
		return
	}
	if n != nil {
		// log.Printf("PassThruMIDI sending note=%s\n", n)
		if DebugUtil.MIDI {
			log.Printf("MIDI.SendNote: n=%+v\n", *n)
		}
		SendNoteToSynth(n)
	}
}

func (r *Stepper) generateSoundFromCursor(ce CursorStepEvent) {
	if !TheRouter().generateSound {
		return
	}
	// if DebugUtil.GenSound {
	// 	log.Printf("Stepper.generateSound: pad=%s activeNotes=%d ce=%+v\n", r.padName, len(r.activeNotes), ce)
	// }
	a := r.getActiveNote(ce.ID)
	switch ce.Downdragup {
	case "down":
		// Send NOTEOFF for current note
		if a.noteOn != nil {
			// I think this happens if we get things coming in
			// faster than the checkDelay can generate the UP event.
			log.Printf("Unexpected down when currentNoteOn is non-nil!? currentNoteOn=%+v\n", a)
			// log.Printf("generateMIDI sending NoteOff before down note\n")
			r.sendNoteOff(a)
		}
		a.noteOn = r.cursorToNoteOn(ce)
		// log.Printf("r=%s down Setting currentNoteOn to %v!\n", r.padName, *(a.currentNoteOn))
		// log.Printf("generateMIDI sending NoteOn for down\n")
		r.sendNoteOn(a)
	case "drag":
		log.Printf("generateSound: shouldn't be seeing drag\n")
		if a.noteOn == nil {
			// if we turn on playing in the middle of an existing loop,
			// we may see some drag events without a down.
			// Also, I'm seeing this pretty commonly in other situations,
			// not really sure what the underlying reason is,
			// but it seems to be harmless at the moment.
			log.Printf("=============== HEY! drag event, a.currentNoteOn == nil?\n")
		} else {
			// log.Printf("generateMIDI sending NoteOff for previous note\n")
			// r.sendNoteOff(a)
			log.Printf("generateMIDI NOT sending NoteOff/NoteOn for previous note\n")
			break
		}
		a.noteOn = r.cursorToNoteOn(ce)
		// log.Printf("r=%s drag Setting currentNoteOn to %v!\n", r.padName, *(a.currentNoteOn))
		// log.Printf("generateMIDI sending NoteOn\n")
		r.sendNoteOn(a)
	case "up":
		if a.noteOn == nil {
			// not sure why this happens, yet
			log.Printf("r=%s Unexpected UP when currentNoteOn is nil?\n", r.padName)
		} else {
			// log.Printf("generateMIDI sending NoteOff for UP\n")
			r.sendNoteOff(a)

			/*
				// the same pitch to the same synth, but I tried fixing it and failed. So.
				if DebugUtil.MIDI {
					log.Printf("MIDI.SendANO: synth=%s\n", a.noteOn.Sound)
				}
				MIDI.SendANO(a.noteOn.Sound)
			*/

			a.noteOn = nil
			// log.Printf("r=%s UP Setting currentNoteOn to nil!\n", r.padName)
		}
		r.activeNotesMutex.Lock()
		delete(r.activeNotes, ce.ID)
		r.activeNotesMutex.Unlock()
	}
}

// StartPhrase xxx
func (r *Stepper) StartPhrase(p *Phrase, cid string) {
	r.activePhrasesManager.StartPhrase(p, "midiplaycid")
}

// AdvanceByOneClick advances time by 1 click in a StepLoop
func (r *Stepper) AdvanceByOneClick() {

	r.activePhrasesManager.AdvanceByOneClick()

	loop := r.loop

	loop.stepsMutex.Lock()
	defer loop.stepsMutex.Unlock()

	stepnum := loop.currentStep
	if DebugUtil.Advance {
		if stepnum%20 == 0 {
			log.Printf("advanceClickby1 start stepnum=%d\n", stepnum)
		}
	}

	step := loop.steps[stepnum]

	var removeCids []string
	if step != nil && step.events != nil && len(step.events) > 0 {
		for _, event := range step.events {

			ce := event.cursorStepEvent

			if DebugUtil.Advance {
				log.Printf("Stepper.advanceClickBy1: pad=%s stepnum=%d ce=%+v\n", r.padName, stepnum, ce)
			}

			ac := r.getActiveStepCursor(ce)
			ac.x = ce.X
			ac.y = ce.Y
			ac.z = ce.Z

			wasFresh := ce.Fresh

			// Freshly added things ALWAYS get played.
			playit := false
			if ce.Fresh || r.loopIsPlaying {
				playit = true
			}
			event.cursorStepEvent.Fresh = false
			ce.Fresh = false // needed?

			// Note that we fade the z values in CursorStepEvent, not ActiveStepCursor,
			// because ActiveStepCursor goes away when the gesture ends,
			// while CursorStepEvents in the loop stick around.
			event.cursorStepEvent.Z = event.cursorStepEvent.Z * r.fadeLoop // fade it
			event.cursorStepEvent.LoopsLeft--
			ce.LoopsLeft--

			minz := float32(0.001)

			// Keep track of the maximum z value for an ActiveCursor,
			// so we know (when get the UP) whether we should
			// delete this gesture.
			switch {
			case ce.Downdragup == "down":
				ac.maxz = -1.0
				ac.lastDrag = -1
			case ce.Downdragup == "drag" && ce.Z > ac.maxz:
				log.Printf("AdvanceByOneClick: shouldn't be seeing drag\n")
				// log.Printf("Saving ce.Z as ac.maxz = %v\n", ce.Z)
				ac.maxz = ce.Z
			default:
				// log.Printf("up!\n")
			}

			// See if this cursor should be removed
			if ce.Downdragup == "up" &&
				(ce.LoopsLeft < 0 || (ac.maxz > 0.0 && ac.maxz < minz) || (ac.maxz < 0.0 && ac.downEvent.Z < minz)) {

				removeCids = append(removeCids, ce.ID)
				// NOTE: playit should still be left true for this UP event
			} else {
				// Don't play any events in this step with an id that we're getting ready to remove
				for _, cid := range removeCids {
					if ce.ID == cid {
						playit = false
					}
				}
			}

			if playit {

				// XXX - eventually, a parameter should allow
				// either of MIDI or Graphics to be quantized,
				// but for the moment, it's hardcoded that
				// MIDI things are generated from quantized events
				// and Graphic things from unquantized events.

				// We actually get two virtually-identical events
				// for each incoming cursor event.  One is unquantized,
				// and one is time-quantized

				if ce.Quantized {
					// MIDI stuff
					if ce.Downdragup == "drag" {
						log.Printf("playit: shouldn't be seeing drag\n")
						dclick := currentClick - ac.lastDrag
						if ac.lastDrag < 0 || dclick >= oneBeat/32 {
							ac.lastDrag = currentClick
							if DebugUtil.GenSound {
								log.Printf("generateSoundFromCursor: downdragup=%s stepnum=%d ce.ID=%s\n", ce.Downdragup, stepnum, ce.ID)
							}
							r.generateSoundFromCursor(ce)
						}
					} else {
						if DebugUtil.GenSound {
							log.Printf("generateSoundFromCursor: downdragup=%s stepnum=%d ce.ID=%s\n", ce.Downdragup, stepnum, ce.ID)
						}
						r.generateSoundFromCursor(ce)
					}
				} else {
					// Graphics and GUI stuff
					ss := r.params.ParamStringValue("visual.spritesource", "")
					if ss == "cursor" {
						if DebugUtil.Loop {
							log.Printf("Stepper.advanceClickBy1: stepnum=%d generateVisuals ce=%+v\n", stepnum, ce)
						}
						r.generateVisualsFromCursor(ce)
					}
					r.notifyGUI(ce, wasFresh)
				}
			}
		}
	}
	if len(removeCids) > 0 {
		for _, removeID := range removeCids {
			for _, step := range loop.steps {
				// We want to delete all events from this step that have removeId

				// This method of deleting things from an array without
				// allocating a new array is found on this page:
				// https://vbauerster.github.io/2017/04/removing-items-from-a-slice-while-iterating-in-go/
				// log.Printf("Before deleting id=%s  events=%v\n", id, step.events)
				newevents := step.events[:0]
				for _, event := range step.events {
					if event.cursorStepEvent.ID != removeID {
						// Include this event
						newevents = append(newevents, event)
					}
				}
				step.events = newevents
			}
		}
	}

	loop.currentStep++
	if loop.currentStep >= loop.length {
		if DebugUtil.Loop {
			log.Printf("Stepper.AdvanceClickBy1: region=%s Loop wrapping around to step 0\n", r.padName)
		}
		loop.currentStep = 0
	}
}

func (r *Stepper) executeIncomingCursor(ce CursorStepEvent) {

	if DebugUtil.Cursor {
		log.Printf("Stepper.executeIncomingCursor: ce=%+v\n", ce)
	}
	log.Printf("Stepper.executeIncomingCursor: ce=%+v\n", ce)

	// Every cursor event actually gets added to the step loop,
	// even if we're not recording a loop.  That way, everything gets
	// step-quantized and played back identically, looped or not.

	if r.loopIsRecording {
		ce.LoopsLeft = loopForever
	} else {
		ce.LoopsLeft = 0
	}

	q := r.cursorToQuant(ce)

	quantizedStepnum := r.nextQuant(r.loop.currentStep, q)
	log.Printf("executeCursor: %s, cstep=%d q=%d initial quantizedStepnum=%d\n", ce.Downdragup, r.loop.currentStep, q, quantizedStepnum)
	for quantizedStepnum >= r.loop.length {
		quantizedStepnum -= r.loop.length
		log.Printf("    executeCursor: %s adjusting quantizedStepnum=%d\n", ce.Downdragup, quantizedStepnum)
	}

	// log.Printf("Quantizing currentClick=%d r.loop.currentStep=%d q=%d quantizedStepnum=%d\n",
	// 	currentClick, r.loop.currentStep, q, quantizedStepnum)

	// We create a new "permanent" id for each incoming ce.id,
	// so that all of that id's CursorEvents (from down through UP)
	// in the steps of the loop will have a unique id.

	r.permInstanceIDMutex.RLock()
	permInstanceIDQuantized, ok1 := r.permInstanceIDQuantized[ce.ID]
	permInstanceIDUnquantized, ok2 := r.permInstanceIDUnquantized[ce.ID]
	r.permInstanceIDMutex.RUnlock()

	if (!ok1 || !ok2) && ce.Downdragup != "down" {
		log.Printf("Stepper.executeIncomingCursor: drag or up event didn't find ce.ID=%s in incomingIDToPermSid*\n", ce.ID)
		return
	}

	if ce.Downdragup == "down" {
		// Whether or not this ce.id is in incomingIDToPermSid,
		// we create a new permanent id.  I.e. every
		// gesture added to the loop has a unique permanent id.
		// We also keep track of the quantized down step, so that
		// any drag and UP things (which aren't quantized)
		// aren't added before or on that step.

		r.permInstanceIDMutex.Lock()

		permInstanceIDQuantized = fmt.Sprintf("%s#%d", ce.ID, uniqueIndex)
		uniqueIndex++
		permInstanceIDUnquantized = fmt.Sprintf("%s#%d", ce.ID, uniqueIndex)
		uniqueIndex++

		r.permInstanceIDQuantized[ce.ID] = permInstanceIDQuantized
		r.permInstanceIDUnquantized[ce.ID] = permInstanceIDUnquantized

		r.permInstanceIDDownClick[permInstanceIDQuantized] = r.nextQuant(currentClick, q)
		r.permInstanceIDDownQuant[permInstanceIDQuantized] = q
		r.permInstanceIDDragOK[permInstanceIDQuantized] = false

		r.permInstanceIDMutex.Unlock()
	}

	// We don't want to quantize drag events, but we also don't want them to do anything
	// before the down event (which is quantized), so we only turn on DragOK when we see
	// a drag event come in shortly after the down event.

	if r.permInstanceIDDragOK[permInstanceIDQuantized] == false && ce.Downdragup == "drag" {
		log.Printf("executeIncoming: shouldn't be seeing drag\n")
		if currentClick <= r.permInstanceIDDownClick[permInstanceIDQuantized] {
			return
		}
		r.permInstanceIDDragOK[permInstanceIDQuantized] = true
	}

	ce.ID = permInstanceIDQuantized
	ce.Fresh = true
	ce.Quantized = true

	// Make a separate CursorEvent for the unquantized event
	ceUnquantized := CursorStepEvent{
		ID:         permInstanceIDUnquantized,
		X:          ce.X,
		Y:          ce.Y,
		Z:          ce.Z,
		LoopsLeft:  ce.LoopsLeft,
		Downdragup: ce.Downdragup,
		Quantized:  false,
		Fresh:      true,
	}

	if ce.Downdragup == "up" {
		// The up event always has a Y value of 0 (someday this may, change, but for now...)
		// So, use the quantize value of the down event
		downQuant := r.permInstanceIDDownQuant[permInstanceIDQuantized]
		quantizedStepnum = r.nextQuant(r.loop.currentStep, downQuant)
		log.Printf("executeCursor: up, using downQuant=%d initial quantstenum=%d\n", downQuant, quantizedStepnum)
		for quantizedStepnum >= r.loop.length {
			quantizedStepnum -= r.loop.length
			log.Printf("    executeCursor: adjusting quantstepnum=%d\n", quantizedStepnum)
		}

		// If the up happens too quickly,
		// the graphics don't have time to fire,
		// so push the up event out a few steps.
		magicUpDelayClicks := Clicks(8)
		quantizedStepnum = (quantizedStepnum + magicUpDelayClicks) % r.loop.length
		log.Printf("    executeCursor: FINAL quantstepnum=%d\n", quantizedStepnum)
	}

	// log.Printf("Stepper %s adding to Steps\n", r.padName)
	r.loop.AddToStep(ce, quantizedStepnum)

	// XXX - PUT THIS BACK!!
	_ = ceUnquantized
	// r.loop.AddToStep(ceUnquantized, r.loop.currentStep)

	return
}

func (r *Stepper) nextQuant(t Clicks, q Clicks) Clicks {
	// the algorithm below is the same as KeyKit's nextquant
	if q <= 1 {
		return t
	}
	tq := t
	rem := tq % q
	if (rem * 2) > q {
		tq += (q - rem)
	} else {
		tq -= rem
	}
	if tq < t {
		tq += q
	}
	return tq
}

func (r *Stepper) sendNoteOn(a *ActiveNote) {

	if DebugUtil.MIDI {
		log.Printf("MIDI.SendNote: noteOn pitch:%d velocity:%d sound:%s\n", a.noteOn.Pitch, a.noteOn.Velocity, a.noteOn.Sound)
	}
	SendNoteToSynth(a.noteOn)

	ss := r.params.ParamStringValue("visual.spritesource", "")
	if ss == "midi" {
		r.generateSpriteFromNote(a)
	}
}

func (r *Stepper) sendNoteOff(a *ActiveNote) {
	n := a.noteOn
	if n == nil {
		// Not sure why this sometimes happens
		// log.Printf("HEY! sendNoteOff got a nil?\n")
		return
	}
	// the Note coming in should be a NOTEON
	if n.TypeOf != NOTEON {
		log.Printf("HEY! sendNoteOff expects a NOTEON!?")
	} else {
		noteOff := NewNoteOff(n.Pitch, n.Velocity, n.Sound)
		if DebugUtil.MIDI {
			log.Printf("MIDI.SendNote: noteOff pitch:%d velocity:%d sound:%s\n", n.Pitch, n.Velocity, n.Sound)
		}
		SendNoteToSynth(noteOff)
	}
}

func (r *Stepper) sendANO() {
	if !TheRouter().generateSound {
		return
	}
	synth := r.params.ParamStringValue("sound.synth", defaultSynth)
	SendANOToSynth(synth)
}

/*
func (r *Stepper) paramStringValue(paramname string, def string) string {
	r.paramsMutex.RLock()
	param, ok := r.params[paramname]
	r.paramsMutex.RUnlock()
	if !ok {
		return def
	}
	return r.params.param).(paramValString).value
}

func (r *Stepper) paramIntValue(paramname string) int {
	r.paramsMutex.RLock()
	param, ok := r.params[paramname]
	r.paramsMutex.RUnlock()
	if !ok {
		log.Printf("No param named %s?\n", paramname)
		return 0
	}
	return (param).(paramValInt).value
}
*/

func (r *Stepper) cursorToNoteOn(ce CursorStepEvent) *Note {
	pitch := r.cursorToPitch(ce)
	pitch = uint8(int(pitch) + r.TransposePitch)
	velocity := r.cursorToVelocity(ce)
	synth := r.params.ParamStringValue("sound.synth", defaultSynth)
	// log.Printf("cursorToNoteOn x=%.5f y=%.5f z=%.5f pitch=%d velocity=%d\n", ce.x, ce.y, ce.z, pitch, velocity)
	return NewNoteOn(pitch, velocity, synth)
}

func (r *Stepper) cursorToPitch(ce CursorStepEvent) uint8 {
	pitchmin := r.params.ParamIntValue("sound.pitchmin")
	pitchmax := r.params.ParamIntValue("sound.pitchmax")
	dp := pitchmax - pitchmin + 1
	p1 := int(ce.X * float32(dp))
	p := uint8(pitchmin + p1%dp)
	scale := r.getScale()
	p = scale.ClosestTo(p)
	pnew := p + uint8(12*r.MIDIOctaveShift)
	if pnew < 0 {
		p = pnew + 12
	} else if pnew > 127 {
		p = pnew - 12
	} else {
		p = uint8(pnew)
	}
	return p
}

func (r *Stepper) cursorToVelocity(ce CursorStepEvent) uint8 {
	vol := r.params.ParamStringValue("misc.vol", "fixed")
	velocitymin := r.params.ParamIntValue("sound.velocitymin")
	velocitymax := r.params.ParamIntValue("sound.velocitymax")
	// bogus, when values in json are missing
	if velocitymin == 0 && velocitymax == 0 {
		velocitymin = 0
		velocitymax = 127
	}
	if velocitymin > velocitymax {
		t := velocitymin
		velocitymin = velocitymax
		velocitymax = t
	}
	v := float32(0.8) // default and fixed value
	switch vol {
	case "frets":
		v = 1.0 - ce.Y
	case "pressure":
		v = ce.Z * 4.0
	case "fixed":
		// do nothing
	default:
		log.Printf("Unrecognized vol value: %s, assuming %f\n", vol, v)
	}
	dv := velocitymax - velocitymin + 1
	p1 := int(v * float32(dv))
	vel := uint8(velocitymin + p1%dv)
	return uint8(vel)
}

func (r *Stepper) cursorToDuration(ce CursorStepEvent) int {
	return 92
}

func (r *Stepper) cursorToQuant(ce CursorStepEvent) Clicks {
	quant := r.params.ParamStringValue("misc.quant", "fixed")

	// XXX - UNDO THIS!
	quant = "fixed"

	q := Clicks(1)
	if quant == "none" || quant == "" {
		// q is 1
	} else if quant == "frets" {
		y := 1.0 - ce.Y
		if y > 0.85 {
			q = oneBeat / 8
		} else if y > 0.55 {
			q = oneBeat / 4
		} else if y > 0.25 {
			q = oneBeat / 2
		} else {
			q = oneBeat
		}
	} else if quant == "fixed" {
		q = oneBeat / 4
	} else if quant == "pressure" {
		if ce.Z > 0.30 {
			q = oneBeat / 8
		} else if ce.Z > 0.15 {
			q = oneBeat / 4
		} else if ce.Z > 0.06 {
			q = oneBeat / 2
		} else {
			q = oneBeat
		}
	} else {
		log.Printf("Unrecognized quant: %s\n", quant)
	}
	q = Clicks(float64(q) / TempoFactor)
	// log.Printf("Quant q=%d tempofactor=%f\n", q, TempoFactor)
	return q
}

// Param is a single parameter name/value
type Param struct {
	name  string
	value string
}

func (r *Stepper) handleSetParam(apiprefix, apisuffix string, args map[string]string) (handled bool, err error) {

	// ALL *.set_params and *.set_param APIs
	// set the params in the Stepper.

	handled = false
	if apisuffix == "set_params" {
		for name, value := range args {
			r.params.SetParamValueWithString(apiprefix+name, value, nil)
			if apiprefix == "effect." {
				r.sendEffectParam(name, value)
			}
		}
		handled = true
	}
	if apisuffix == "set_param" {
		name, okname := args["param"]
		value, okvalue := args["value"]
		if !okname || !okvalue {
			err = fmt.Errorf("Stepper.handleSetParam: api=%s%s, missing param or value", apiprefix, apisuffix)
		} else {
			r.params.SetParamValueWithString(apiprefix+name, value, nil)
			if apiprefix == "effect." {
				r.sendEffectParam(name, value)
			}
			handled = true
		}
	}
	return handled, err
}

// ExecuteAPI xxx
func (r *Stepper) ExecuteAPI(api string, args map[string]string, rawargs string) (result string, err error) {

	// log.Printf("ExecuteAPI: api=%s args=%s\n", api, rawargs)

	dot := strings.Index(api, ".")
	var apiprefix string
	apisuffix := api
	if dot >= 0 {
		apiprefix = api[0 : dot+1] // includes the dot
		apisuffix = api[dot+1:]
	}

	handled, err := r.handleSetParam(apiprefix, apisuffix, args)
	if err != nil {
		return "", err
	}

	// ALL visual.* APIs get forwarded to the FreeFrame plugin inside Resolume
	if apiprefix == "visual." {
		msg := osc.NewMessage("/api")
		msg.Append(apisuffix)
		msg.Append(rawargs)
		r.toFreeFramePluginForLayer(msg)
		handled = true
	}

	known := true
	switch api {

	case "loop_recording":
		v, err := needBoolArg("onoff", api, args)
		if err == nil {
			r.loopIsRecording = v
		}

	case "loop_playing":
		v, err := needBoolArg("onoff", api, args)
		if err == nil {
			r.loopIsPlaying = v
			r.terminateActiveNotes()
		}

	case "loop_clear":
		r.loop.Clear()
		r.clearGraphics()
		r.sendANO()

	case "loop_comb":
		r.loopComb()

	case "loop_length":
		i, err := needIntArg("length", api, args)
		if err == nil {
			r.loop.SetLength(Clicks(i))
		}

	case "loop_fade":
		f, err := needFloatArg("fade", api, args)
		if err == nil {
			r.fadeLoop = f
		}

	case "ANO":
		r.sendANO()

	case "midi_thru":
		v, err := needStringArg("thru", api, args)
		if err == nil {
			r.MIDIThru = v
		}

	case "useexternalscale":
		v, err := needBoolArg("onoff", api, args)
		if err == nil {
			r.useExternalScale = v
		}

	case "clearexternalscale":
		// log.Printf("router is clearing external scale\n")
		r.ClearExternalScale()
		r.MIDINumDown = 0

	case "midi_quantized":
		v, err := needBoolArg("quantized", api, args)
		if err == nil {
			r.MIDIQuantized = v
		}

	case "set_transpose":
		v, err := needIntArg("value", api, args)
		if err == nil {
			r.TransposePitch = v
		}

	default:
		known = false
	}

	if !handled && !known {
		err = fmt.Errorf("Stepper.ExecuteAPI: unknown api=%s", api)
	}

	return result, err
}

func (r *Stepper) loopComb() {

	r.loop.stepsMutex.Lock()
	defer r.loop.stepsMutex.Unlock()

	// Create a map of the UP cursor events, so we only do completed notes
	upEvents := make(map[string]CursorStepEvent)
	for _, step := range r.loop.steps {
		if step.events != nil && len(step.events) > 0 {
			for _, event := range step.events {
				if event.cursorStepEvent.Downdragup == "up" {
					upEvents[event.cursorStepEvent.ID] = event.cursorStepEvent
				}
			}
		}
	}
	// log.Printf("loopComb, len(upEvents)=%d\n", len(upEvents))
	combme := 0
	combmod := 2 // should be a parameter
	for id := range upEvents {
		// log.Printf("AT END id = %s\n", id)
		if combme == 0 {
			r.loop.ClearID(id)
			// log.Printf("loopComb, ClearID id=%s upEvents[id]=%+v\n", id, upEvents[id])
			r.generateSoundFromCursor(upEvents[id])
		}
		combme = (combme + 1) % combmod
	}
}

func (r *Stepper) loopQuant() {

	r.loop.stepsMutex.Lock()
	defer r.loop.stepsMutex.Unlock()

	// XXX - Need to make sure we have mutex for changing loop steps
	// XXX - DOES THIS EVEN WORK?

	log.Printf("DOES LOOPQUANT WORK????\n")

	// Create a map of the UP cursor events, so we only do completed notes
	// Create a map of the DOWN events so we know how much to shift that cursor.
	quant := oneBeat / 2
	upEvents := make(map[string]CursorStepEvent)
	downEvents := make(map[string]CursorStepEvent)
	shiftOf := make(map[string]Clicks)
	for stepnum, step := range r.loop.steps {
		if step.events != nil && len(step.events) > 0 {
			for _, e := range step.events {
				switch e.cursorStepEvent.Downdragup {
				case "up":
					upEvents[e.cursorStepEvent.ID] = e.cursorStepEvent
				case "down":
					downEvents[e.cursorStepEvent.ID] = e.cursorStepEvent
					shift := r.nextQuant(Clicks(stepnum), quant)
					log.Printf("Down, shift=%d\n", shift)
					shiftOf[e.cursorStepEvent.ID] = Clicks(shift)
				}
			}
		}
	}

	if len(shiftOf) == 0 {
		return
	}

	// We're going to create a brand new steps array
	newsteps := make([]*Step, len(r.loop.steps))
	for stepnum, step := range r.loop.steps {
		if step.events == nil || len(step.events) == 0 {
			continue
		}
		for _, e := range step.events {
			if !e.hasCursor {
				newsteps[stepnum].events = append(newsteps[stepnum].events, e)
			} else {
				log.Printf("IS THIS CODE EVER EXECUTED?\n")
				id := e.cursorStepEvent.ID
				newstepnum := stepnum
				shift, ok := shiftOf[id]
				// It's shifted
				if ok {
					shiftStep := Clicks(stepnum) + shift
					newstepnum = int(shiftStep) % len(r.loop.steps)
				}
				newsteps[newstepnum].events = append(newsteps[newstepnum].events, e)
			}
		}
	}
}

func (r *Stepper) sendEffectParam(name string, value string) {
	// Effect parameters that have ":" in their name are plugin parameters
	i := strings.Index(name, ":")
	if i > 0 {
		var f float64
		if value == "" {
			f = 0.0
		} else {
			var err error
			f, err = strconv.ParseFloat(value, 64)
			if err != nil {
				log.Printf("Unable to parse float!? value=%s\n", value)
				f = 0.0
			}
		}
		r.sendPadOneEffectParam(name[0:i], name[i+1:], f)
	} else {
		onoff, err := strconv.ParseBool(value)
		if err != nil {
			log.Printf("Unable to parse bool!? value=%s\n", value)
			onoff = false
		}
		r.sendPadOneEffectOnOff(name, onoff)

	}
}

// getEffectMap returns the resolume.json map for a given effect
// and map type ("on", "off", or "params")
func (r *Stepper) getEffectMap(effectName string, mapType string) (map[string]interface{}, string, int, error) {
	if effectName[1] != '-' {
		err := fmt.Errorf("No dash in effect, name=%s", effectName)
		return nil, "", 0, err
	}
	effects, ok := ResolumeJSON["effects"]
	if !ok {
		err := fmt.Errorf("No effects value in resolume.json?")
		return nil, "", 0, err
	}
	realEffectName := effectName[2:]

	n, err := strconv.Atoi(effectName[0:1])
	if err != nil {
		return nil, "", 0, fmt.Errorf("Bad format of effectName=%s", effectName)
	}
	effnum := int(n)

	effectsmap := effects.(map[string]interface{})
	oneEffect, ok := effectsmap[realEffectName]
	if !ok {
		err := fmt.Errorf("No effects value for effect=%s", effectName)
		return nil, "", 0, err
	}
	oneEffectMap := oneEffect.(map[string]interface{})
	mapValue, ok := oneEffectMap[mapType]
	if !ok {
		err := fmt.Errorf("No params value for effect=%s", effectName)
		return nil, "", 0, err
	}
	return mapValue.(map[string]interface{}), realEffectName, effnum, nil
}

func (r *Stepper) addLayerAndClipNums(addr string, layerNum int, clipNum int) string {
	if addr[0] != '/' {
		log.Printf("WARNING, addr in resolume.json doesn't start with / : %s", addr)
		addr = "/" + addr
	}
	addr = fmt.Sprintf("/composition/layers/%d/clips/%d/video/effects%s", layerNum, clipNum, addr)
	return addr
}

func (r *Stepper) resolumeEffectNameOf(name string, num int) string {
	if num == 1 {
		return name
	}
	return fmt.Sprintf("%s%d", name, num)
}

func (r *Stepper) sendPadOneEffectParam(effectName string, paramName string, value float64) {
	paramsMap, realEffectName, realEffectNum, err := r.getEffectMap(effectName, "params")
	if err != nil {
		log.Printf("sendPadOneEffectParam: err=%s\n", err)
		return
	}
	if paramsMap == nil {
		log.Printf("No params value for effect=%s\n", effectName)
		return
	}
	oneParam, ok := paramsMap[paramName]
	if !ok {
		log.Printf("No params value for param=%s in effect=%s\n", paramName, effectName)
		return
	}
	addr := oneParam.(string)

	resEffectName := r.resolumeEffectNameOf(realEffectName, realEffectNum)
	addr = strings.Replace(addr, realEffectName, resEffectName, 1)

	addr = r.addLayerAndClipNums(addr, r.resolumeLayer, 1)

	// log.Printf("sendPadOneEffectParam effect=%s param=%s value=%f\n", effectName, paramName, value)
	msg := osc.NewMessage(addr)
	msg.Append(float32(value))
	r.toResolume(msg)
}

func (r *Stepper) addEffectNum(addr string, effect string, num int) string {
	if num == 1 {
		return addr
	}
	// e.g. "blur" becomes "blur2"
	return strings.Replace(addr, effect, fmt.Sprintf("%s%d", effect, num), 1)
}

func (r *Stepper) sendPadOneEffectOnOff(effectName string, onoff bool) {
	var mapType string
	if onoff {
		mapType = "on"
	} else {
		mapType = "off"
	}

	onoffMap, realEffectName, realEffectNum, err := r.getEffectMap(effectName, mapType)
	if err != nil {
		log.Printf("SendPadOneEffectOnOff: err=%s\n", err)
		return
	}
	if onoffMap == nil {
		log.Printf("No %s value for effect=%s\n", mapType, effectName)
		return
	}

	onoffAddr, ok := onoffMap["addr"]
	if !ok {
		log.Printf("No addr value in onoff for effect=%s\n", effectName)
		return
	}
	onoffArg, ok := onoffMap["arg"]
	if !ok {
		log.Printf("No arg valuei in onoff for effect=%s\n", effectName)
		return
	}
	addr := onoffAddr.(string)
	addr = r.addEffectNum(addr, realEffectName, realEffectNum)
	addr = r.addLayerAndClipNums(addr, r.resolumeLayer, 1)
	onoffValue := int(onoffArg.(float64))

	msg := osc.NewMessage(addr)
	msg.Append(int32(onoffValue))
	r.toResolume(msg)

}