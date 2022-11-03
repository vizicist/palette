package engine

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"sync"

	"github.com/hypebeast/go-osc/osc"
)

/*
var uniqueIndex = 0

// ActiveNote is a currently active MIDI note
type ActiveNote struct {
	id     int
	noteOn *Note
	ce     CursorDeviceEvent // the one that triggered the note
}

*/

// Motor is an entity that that reacts to things (cursor events, apis) and generates output (midi, graphics)
type Motor struct {
	padName         string
	resolumeLayer   int // see ResolumeLayerForPad
	freeframeClient *osc.Client
	resolumeClient  *osc.Client
	guiClient       *osc.Client
	// lastActiveID    int

	// tempoFactor            float64
	// loop            *StepLoop
	// loopIsRecording bool
	// loopIsPlaying   bool
	// fadeLoop        float32
	// lastCursorStepEvent    CursorStepEvent
	// lastUnQuantizedStepNum Clicks

	// params                         map[string]interface{}
	params *ParamValues

	// paramsMutex               sync.RWMutex
	// activeNotes      map[string]*ActiveNote
	// activeNotesMutex sync.RWMutex

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
		params: NewParamValues(),
		// activeNotes: make(map[string]*ActiveNote),
		// fadeLoop:    0.5,

		MIDIOctaveShift:  0,
		MIDIThru:         true,
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

func (motor *Motor) SendNoteToSynth(n *Note) {

	// XXX - eventually this should be done by a responder?
	ss := motor.params.ParamStringValue("visual.spritesource", "")
	if ss == "midi" {
		motor.generateSpriteFromNote(n)
	}
	// if Debug.MIDI {
	// 	log.Printf("FUNC %s: n=%+v\n", CallerFunc(), *n)
	// }

	SendNoteToSynth(n)
}

// PassThruMIDI xxx
func (motor *Motor) PassThruMIDI(e MidiEvent) {

	if Debug.MIDI {
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
	if (status == 0x90 || status == 0x80) && motor.MIDIThruScadjust {
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
		// log.Printf("PassThruMIDI sending note=%v\n", *n)
		motor.SendNoteToSynth(n)
	}
}

/*
// AdvanceByOneClick advances time by 1 click in a StepLoop
func (motor *Motor) AdvanceByOneClick() {

	motor.deviceCursorsMutex.Lock()
	defer motor.deviceCursorsMutex.Unlock()

	motor.activePhrasesManager.AdvanceByOneClick()
}
*/

// HandleMIDITimeReset xxx
func (motor *Motor) HandleMIDITimeReset() {
	log.Printf("HandleMIDITimeReset!! needs implementation\n")
}

// HandleMIDIInput xxx
func (motor *Motor) HandleMidiInput(e MidiEvent) {

	motor.midiInputMutex.Lock()
	defer motor.midiInputMutex.Unlock()

	if Debug.MIDI {
		log.Printf("Router.HandleMIDIInput: event=%+v\n", e)
	}
	if motor.MIDIThru {
		motor.PassThruMIDI(e)
	}
	if motor.MIDISetScale {
		motor.handleMIDISetScaleNote(e)
	}
}

/*
func CallerPackageAndFunc() (string, string) {
	pc, _, _, _ := runtime.Caller(1)

	funcName := runtime.FuncForPC(pc).Name()
	lastDot := strings.LastIndexByte(funcName, '.')

	packagename := funcName[:lastDot]
	funcname := funcName[lastDot+1:]
	return packagename, funcname
}

// CallerFunc gives just the final func name
func CallerFunc() string {
	pc, _, _, _ := runtime.Caller(1)

	funcName := runtime.FuncForPC(pc).Name()
	lastSlash := strings.LastIndexByte(funcName, '/')

	funcname := funcName[lastSlash+1:]
	return funcname
}
*/

func (motor *Motor) SetOneParamValue(fullname, value string) error {

	if Debug.Values {
		log.Printf("SetOneParamValue motor=%s %s %s\n", motor.padName, fullname, value)
	}
	err := motor.params.SetParamValueWithString(fullname, value, nil)
	if err != nil {
		return err
	}

	if strings.HasPrefix(fullname, "visual.") {
		name := strings.TrimPrefix(fullname, "visual.")
		msg := osc.NewMessage("/api")
		msg.Append("set_params")
		args := fmt.Sprintf("{\"%s\":\"%s\"}", name, value)
		msg.Append(args)
		motor.toFreeFramePluginForLayer(msg)
	}

	if strings.HasPrefix(fullname, "effect.") {
		name := strings.TrimPrefix(fullname, "effect.")
		// Effect parameters get sent to Resolume
		motor.sendEffectParam(name, value)
	}

	return nil
}

// ClearExternalScale xxx
func (motor *Motor) clearExternalScale() {
	if Debug.Scale {
		log.Printf("clearExternalScale pad=%s", motor.padName)
	}
	motor.externalScale = MakeScale()
}

// SetExternalScale xxx
func (motor *Motor) setExternalScale(pitch int, on bool) {
	s := motor.externalScale
	for p := pitch; p < 128; p += 12 {
		s.HasNote[p] = on
	}
	if Debug.Scale {
		log.Printf("setExternalScale pad=%s pitch=%v on=%v", motor.padName, pitch, on)
	}
}

/*
func (motor *Motor) handleCursorDeviceEvent(e CursorDeviceEvent) {

	id := e.Source

	motor.deviceCursorsMutex.Lock()
	defer motor.deviceCursorsMutex.Unlock()

	// log.Printf("handleCursorDeviceEvent e=%v\n", e)

	// Special event to clear cursors (by sending them "up" events)
	if e.Ddu == "clear" {
		for id, c := range motor.deviceCursors {
			if !c.downed {
				log.Printf("Hmmm, why is a cursor not downed?\n")
			} else {
				motor.executeIncomingCursor(CursorStepEvent{ID: id, Ddu: "up"})
				if Debug.Cursor {
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
	if Debug.Cursor {
		log.Printf("Motor.handleCursorDeviceEvent: pad=%s id=%s ddu=%s xyz=%.4f,%.4f,%.4f\n", motor.padName, id, e.Ddu, e.X, e.Y, e.Z)
	}

	motor.executeIncomingCursor(cse)

	if e.Ddu == "up" {
		// if Debug.Cursor {
		// 	log.Printf("Router.handleCursorDeviceEvent: deleting cursor id=%s\n", id)
		// }
		delete(motor.deviceCursors, id)
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
*/

/*
func (motor *Motor) clearGraphics() {
	// send an OSC message to Resolume
	motor.toFreeFramePluginForLayer(osc.NewMessage("/clear"))
}
*/

func (motor *Motor) generateSprite(id string, x, y, z float32) {
	if !TheEngine.Router.generateVisuals {
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

/*
func (motor *Motor) generateVisualsFromCursor(ce CursorDeviceEvent) {
	if !TheEngine.Router.generateVisuals {
		return
	}
	// send an OSC message to Resolume
	msg := osc.NewMessage("/cursor")
	msg.Append(ce.Ddu)
	msg.Append(ce.ID)
	msg.Append(float32(ce.X))
	msg.Append(float32(ce.Y))
	msg.Append(float32(ce.Z))
	if Debug.GenVisual {
		log.Printf("Motor.generateVisuals: pad=%s click=%d OSC message = %+v\n", motor.padName, CurrentClick(), msg)
	}
	motor.toFreeFramePluginForLayer(msg)
}
*/

func (motor *Motor) generateSpriteFromNote(n *Note) {

	if n.TypeOf != "noteon" {
		return
	}

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
	case "top":
		y = 1.0
		x = float32(n.Pitch-pitchmin) / float32(pitchmax-pitchmin)
	case "bottom":
		y = 0.0
		x = float32(n.Pitch-pitchmin) / float32(pitchmax-pitchmin)
	case "left":
		y = float32(n.Pitch-pitchmin) / float32(pitchmax-pitchmin)
		x = 0.0
	case "right":
		y = float32(n.Pitch-pitchmin) / float32(pitchmax-pitchmin)
		x = 1.0
	default:
		x = rand.Float32()
		y = rand.Float32()
	}

	// send an OSC message to Resolume
	msg := osc.NewMessage("/sprite")
	msg.Append(x)
	msg.Append(y)
	msg.Append(float32(n.Velocity) / 127.0)

	// Someday localhost should be changed to the actual IP address.
	// XXX - Set sprite ID to pitch, is this right?
	msg.Append(fmt.Sprintf("%d@localhost", n.Pitch))

	// log.Printf("generateSprite msg=%+v\n", msg)
	motor.toFreeFramePluginForLayer(msg)
}

/*
func (motor *Motor) notifyGUI(ce CursorDeviceEvent, wasFresh bool) {
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
	if Debug.Notify {
		log.Printf("Motor.notifyGUI: msg=%v\n", msg)
	}
}
*/

func (motor *Motor) toFreeFramePluginForLayer(msg *osc.Message) {
	motor.freeframeClient.Send(msg)
	if Debug.OSC {
		log.Printf("Motor.toFreeFramePlugin: layer=%d port=%d msg=%v\n", motor.resolumeLayer, motor.freeframeClient.Port(), msg)
	}
}

func (motor *Motor) toResolume(msg *osc.Message) {
	motor.resolumeClient.Send(msg)
	if Debug.OSC || Debug.Resolume {
		log.Printf("Motor.toResolume: msg=%v\n", msg)
	}
}

func (motor *Motor) handleMIDISetScaleNote(e MidiEvent) {
	status := e.Status & 0xf0
	pitch := int(e.Data1)
	if status == 0x90 {
		// If there are no notes held down (i.e. this is the first), clear the scale
		if motor.MIDINumDown < 0 {
			// this can happen when there's a Read error that misses a noteon
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

/*
func (motor *Motor) generateSoundFromCursor(ce CursorDeviceEvent) {
	if !TheEngine.Router.generateSound {
		return
	}
	a := motor.getActiveNote(ce.ID)
		if Debug.Transpose {
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
		// Send noteoff for current note
		if a.noteOn != nil {
			// I think this happens if we get things coming in
			// faster than the checkDelay can generate the UP event.
			log.Printf("Unexpected down when currentNoteOn is non-nil!? currentNoteOn=%+v\n", a)
			// log.Printf("generateMIDI sending NoteOff before down note\n")
			motor.sendNoteOff(a)
		}
		a.noteOn = motor.cursorToNoteOn(ce)
		a.ce = ce
		motor.sendNoteOn(a)
	case "drag":
		if a.noteOn == nil {
			// if we turn on playing in the middle of an existing loop,
			// we may see some drag events without a down.
			// Also, I'm seeing this pretty commonly in other situations,
			// not really sure what the underlying reason is,
			// but it seems to be harmless at the moment.
			if Debug.Router {
				log.Printf("=============== HEY! drag event, a.currentNoteOn == nil?\n")
			}
			return
		}
		newNoteOn := motor.cursorToNoteOn(ce)
		oldpitch := a.noteOn.Pitch
		newpitch := newNoteOn.Pitch
		// We only turn off the existing note (for a given Cursor ID)
		// and start the new one if the pitch changes

		// Also do this if the Z/Velocity value changes more than the trigger value

		// NOTE: this could and perhaps should use a.ce.Z now that we're
		// saving a.ce, like the deltay value

		dz := float64(int(a.noteOn.Velocity) - int(newNoteOn.Velocity))
		deltaz := float32(math.Abs(dz) / 128.0)
		deltaztrig := motor.params.ParamFloatValue("sound._deltaztrig")

		deltay := float32(math.Abs(float64(a.ce.Y - ce.Y)))
		deltaytrig := motor.params.ParamFloatValue("sound._deltaytrig")
		// log.Printf("genSound for drag!   a.noteOn.vel=%d  newNoteOn.vel=%d deltaz=%f deltaztrig=%f\n", a.noteOn.Velocity, newNoteOn.Velocity, deltaz, deltaztrig)

		if motor.params.ParamStringValue("sound.controllerstyle", "nothing") == "modulationonly" {
			zmin := motor.params.ParamFloatValue("sound._controllerzmin")
			zmax := motor.params.ParamFloatValue("sound._controllerzmax")
			cmin := motor.params.ParamIntValue("sound._controllermin")
			cmax := motor.params.ParamIntValue("sound._controllermax")
			oldz := a.ce.Z
			newz := ce.Z
			// XXX - should put the old controller value in ActiveNote so
			// it doesn't need to be computed every time
			oldzc := BoundAndScaleController(oldz, zmin, zmax, cmin, cmax)
			newzc := BoundAndScaleController(newz, zmin, zmax, cmin, cmax)

			if newzc != 0 && newzc != oldzc {
				SendControllerToSynth(a.noteOn.Synth, 1, newzc)
			}
		}

		if newpitch != oldpitch || deltaz > deltaztrig || deltay > deltaytrig {
			motor.sendNoteOff(a)
			a.noteOn = newNoteOn
			a.ce = ce
			if Debug.Transpose {
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
			a.ce = ce // Hmmmm, might be useful, or wrong
			// log.Printf("r=%s UP Setting currentNoteOn to nil!\n", r.padName)
		}
		motor.activeNotesMutex.Lock()
		delete(motor.activeNotes, ce.ID)
		motor.activeNotesMutex.Unlock()
	}
}
*/

/*
// XXXXXXXXXX This is a Responder!
func (router *Router) executeIncomingCursor(ce CursorStepEvent) {
	q := motor.cursorToQuant(ce)
	log.Printf("Should be scheduling a note here!  q=%d\n",q)
}
*/

/*
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

	// if Debug.MIDI {
	// 	log.Printf("MIDI.SendNote: noteOn pitch:%d velocity:%d sound:%s\n", a.noteOn.Pitch, a.noteOn.Velocity, a.noteOn.Sound)
	// }
	motor.SendNoteToSynth(a.noteOn)

	ss := motor.params.ParamStringValue("visual.spritesource", "")
	if ss == "midi" {
		n := a.noteOn
		motor.generateSpriteFromNote(n)
	}
}

// func (motor *Motor) sendController(a *ActiveNote) {
// }

func (motor *Motor) sendNoteOff(a *ActiveNote) {
	n := a.noteOn
	if n == nil {
		// Not sure why this sometimes happens
		// log.Printf("HEY! sendNoteOff got a nil?\n")
		return
	}
	// the Note coming in should be a noteon or noteoff
	if n.TypeOf != "noteon" && n.TypeOf != "noteoff" {
		log.Printf("HEY! sendNoteOff didn't get a noteon or noteoff!?")
		return
	}

	noteOff := NewNoteOff(n.Pitch, n.Velocity, n.Synth)
	// if Debug.MIDI {
	// 	log.Printf("MIDI.SendNote: noteOff pitch:%d velocity:%d sound:%s\n", n.Pitch, n.Velocity, n.Sound)
	// }
	motor.SendNoteToSynth(noteOff)
}
*/

func (motor *Motor) sendANO() {
	if !TheEngine.Router.generateSound {
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

/*
func (motor *Motor) cursorToNoteOn(ce CursorStepEvent) *Note {
	pitch := motor.cursorToPitch(ce)
	// log.Printf("cursorToNoteOn pitch=%v trans=%v", pitch, r.TransposePitch)
	velocity := motor.cursorToVelocity(ce)
	synth := motor.params.ParamStringValue("sound.synth", defaultSynth)
	// log.Printf("cursorToNoteOn x=%.5f y=%.5f z=%.5f pitch=%d velocity=%d\n", ce.x, ce.y, ce.z, pitch, velocity)
	return NewNoteOn(pitch, velocity, synth)
}
*/

/*
func (motor *Motor) cursorToPitch(ce CursorStepEvent) uint8 {
	pitchmin := motor.params.ParamIntValue("sound.pitchmin")
	pitchmax := motor.params.ParamIntValue("sound.pitchmax")
	dp := pitchmax - pitchmin + 1
	p1 := int(ce.X * float32(dp))
	p := uint8(pitchmin + p1%dp)
	// log.Printf("cursorToPitch: X=%f p=%d\n", ce.X, p)
	chromatic := motor.params.ParamBoolValue("sound.chromatic")
	if !chromatic {
		scale := motor.getScale()
		p = scale.ClosestTo(p)
		// MIDIOctaveShift might be negative
		i := int(p) + 12*motor.MIDIOctaveShift
		for i < 0 {
			i += 12
		}
		for i > 127 {
			i -= 12
		}
		p = uint8(i + motor.TransposePitch)
	}
	return p
}
*/

/*
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
*/

/*
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
*/

/*
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
*/

func (motor *Motor) sendEffectParam(name string, value string) {
	// Effect parameters that have ":" in their name are plugin parameters
	i := strings.Index(name, ":")
	if i > 0 {
		effectName := name[0:i]
		paramName := name[i+1:]
		motor.sendPadOneEffectParam(effectName, paramName, value)
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

func (motor *Motor) sendPadOneEffectParam(effectName string, paramName string, value string) {
	fullName := "effect" + "." + effectName + ":" + paramName
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

	oneDef, ok := ParamDefs[fullName]
	if !ok {
		log.Printf("No paramdef value for param=%s in effect=%s\n", paramName, effectName)
		return
	}

	addr := oneParam.(string)
	resEffectName := motor.resolumeEffectNameOf(realEffectName, realEffectNum)
	addr = strings.Replace(addr, realEffectName, resEffectName, 1)
	addr = motor.addLayerAndClipNums(addr, motor.resolumeLayer, 1)

	msg := osc.NewMessage(addr)

	// Append the value to the message, depending on the type of the parameter

	switch oneDef.typedParamDef.(type) {

	case paramDefInt:
		valint, err := strconv.Atoi(value)
		if err != nil {
			log.Printf("paramDefInt conversion err=%s", err)
			valint = 0
		}
		msg.Append(int32(valint))

	case paramDefBool:
		valbool, err := strconv.ParseBool(value)
		if err != nil {
			log.Printf("paramDefBool conversion err=%s", err)
			valbool = false
		}
		onoffValue := 0
		if valbool {
			onoffValue = 1
		}
		msg.Append(int32(onoffValue))

	case paramDefString:
		valstr := value
		msg.Append(valstr)

	case paramDefFloat:
		var valfloat float32
		valfloat, err := ParseFloat32(value, resEffectName)
		if err != nil {
			log.Printf("paramDefFloat conversion err=%s", err)
			valfloat = 0.0
		}
		msg.Append(float32(valfloat))

	default:
		log.Printf("SetParamValueWithString: unknown type of ParamDef for name=%s", fullName)
		return
	}

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

func MotorForRegion(region string) *Motor {
	motor, ok := TheEngine.Router.motors[region]
	if !ok {
		return nil
	} else {
		return motor
	}
}
