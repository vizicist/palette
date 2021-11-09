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
)

var uniqueIndex = 0

// ActiveNote is a currently active MIDI note
type ActiveNote struct {
	id     int
	noteOn *Note
}

// Motor is an entity that that reacts to things (cursor events, apis) and generates output (midi, graphics)
type Motor struct {
	padName         string
	resolumeLayer   int // 1,2,3,4
	freeframeClient *osc.Client
	resolumeClient  *osc.Client
	guiClient       *osc.Client
	lastActiveID    int

	// tempoFactor            float64
	loop            *StepLoop
	loopIsRecording bool
	loopIsPlaying   bool
	fadeLoop        float32
	// lastCursorStepEvent    CursorStepEvent
	// lastUnQuantizedStepNum Clicks

	// params                         map[string]interface{}
	params *ParamValues

	// paramsMutex               sync.RWMutex
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
	MIDIThru         bool
	MIDIThruScadjust bool
	MIDISetScale     bool
	MIDIQuantized    bool
	MIDIUseScale     bool // if true, scadjust uses "external" Scale
	TransposePitch   int
	midiInputMutex   sync.RWMutex
	externalScale    *Scale
}

// NewMotor makes a new Motor
func NewMotor(pad string, resolumeLayer int, freeframeClient *osc.Client, resolumeClient *osc.Client, guiClient *osc.Client) *Motor {
	r := &Motor{
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
		loop:                      NewLoop(oneBeat * 8), // matches default in GUI
		deviceCursors:             make(map[string]*DeviceCursor),
		activePhrasesManager:      NewActivePhrasesManager(),

		MIDIOctaveShift:  0,
		MIDIThru:         false,
		MIDISetScale:     false,
		MIDIThruScadjust: false,
		MIDIUseScale:     false,
		MIDIQuantized:    false,
		TransposePitch:   0,
	}
	r.params.SetDefaultValues()
	r.clearExternalScale()
	r.setExternalScale(60%12, true) // Middle C

	return r
}

// PassThruMIDI xxx
func (motor *Motor) PassThruMIDI(e MidiEvent, scadjust bool) {

	if DebugUtil.MIDI {
		log.Printf("Motor.PassThruMIDI e=%+v\n", e)
	}

	// channel on incoming MIDI is ignored
	// it uses whatever sound the Motor is using
	status := e.Status & 0xf0

	data1 := uint8(e.Data1)
	data2 := uint8(e.Data2)
	pitch := data1

	synth := motor.params.ParamStringValue("sound.synth", defaultSynth)
	var n *Note
	if (status == 0x90 || status == 0x80) && scadjust {
		scale := motor.getScale()
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
		log.Printf("PassThruMIDI unable to handle status=0x%02x\n", status)
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

// AdvanceByOneClick advances time by 1 click in a StepLoop
func (motor *Motor) AdvanceByOneClick() {

	motor.deviceCursorsMutex.Lock()
	defer motor.deviceCursorsMutex.Unlock()

	// motor.activeCursorsMutex.Lock()
	// defer motor.activeCursorsMutex.Unlock()

	// motor.activeNotesMutex.Lock()
	// defer motor.activeNotesMutex.Unlock()

	motor.activePhrasesManager.AdvanceByOneClick()

	loop := motor.loop

	loop.stepsMutex.Lock()
	defer loop.stepsMutex.Unlock()

	stepnum := loop.currentStep

	if DebugUtil.Transpose {
		log.Printf("Advance 1 click stepnum=%d\n", stepnum)
	}

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
				log.Printf("Motor.advanceClickBy1: pad=%s stepnum=%d ce=%+v\n", motor.padName, stepnum, ce)
			}

			ac := motor.getActiveStepCursor(ce)
			ac.x = ce.X
			ac.y = ce.Y
			ac.z = ce.Z

			wasFresh := ce.Fresh

			// Freshly added things ALWAYS get played.
			playit := false
			if ce.Fresh || motor.loopIsPlaying {
				playit = true
			}
			event.cursorStepEvent.Fresh = false
			ce.Fresh = false // needed?

			// Note that we fade the z values in CursorStepEvent, not ActiveStepCursor,
			// because ActiveStepCursor goes away when the gesture ends,
			// while CursorStepEvents in the loop stick around.
			event.cursorStepEvent.Z = event.cursorStepEvent.Z * motor.fadeLoop // fade it
			event.cursorStepEvent.LoopsLeft--
			ce.LoopsLeft--

			minz := float32(0.001)

			// Keep track of the maximum z value for an ActiveCursor,
			// so we know (when get the UP) whether we should
			// delete this gesture.
			switch {
			case ce.Ddu == "down":
				ac.maxzSoFar = -1.0
				ac.lastDrag = -1
			case ce.Ddu == "drag" && ce.Z > ac.maxzSoFar:
				// log.Printf("Saving ce.Z as ac.maxz = %v\n", ce.Z)
				ac.maxzSoFar = ce.Z
			default:
				// log.Printf("up!\n")
			}

			// See if this cursor should be removed
			if ce.Ddu == "up" && (ce.LoopsLeft < 0 || (ce.Z < minz) ||
				(ac.maxzSoFar > 0.0 && ac.maxzSoFar < minz) ||
				(ac.maxzSoFar < 0.0 && ac.downEvent.Z < minz)) {

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
					if ce.Ddu == "drag" {
						currentClick := CurrentClick()
						dclick := currentClick - ac.lastDrag
						if ac.lastDrag < 0 || dclick >= oneBeat/32 {
							ac.lastDrag = currentClick
							if DebugUtil.GenSound {
								log.Printf("generateSoundFromCursor: ddu=%s stepnum=%d ce.ID=%s\n", ce.Ddu, stepnum, ce.ID)
							}
							motor.generateSoundFromCursor(ce)
						}
					} else {
						if DebugUtil.GenSound {
							log.Printf("generateSoundFromCursor: ddu=%s stepnum=%d ce.ID=%s\n", ce.Ddu, stepnum, ce.ID)
						}
						motor.generateSoundFromCursor(ce)
					}
				} else {
					// Graphics and GUI stuff
					ss := motor.params.ParamStringValue("visual.spritesource", "")
					if ss == "cursor" {
						if TheRouter().generateVisuals && DebugUtil.Loop {
							log.Printf("Motor.advanceClickBy1: stepnum=%d generateVisuals ce=%+v\n", stepnum, ce)
						}
						motor.generateVisualsFromCursor(ce)
					}
					motor.notifyGUI(ce, wasFresh)
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
		// if DebugUtil.MIDI {
		// 	log.Printf("Motor.AdvanceClickBy1: region=%s Loop length=%d wrapping around to step 0\n", r.padName, loop.length)
		// }
		loop.currentStep = 0
	}
}

// ExecuteAPI xxx
func (motor *Motor) ExecuteAPI(api string, args map[string]string, rawargs string) (result string, err error) {

	if DebugUtil.MotorAPI {
		log.Printf("MotorAPI: api=%s rawargs=%s\n", api, rawargs)
	}

	dot := strings.Index(api, ".")
	var apiprefix string
	apisuffix := api
	if dot >= 0 {
		apiprefix = api[0 : dot+1] // includes the dot
		apisuffix = api[dot+1:]
	}

	// ALL *.set_params and *.set_param APIs set the params in the Motor.
	//
	// In addition:
	//    Effect parameters get sent one-at-a-time to Resolume's main OSC port (typically 7000).

	handled := false
	if apisuffix == "set_params" {
		for name, value := range args {
			motor.setOneParamValue(apiprefix, name, value)
		}
		handled = true
	}
	if apisuffix == "set_param" {
		name, okname := args["param"]
		value, okvalue := args["value"]
		if !okname || !okvalue {
			return "", fmt.Errorf("Motor.handleSetParam: api=%s%s, missing param or value", apiprefix, apisuffix)
		}
		motor.setOneParamValue(apiprefix, name, value)
		handled = true
	}

	// ALL visual.* APIs get forwarded to the FreeFrame plugin inside Resolume
	if apiprefix == "visual." {
		msg := osc.NewMessage("/api")
		msg.Append(apisuffix)
		msg.Append(rawargs)
		motor.toFreeFramePluginForLayer(msg)
		handled = true
	}

	if handled {
		return result, nil
	}

	known := true
	switch api {

	case "loop_recording":
		v, err := needBoolArg("onoff", api, args)
		if err == nil {
			motor.loopIsRecording = v
		}

	case "loop_playing":
		v, err := needBoolArg("onoff", api, args)
		if err == nil && v != motor.loopIsPlaying {
			motor.loopIsPlaying = v
			motor.terminateActiveNotes()
		}

	case "loop_clear":
		motor.loop.Clear()
		motor.clearGraphics()
		motor.sendANO()

	case "loop_comb":
		motor.loopComb()

	case "loop_length":
		i, err := needIntArg("length", api, args)
		if err == nil {
			nclicks := Clicks(i)
			if nclicks != motor.loop.length {
				motor.loop.SetLength(nclicks)
			}
		}

	case "loop_fade":
		f, err := needFloatArg("fade", api, args)
		if err == nil {
			motor.fadeLoop = f
		}

	case "ANO":
		motor.sendANO()

	case "midi_thru":
		v, err := needBoolArg("onoff", api, args)
		if err == nil {
			motor.MIDIThru = v
		}

	case "midi_setscale":
		v, err := needBoolArg("onoff", api, args)
		if err == nil {
			motor.MIDISetScale = v
		}

	case "midi_usescale":
		v, err := needBoolArg("onoff", api, args)
		if err == nil {
			motor.MIDIUseScale = v
		}

	case "clearexternalscale":
		motor.clearExternalScale()
		motor.MIDINumDown = 0

	case "midi_quantized":
		v, err := needBoolArg("onoff", api, args)
		if err == nil {
			motor.MIDIQuantized = v
		}

	case "midi_thruscadjust":
		v, err := needBoolArg("onoff", api, args)
		if err == nil {
			motor.MIDIThruScadjust = v
		}

	case "set_transpose":
		v, err := needIntArg("value", api, args)
		if err == nil {
			motor.TransposePitch = v
			if DebugUtil.Transpose {
				log.Printf("motor API set_transpose TransposePitch=%v", v)
			}
		}

	default:
		known = false
	}

	if !handled && !known {
		err = fmt.Errorf("Motor.ExecuteAPI: unknown api=%s", api)
	}

	return result, err
}

// HandleMIDITimeReset xxx
func (motor *Motor) HandleMIDITimeReset() {
	log.Printf("HandleMIDITimeReset!! needs implementation\n")
}

// HandleMIDIDeviceInput xxx
func (motor *Motor) HandleMIDIDeviceInput(e MidiEvent) {

	motor.midiInputMutex.Lock()
	defer motor.midiInputMutex.Unlock()

	if DebugUtil.MIDI {
		log.Printf("Router.HandleMIDIDeviceInput: MIDIInput event=%+v\n", e)
	}
	if motor.MIDIThru {
		motor.PassThruMIDI(e, motor.MIDIThruScadjust)
	}
	if motor.MIDISetScale {
		motor.handleMIDISetScaleNote(e)
	}
}

func (motor *Motor) setOneParamValue(apiprefix, name, value string) {
	motor.params.SetParamValueWithString(apiprefix+name, value, nil)
	if apiprefix == "effect." {
		motor.sendEffectParam(name, value)
	}
}

// ClearExternalScale xxx
func (motor *Motor) clearExternalScale() {
	if DebugUtil.Scale {
		log.Printf("clearExternalScale pad=%s", motor.padName)
	}
	motor.externalScale = makeScale()
}

// SetExternalScale xxx
func (motor *Motor) setExternalScale(pitch int, on bool) {
	s := motor.externalScale
	for p := pitch; p < 128; p += 12 {
		s.hasNote[p] = on
	}
	if DebugUtil.Scale {
		log.Printf("setExternalScale pad=%s pitch=%v on=%v", motor.padName, pitch, on)
	}
}

// Time returns the current time
func (motor *Motor) time() time.Time {
	return time.Now()
}

func (motor *Motor) handleCursorDeviceEvent(e CursorDeviceEvent) {

	id := e.NUID + "." + e.CID

	motor.deviceCursorsMutex.Lock()
	defer motor.deviceCursorsMutex.Unlock()

	// Special event to clear cursors (by sending them "up" events)
	if e.Ddu == "clear" {
		for id, c := range motor.deviceCursors {
			if !c.downed {
				log.Printf("Hmmm, why is a cursor not downed?\n")
			} else {
				motor.executeIncomingCursor(CursorStepEvent{ID: id, Ddu: "up"})
				if DebugUtil.Cursor {
					log.Printf("Clearing cursor id=%s\n", id)
				}
				delete(motor.deviceCursors, id)
			}
		}
		return
	}

	// e.Ddu is "down", "drag", or "up"

	tc, ok := motor.deviceCursors[id]
	if !ok {
		// new DeviceCursor
		tc = &DeviceCursor{}
		motor.deviceCursors[id] = tc
	}
	tc.lastTouch = motor.time()

	// If it's a new (downed==false) cursor, make sure the first step event is "down
	if !tc.downed {
		e.Ddu = "down"
		tc.downed = true
	}
	cse := CursorStepEvent{
		ID:  id,
		X:   e.X,
		Y:   e.Y,
		Z:   e.Z,
		Ddu: e.Ddu,
	}
	if DebugUtil.Cursor {
		log.Printf("Motor.handleCursorDeviceEvent: pad=%s id=%s ddu=%s xyz=%.4f,%.4f,%.4f\n", motor.padName, id, e.Ddu, e.X, e.Y, e.Z)
	}

	motor.executeIncomingCursor(cse)

	if e.Ddu == "up" {
		// if DebugUtil.Cursor {
		// 	log.Printf("Router.handleCursorDeviceEvent: deleting cursor id=%s\n", id)
		// }
		delete(motor.deviceCursors, id)
	}
}

// checkDelay is the Duration that has to pass
// before we decide a cursor is no longer present,
// resulting in a cursor UP event.
var checkDelay time.Duration = 0

func (motor *Motor) checkCursorUp() {
	now := motor.time()

	if checkDelay == 0 {
		milli := ConfigIntWithDefault("upcheckmillisecs", 1000)
		checkDelay = time.Duration(milli) * time.Millisecond
	}

	motor.deviceCursorsMutex.Lock()
	defer motor.deviceCursorsMutex.Unlock()

	for id, c := range motor.deviceCursors {
		elapsed := now.Sub(c.lastTouch)
		if elapsed > checkDelay {
			motor.executeIncomingCursor(CursorStepEvent{ID: id, Ddu: "up"})
			if DebugUtil.Cursor {
				log.Printf("Motor.checkCursorUp: deleting cursor id=%s elapsed=%v\n", id, elapsed)
			}
			delete(motor.deviceCursors, id)
		}
	}
}

func (motor *Motor) getActiveNote(id string) *ActiveNote {
	motor.activeNotesMutex.RLock()
	a, ok := motor.activeNotes[id]
	motor.activeNotesMutex.RUnlock()
	if !ok {
		motor.lastActiveID++
		a = &ActiveNote{
			id:     motor.lastActiveID,
			noteOn: nil,
		}
		motor.activeNotesMutex.Lock()
		motor.activeNotes[id] = a
		motor.activeNotesMutex.Unlock()
	}
	return a
}

func (motor *Motor) getActiveStepCursor(ce CursorStepEvent) *ActiveStepCursor {
	sid := ce.ID
	motor.activeCursorsMutex.RLock()
	ac, ok := motor.activeCursors[sid]
	motor.activeCursorsMutex.RUnlock()
	if !ok {
		ac = &ActiveStepCursor{downEvent: ce}
		motor.activeCursorsMutex.Lock()
		motor.activeCursors[sid] = ac
		motor.activeCursorsMutex.Unlock()
	}
	return ac
}

func (motor *Motor) terminateActiveNotes() {
	motor.activeNotesMutex.RLock()
	for id, a := range motor.activeNotes {
		// log.Printf("terminateActiveNotes n=%v\n", a.currentNoteOn)
		if a != nil {
			motor.sendNoteOff(a)
		} else {
			log.Printf("Hey, activeNotes entry for id=%s\n", id)
		}
	}
	motor.activeNotesMutex.RUnlock()
}

func (motor *Motor) clearGraphics() {
	// send an OSC message to Resolume
	motor.toFreeFramePluginForLayer(osc.NewMessage("/clear"))
}

func (motor *Motor) publishSprite(id string, x, y, z float32) {
	err := PublishSpriteEvent(x, y, z)
	if err != nil {
		log.Printf("publishSprite: err=%s\n", err)
	}
}

func (motor *Motor) generateSprite(id string, x, y, z float32) {
	if !TheRouter().generateVisuals {
		return
	}
	// send an OSC message to Resolume
	msg := osc.NewMessage("/sprite")
	msg.Append(x)
	msg.Append(y)
	msg.Append(z)
	msg.Append(id)
	motor.toFreeFramePluginForLayer(msg)
}

func (motor *Motor) generateVisualsFromCursor(ce CursorStepEvent) {
	if !TheRouter().generateVisuals {
		return
	}
	// send an OSC message to Resolume
	msg := osc.NewMessage("/cursor")
	msg.Append(ce.Ddu)
	msg.Append(ce.ID)
	msg.Append(float32(ce.X))
	msg.Append(float32(ce.Y))
	msg.Append(float32(ce.Z))
	if DebugUtil.GenVisual {
		log.Printf("Motor.generateVisuals: pad=%s click=%d stepnum=%d OSC message = %+v\n", motor.padName, CurrentClick(), motor.loop.currentStep, msg)
	}
	motor.toFreeFramePluginForLayer(msg)
}

func (motor *Motor) generateSpriteFromNote(a *ActiveNote) {

	n := a.noteOn
	if n.TypeOf != NOTEON {
		return
	}

	// send an OSC message to Resolume
	oscaddr := "/sprite"

	// The first argument is a relative pitch amoun (0.0 to 1.0) within its range
	pitchmin := uint8(motor.params.ParamIntValue("sound.pitchmin"))
	pitchmax := uint8(motor.params.ParamIntValue("sound.pitchmax"))
	if n.Pitch < pitchmin || n.Pitch > pitchmax {
		log.Printf("Unexpected value of n.Pitch=%d, not between %d and %d\n", n.Pitch, pitchmin, pitchmax)
		return
	}

	var x float32
	var y float32
	switch motor.params.ParamStringValue("visual.placement", "random") {
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
	motor.toFreeFramePluginForLayer(msg)
}

func (motor *Motor) notifyGUI(ce CursorStepEvent, wasFresh bool) {
	if !ConfigBool("notifygui") {
		return
	}
	// send an OSC message to GUI
	msg := osc.NewMessage("/notify")
	msg.Append(ce.Ddu)
	msg.Append(ce.ID)
	msg.Append(float32(ce.X))
	msg.Append(float32(ce.Y))
	msg.Append(float32(ce.Z))
	msg.Append(int32(motor.resolumeLayer))
	msg.Append(wasFresh)
	motor.guiClient.Send(msg)
	if DebugUtil.Notify {
		log.Printf("Motor.notifyGUI: msg=%v\n", msg)
	}
}

func (motor *Motor) toFreeFramePluginForLayer(msg *osc.Message) {
	motor.freeframeClient.Send(msg)
	if DebugUtil.OSC {
		log.Printf("Motor.toFreeFramePlugin: layer=%d msg=%v\n", motor.resolumeLayer, msg)
	}
}

func (motor *Motor) toResolume(msg *osc.Message) {
	motor.resolumeClient.Send(msg)
	if DebugUtil.OSC || DebugUtil.Resolume {
		log.Printf("Motor.toResolume: msg=%v\n", msg)
	}
}

func (motor *Motor) handleMIDISetScaleNote(e MidiEvent) {
	status := e.Status & 0xf0
	pitch := int(e.Data1)
	if status == 0x90 {
		// If there are no notes held down (i.e. this is the first), clear the scale
		if motor.MIDINumDown < 0 {
			// this can happen when there's a Read error that misses a NOTEON
			motor.MIDINumDown = 0
		}
		if motor.MIDINumDown == 0 {
			motor.clearExternalScale()
		}
		motor.setExternalScale(pitch%12, true)
		motor.MIDINumDown++
		if pitch < 60 {
			motor.MIDIOctaveShift = -1
		} else if pitch > 72 {
			motor.MIDIOctaveShift = 1
		} else {
			motor.MIDIOctaveShift = 0
		}
	} else if status == 0x80 {
		motor.MIDINumDown--
	}
}

// getScale xxx
func (motor *Motor) getScale() *Scale {
	var scaleName string
	var scale *Scale
	if motor.MIDIUseScale {
		scale = motor.externalScale
	} else {
		scaleName = motor.params.ParamStringValue("misc.scale", "newage")
		scale = GlobalScale(scaleName)
	}
	return scale
}

func (motor *Motor) generateSoundFromCursor(ce CursorStepEvent) {
	if !TheRouter().generateSound {
		return
	}
	a := motor.getActiveNote(ce.ID)
	if DebugUtil.Transpose {
		log.Printf("Motor.gen: pad=%s ntsactive=%d ce.id=%s ddu=%s\n",
			motor.padName, len(motor.activeNotes), ce.ID, ce.Ddu)
		if a == nil {
			log.Printf("   a is nil\n")
		} else {
			if a.noteOn == nil {
				log.Printf("   a.id=%d a.noteOn=nil\n", a.id)
			} else {
				s := fmt.Sprintf("   a.id=%d a.noteOn=%+v\n", a.id, *(a.noteOn))
				if strings.Contains(s, "PANIC") {
					log.Printf("HEY, PANIC?\n")
				} else {
					log.Printf("%s\n", s)

				}
			}
		}
	}
	switch ce.Ddu {
	case "down":
		// Send NOTEOFF for current note
		if a.noteOn != nil {
			// I think this happens if we get things coming in
			// faster than the checkDelay can generate the UP event.
			log.Printf("Unexpected down when currentNoteOn is non-nil!? currentNoteOn=%+v\n", a)
			// log.Printf("generateMIDI sending NoteOff before down note\n")
			motor.sendNoteOff(a)
		}
		a.noteOn = motor.cursorToNoteOn(ce)
		if DebugUtil.Transpose {
			s := fmt.Sprintf("Setting a.noteOn to %+v!\n", *(a.noteOn))
			if strings.Contains(s, "PANIC") {
				log.Printf("PANIC, a.noteOn?\n")
			} else {
				log.Printf("%s\n", s)
			}
			log.Printf("generateMIDI sending NoteOn for down\n")
		}
		motor.sendNoteOn(a)
	case "drag":
		if a.noteOn == nil {
			// if we turn on playing in the middle of an existing loop,
			// we may see some drag events without a down.
			// Also, I'm seeing this pretty commonly in other situations,
			// not really sure what the underlying reason is,
			// but it seems to be harmless at the moment.
			if DebugUtil.Router {
				log.Printf("=============== HEY! drag event, a.currentNoteOn == nil?\n")
			}
			return
		}
		newNoteOn := motor.cursorToNoteOn(ce)
		oldpitch := a.noteOn.Pitch
		newpitch := newNoteOn.Pitch
		// We only turn off the existing note (for a given Cursor ID)
		// and start the new one if the pitch changes
		if newpitch != oldpitch {
			motor.sendNoteOff(a)
			a.noteOn = newNoteOn
			if DebugUtil.Transpose {
				s := fmt.Sprintf("r=%s drag Setting currentNoteOn to %+v\n", motor.padName, *(a.noteOn))
				if strings.Contains(s, "PANIC") {
					log.Printf("PANIC? setting currentNoteOn\n")
				} else {
					log.Printf("%s\n", s)
				}
				log.Printf("generateMIDI sending NoteOn\n")
			}
			motor.sendNoteOn(a)
		}
	case "up":
		if a.noteOn == nil {
			// not sure why this happens, yet
			log.Printf("r=%s Unexpected UP when currentNoteOn is nil?\n", motor.padName)
		} else {
			// log.Printf("generateMIDI sending NoteOff for UP\n")
			motor.sendNoteOff(a)

			a.noteOn = nil
			// log.Printf("r=%s UP Setting currentNoteOn to nil!\n", r.padName)
		}
		motor.activeNotesMutex.Lock()
		delete(motor.activeNotes, ce.ID)
		motor.activeNotesMutex.Unlock()
	}
}

func (motor *Motor) executeIncomingCursor(ce CursorStepEvent) {

	// Every cursor event actually gets added to the step loop,
	// even if we're not recording a loop.  That way, everything gets
	// step-quantized and played back identically, looped or not.

	if motor.loopIsRecording {
		ce.LoopsLeft = loopForever
	} else {
		ce.LoopsLeft = 0
	}

	if DebugUtil.Transpose {
		log.Printf("MOTOR.INCOMINGCURSOR: ce.id=%s ddu=%s\n", ce.ID, ce.Ddu)
	}

	q := motor.cursorToQuant(ce)

	quantizedStepnum := motor.nextQuant(motor.loop.currentStep, q)
	for quantizedStepnum >= motor.loop.length {
		quantizedStepnum -= motor.loop.length
	}

	// log.Printf("Quantizing currentClick=%d r.loop.currentStep=%d q=%d quantizedStepnum=%d\n",
	// 	currentClick, r.loop.currentStep, q, quantizedStepnum)

	// We create a new "permanent" id for each incoming ce.id,
	// so that all of that id's CursorEvents (from down through UP)
	// in the steps of the loop will have a unique id.

	motor.permInstanceIDMutex.RLock()
	permInstanceIDQuantized, ok1 := motor.permInstanceIDQuantized[ce.ID]
	permInstanceIDUnquantized, ok2 := motor.permInstanceIDUnquantized[ce.ID]
	motor.permInstanceIDMutex.RUnlock()

	if (!ok1 || !ok2) && ce.Ddu != "down" {
		log.Printf("Motor.executeIncomingCursor: drag or up event didn't find ce.ID=%s in incomingIDToPermSid*\n", ce.ID)
		return
	}

	if ce.Ddu == "down" {
		// Whether or not this ce.id is in incomingIDToPermSid,
		// we create a new permanent id.  I.e. every
		// gesture added to the loop has a unique permanent id.
		// We also keep track of the quantized down step, so that
		// any drag and UP things (which aren't quantized)
		// aren't added before or on that step.

		motor.permInstanceIDMutex.Lock()

		permInstanceIDQuantized = fmt.Sprintf("%s#%d", ce.ID, uniqueIndex)
		uniqueIndex++
		permInstanceIDUnquantized = fmt.Sprintf("%s#%d", ce.ID, uniqueIndex)
		uniqueIndex++

		motor.permInstanceIDQuantized[ce.ID] = permInstanceIDQuantized
		motor.permInstanceIDUnquantized[ce.ID] = permInstanceIDUnquantized

		motor.permInstanceIDDownClick[permInstanceIDQuantized] = motor.nextQuant(CurrentClick(), q)
		motor.permInstanceIDDownQuant[permInstanceIDQuantized] = q
		motor.permInstanceIDDragOK[permInstanceIDQuantized] = false

		motor.permInstanceIDMutex.Unlock()
	}

	// We don't want to quantize drag events, but we also don't want them to do anything
	// before the down event (which is quantized), so we only turn on DragOK when we see
	// a drag event come in shortly after the down event.

	if !motor.permInstanceIDDragOK[permInstanceIDQuantized] && ce.Ddu == "drag" {
		if CurrentClick() <= motor.permInstanceIDDownClick[permInstanceIDQuantized] {
			return
		}
		motor.permInstanceIDDragOK[permInstanceIDQuantized] = true
	}

	ce.ID = permInstanceIDQuantized
	ce.Fresh = true
	ce.Quantized = true

	// Make a separate CursorEvent for the unquantized event
	ceUnquantized := CursorStepEvent{
		ID:        permInstanceIDUnquantized,
		X:         ce.X,
		Y:         ce.Y,
		Z:         ce.Z,
		LoopsLeft: ce.LoopsLeft,
		Ddu:       ce.Ddu,
		Quantized: false,
		Fresh:     true,
	}

	if ce.Ddu == "up" {
		// The up event always has a Y value of 0 (someday this may, change, but for now...)
		// So, use the quantize value of the down event
		downQuant := motor.permInstanceIDDownQuant[permInstanceIDQuantized]
		quantizedStepnum = motor.nextQuant(motor.loop.currentStep, downQuant)
		if DebugUtil.Loop {
			log.Printf("UP event: qsn1=%d loopleng=%d\n", quantizedStepnum, motor.loop.length)
		}
		for quantizedStepnum >= motor.loop.length {
			quantizedStepnum -= motor.loop.length
			if DebugUtil.Loop {
				log.Printf("   quantizedStepnum2=%d\n", quantizedStepnum)
			}
		}

		// If the up happens too quickly,
		// the graphics don't have time to fire,
		// so push the up event out a few steps.
		magicUpDelayClicks := Clicks(8)
		quantizedStepnum = (quantizedStepnum + magicUpDelayClicks) % motor.loop.length
		if DebugUtil.Loop {
			log.Printf("   FINAL quantizedStepnum2=%d\n", quantizedStepnum)
		}
	}

	motor.loop.AddToStep(ce, quantizedStepnum)
	motor.loop.AddToStep(ceUnquantized, motor.loop.currentStep)
}

func (motor *Motor) nextQuant(t Clicks, q Clicks) Clicks {
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

func (motor *Motor) sendNoteOn(a *ActiveNote) {

	if DebugUtil.MIDI {
		log.Printf("MIDI.SendNote: noteOn pitch:%d velocity:%d sound:%s\n", a.noteOn.Pitch, a.noteOn.Velocity, a.noteOn.Sound)
	}
	SendNoteToSynth(a.noteOn)

	ss := motor.params.ParamStringValue("visual.spritesource", "")
	if ss == "midi" {
		motor.generateSpriteFromNote(a)
	}
}

func (motor *Motor) sendNoteOff(a *ActiveNote) {
	n := a.noteOn
	if n == nil {
		// Not sure why this sometimes happens
		// log.Printf("HEY! sendNoteOff got a nil?\n")
		return
	}
	// the Note coming in should be a NOTEON or NOTEOFF
	if n.TypeOf != NOTEON && n.TypeOf != NOTEOFF {
		log.Printf("HEY! sendNoteOff didn't get a NOTEON or NOTEOFF!?")
		return
	}

	noteOff := NewNoteOff(n.Pitch, n.Velocity, n.Sound)
	if DebugUtil.MIDI {
		log.Printf("MIDI.SendNote: noteOff pitch:%d velocity:%d sound:%s\n", n.Pitch, n.Velocity, n.Sound)
	}
	SendNoteToSynth(noteOff)
}

func (motor *Motor) sendANO() {
	if !TheRouter().generateSound {
		return
	}
	synth := motor.params.ParamStringValue("sound.synth", defaultSynth)
	SendANOToSynth(synth)
}

/*
func (r *Motor) paramStringValue(paramname string, def string) string {
	r.paramsMutex.RLock()
	param, ok := r.params[paramname]
	r.paramsMutex.RUnlock()
	if !ok {
		return def
	}
	return r.params.param).(paramValString).value
}

func (r *Motor) paramIntValue(paramname string) int {
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

func (motor *Motor) cursorToNoteOn(ce CursorStepEvent) *Note {
	pitch := motor.cursorToPitch(ce)
	pitch = uint8(int(pitch) + motor.TransposePitch)
	// log.Printf("cursorToNoteOn pitch=%v trans=%v", pitch, r.TransposePitch)
	velocity := motor.cursorToVelocity(ce)
	synth := motor.params.ParamStringValue("sound.synth", defaultSynth)
	// log.Printf("cursorToNoteOn x=%.5f y=%.5f z=%.5f pitch=%d velocity=%d\n", ce.x, ce.y, ce.z, pitch, velocity)
	return NewNoteOn(pitch, velocity, synth)
}

func (motor *Motor) cursorToPitch(ce CursorStepEvent) uint8 {
	pitchmin := motor.params.ParamIntValue("sound.pitchmin")
	pitchmax := motor.params.ParamIntValue("sound.pitchmax")
	dp := pitchmax - pitchmin + 1
	p1 := int(ce.X * float32(dp))
	p := uint8(pitchmin + p1%dp)
	scale := motor.getScale()
	p = scale.ClosestTo(p)
	// MIDIOctaveShift might be negative
	pnew := int(p) + 12*motor.MIDIOctaveShift
	for pnew < 0 {
		pnew += 12
	}
	for pnew > 127 {
		pnew -= 12
	}
	return uint8(pnew)
}

func (motor *Motor) cursorToVelocity(ce CursorStepEvent) uint8 {
	vol := motor.params.ParamStringValue("misc.vol", "fixed")
	velocitymin := motor.params.ParamIntValue("sound.velocitymin")
	velocitymax := motor.params.ParamIntValue("sound.velocitymax")
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

func (motor *Motor) cursorToDuration(ce CursorStepEvent) int {
	return 92
}

func (motor *Motor) cursorToQuant(ce CursorStepEvent) Clicks {
	quant := motor.params.ParamStringValue("misc.quant", "fixed")

	q := Clicks(1)
	if quant == "none" || quant == "" {
		// q is 1
	} else if quant == "frets" {
		if ce.Y > 0.85 {
			q = oneBeat / 8
		} else if ce.Y > 0.55 {
			q = oneBeat / 4
		} else if ce.Y > 0.25 {
			q = oneBeat / 2
		} else {
			q = oneBeat
		}
	} else if quant == "fixed" {
		q = oneBeat / 4
	} else if quant == "pressure" {
		if ce.Z > 0.20 {
			q = oneBeat / 8
		} else if ce.Z > 0.10 {
			q = oneBeat / 4
		} else if ce.Z > 0.05 {
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

func (motor *Motor) loopComb() {

	motor.loop.stepsMutex.Lock()
	defer motor.loop.stepsMutex.Unlock()

	// Create a map of the UP cursor events, so we only do completed notes
	upEvents := make(map[string]CursorStepEvent)
	for _, step := range motor.loop.steps {
		if step.events != nil && len(step.events) > 0 {
			for _, event := range step.events {
				if event.cursorStepEvent.Ddu == "up" {
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
			motor.loop.ClearID(id)
			// log.Printf("loopComb, ClearID id=%s upEvents[id]=%+v\n", id, upEvents[id])
			motor.generateSoundFromCursor(upEvents[id])
		}
		combme = (combme + 1) % combmod
	}
}

func (motor *Motor) loopQuant() {

	motor.loop.stepsMutex.Lock()
	defer motor.loop.stepsMutex.Unlock()

	// XXX - Need to make sure we have mutex for changing loop steps
	// XXX - DOES THIS EVEN WORK?

	log.Printf("DOES LOOPQUANT WORK????\n")

	// Create a map of the UP cursor events, so we only do completed notes
	// Create a map of the DOWN events so we know how much to shift that cursor.
	quant := oneBeat / 2
	upEvents := make(map[string]CursorStepEvent)
	downEvents := make(map[string]CursorStepEvent)
	shiftOf := make(map[string]Clicks)
	for stepnum, step := range motor.loop.steps {
		if step.events != nil && len(step.events) > 0 {
			for _, e := range step.events {
				switch e.cursorStepEvent.Ddu {
				case "up":
					upEvents[e.cursorStepEvent.ID] = e.cursorStepEvent
				case "down":
					downEvents[e.cursorStepEvent.ID] = e.cursorStepEvent
					shift := motor.nextQuant(Clicks(stepnum), quant)
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
	newsteps := make([]*Step, len(motor.loop.steps))
	for stepnum, step := range motor.loop.steps {
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
					newstepnum = int(shiftStep) % len(motor.loop.steps)
				}
				newsteps[newstepnum].events = append(newsteps[newstepnum].events, e)
			}
		}
	}
}

func (motor *Motor) sendEffectParam(name string, value string) {
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
		motor.sendPadOneEffectParam(name[0:i], name[i+1:], f)
	} else {
		onoff, err := strconv.ParseBool(value)
		if err != nil {
			log.Printf("Unable to parse bool!? value=%s\n", value)
			onoff = false
		}
		motor.sendPadOneEffectOnOff(name, onoff)

	}
}

// getEffectMap returns the resolume.json map for a given effect
// and map type ("on", "off", or "params")
func (motor *Motor) getEffectMap(effectName string, mapType string) (map[string]interface{}, string, int, error) {
	if effectName[1] != '-' {
		err := fmt.Errorf("no dash in effect, name=%s", effectName)
		return nil, "", 0, err
	}
	effects, ok := ResolumeJSON["effects"]
	if !ok {
		err := fmt.Errorf("no effects value in resolume.json?")
		return nil, "", 0, err
	}
	realEffectName := effectName[2:]

	n, err := strconv.Atoi(effectName[0:1])
	if err != nil {
		return nil, "", 0, fmt.Errorf("bad format of effectName=%s", effectName)
	}
	effnum := int(n)

	effectsmap := effects.(map[string]interface{})
	oneEffect, ok := effectsmap[realEffectName]
	if !ok {
		err := fmt.Errorf("no effects value for effect=%s", effectName)
		return nil, "", 0, err
	}
	oneEffectMap := oneEffect.(map[string]interface{})
	mapValue, ok := oneEffectMap[mapType]
	if !ok {
		err := fmt.Errorf("no params value for effect=%s", effectName)
		return nil, "", 0, err
	}
	return mapValue.(map[string]interface{}), realEffectName, effnum, nil
}

func (motor *Motor) addLayerAndClipNums(addr string, layerNum int, clipNum int) string {
	if addr[0] != '/' {
		log.Printf("WARNING, addr in resolume.json doesn't start with / : %s", addr)
		addr = "/" + addr
	}
	addr = fmt.Sprintf("/composition/layers/%d/clips/%d/video/effects%s", layerNum, clipNum, addr)
	return addr
}

func (motor *Motor) resolumeEffectNameOf(name string, num int) string {
	if num == 1 {
		return name
	}
	return fmt.Sprintf("%s%d", name, num)
}

func (motor *Motor) sendPadOneEffectParam(effectName string, paramName string, value float64) {
	paramsMap, realEffectName, realEffectNum, err := motor.getEffectMap(effectName, "params")
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

	resEffectName := motor.resolumeEffectNameOf(realEffectName, realEffectNum)
	addr = strings.Replace(addr, realEffectName, resEffectName, 1)

	addr = motor.addLayerAndClipNums(addr, motor.resolumeLayer, 1)

	// log.Printf("sendPadOneEffectParam effect=%s param=%s value=%f\n", effectName, paramName, value)
	msg := osc.NewMessage(addr)
	msg.Append(float32(value))
	motor.toResolume(msg)
}

func (motor *Motor) addEffectNum(addr string, effect string, num int) string {
	if num == 1 {
		return addr
	}
	// e.g. "blur" becomes "blur2"
	return strings.Replace(addr, effect, fmt.Sprintf("%s%d", effect, num), 1)
}

func (motor *Motor) sendPadOneEffectOnOff(effectName string, onoff bool) {
	var mapType string
	if onoff {
		mapType = "on"
	} else {
		mapType = "off"
	}

	onoffMap, realEffectName, realEffectNum, err := motor.getEffectMap(effectName, mapType)
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
	addr = motor.addEffectNum(addr, realEffectName, realEffectNum)
	addr = motor.addLayerAndClipNums(addr, motor.resolumeLayer, 1)
	onoffValue := int(onoffArg.(float64))

	msg := osc.NewMessage(addr)
	msg.Append(int32(onoffValue))
	motor.toResolume(msg)

}

// This silliness is to avoid unused function errors from go-staticcheck
var ss *Motor
var _ = ss.publishSprite
var _ = ss.cursorToDuration
var _ = ss.loopQuant
