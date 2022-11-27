package engine

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/hypebeast/go-osc/osc"
)

// Player is an entity that that reacts to things (cursor events, apis) and generates output (midi, graphics)
type Player struct {
	playerName      string
	resolumeLayer   int // see ResolumeLayerForPad
	freeframeClient *osc.Client
	resolumeClient  *osc.Client

	params  *ParamValues // across all presets
	sources map[string]string

	responders        []Responder
	respondersContext []*ResponderContext

	// lastActiveID    int

	// tempoFactor            float64
	// loop            *StepLoop
	loopLength      Clicks
	loopIsRecording bool
	loopIsPlaying   bool
	fadeLoop        float32
	// lastCursorStepEvent    CursorStepEvent
	// lastUnQuantizedStepNum Clicks

	// params                         map[string]interface{}

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
	externalScale    *Scale
}

func (p *Player) ApplyPreset(presetName string) error {
	preset, err := LoadPreset(presetName)
	if err != nil {
		return err
	}
	preset.ApplyTo(p.playerName)
	return nil
}

func (p *Player) SavePreset(presetName string) error {
	return p.saveCurrentAsPreset(presetName)
}

func (p *Player) IsSourceAllowed(source string) bool {
	_, ok := p.sources[source]
	return ok
}

func (p *Player) AllowSource(source string) {
	var ok bool
	_, ok = p.sources[source]
	if ok {
		Info("Player.AttachSource already has input", "source", source)
	} else {
		p.sources[source] = source
	}
}

func (p *Player) AttachResponder(responder Responder) {
	p.responders = append(p.responders, responder)
	p.respondersContext = append(p.respondersContext, NewResponderContext(p))
}

func (p *Player) SetResolumeLayer(layernum int, ffglport int) {
	p.resolumeLayer = layernum
	p.resolumeClient = osc.NewClient(LocalAddress, ResolumePort)
	p.freeframeClient = osc.NewClient(LocalAddress, ffglport)
}

// NewPlayer makes a new Player
func NewPlayer(playerName string) *Player {
	p := &Player{
		playerName:      playerName,
		resolumeLayer:   0,
		freeframeClient: nil,
		resolumeClient:  nil,
		// tempoFactor:         1.0,
		// params:                         make(map[string]interface{}),
		params: NewParamValues(),
		// activeNotes: make(map[string]*ActiveNote),
		// fadeLoop:    0.5,
		sources: map[string]string{},

		MIDIOctaveShift:  0,
		MIDIThru:         true,
		MIDISetScale:     false,
		MIDIThruScadjust: false,
		MIDIUseScale:     false,
		MIDIQuantized:    false,
		TransposePitch:   0,
	}
	p.params.SetDefaultValues()
	p.clearExternalScale()
	p.setExternalScale(60%12, true) // Middle C

	TheRouter().PlayerManager.AddPlayer(p)

	return p
}

func (player *Player) SendPhraseElementToSynth(pe *PhraseElement) {

	// XXX - eventually this should be done by a responder?
	ss := player.params.ParamStringValue("visual.spritesource", "")
	if ss == "midi" {
		player.generateSpriteFromPhraseElement(pe)
	}
	SendToSynth(pe)
}

func (player *Player) MidiEventToPhrase(me MidiEvent) (*Phrase, error) {
	pe, err := player.midiEventToPhraseElement(me)
	if err != nil {
		return nil, err
	}
	phr := NewPhrase().InsertElement(pe)
	return phr, nil
}

func (player *Player) midiEventToPhraseElement(me MidiEvent) (*PhraseElement, error) {

	bytes := me.msg.Bytes()
	lng := len(bytes)
	if lng == 0 {
		err := fmt.Errorf("no bytes in midi.Message")
		LogError(err)
		return nil, err
	}
	if lng == 1 {
		return &PhraseElement{Value: NewNoteSystem(bytes[0])}, nil
	}

	synth := player.params.ParamStringValue("sound.synth", defaultSynth)

	var data1, data2 byte
	// NOTE: channel on incoming MIDI is ignored
	// it uses whatever sound the Player is using.
	status := bytes[0] & 0xf0
	if lng == 2 {
		switch status {
		case 0xd0: // channel pressure
			return &PhraseElement{Value: NewChanPressure(bytes[1], synth)}, nil
		}

	}

	switch len(bytes) {
	case 0:
	case 1:
		status = bytes[0] & 0xf0
	case 2:
		status = bytes[0]
		data1 = bytes[1]
	case 3:
		status = bytes[0]
		data1 = bytes[1]
		data2 = bytes[2]
	default:
	}
	pitch := data1

	if (status == 0x90 || status == 0x80) && player.MIDIThruScadjust {
		scale := player.getScale()
		pitch = scale.ClosestTo(pitch)
	}
	var val interface{}
	switch status {
	case 0x90:
		val = NewNoteOn(pitch, data2, synth)
	case 0x80:
		val = NewNoteOff(pitch, data2, synth)
	case 0xB0:
		val = NewController(data1, data2, synth)
	case 0xC0:
		val = NewProgramChange(data1)
	case 0xD0:
		val = NewChanPressure(data1, synth)
	case 0xE0:
		val = NewPitchBend(data1, data2)
	default:
		err := fmt.Errorf("MidiEventToPhraseElement: unable to handle status=0x%02x", status)
		LogError(err, "status", status)
		return nil, err
	}

	return &PhraseElement{Value: val}, nil
}

// HandleMIDITimeReset xxx
func (player *Player) HandleMIDITimeReset() {
	Warn("HandleMIDITimeReset!! needs implementation")
}

// HandleMIDIInput xxx
func (player *Player) HandleCursorEvent(ce CursorEvent) {
	DebugLogOfType("cursor", "Player.HandleCursorEvent", "ce", ce)
	for n, responder := range player.responders {
		ctx := player.respondersContext[n]
		responder.OnCursorEvent(ctx, ce)
	}
}

// HandleMIDIInput xxx
func (player *Player) HandleMidiEvent(me MidiEvent) {
	Info("Player.HandleMidiEvent", "me", me)
	for n, responder := range player.responders {
		ctx := player.respondersContext[n]
		responder.OnMidiEvent(ctx, me)
	}

	/*
		player.midiInputMutex.Lock()
		defer player.midiInputMutex.Unlock()

		DebugLogOfType("midi", "Router.HandleMIDIInput", "event", e)

		if player.MIDIThru {
			player.PassThruMIDI(e)
		}
		if player.MIDISetScale {
			player.handleMIDISetScaleNote(e)
		}
	*/
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

func (player *Player) SetParam(fullname, value string) error {
	return player.SetOneParamValue(fullname, value)
}

func (player *Player) SetOneParamValue(fullname, value string) error {

	DebugLogOfType("value", "SetOneParamValue", "player", player.playerName, "fullname", fullname, "value", value)
	err := player.params.SetParamValueWithString(fullname, value, nil)
	if err != nil {
		return err
	}

	if strings.HasPrefix(fullname, "visual.") {
		name := strings.TrimPrefix(fullname, "visual.")
		msg := osc.NewMessage("/api")
		msg.Append("set_params")
		args := fmt.Sprintf("{\"%s\":\"%s\"}", name, value)
		msg.Append(args)
		player.toFreeFramePluginForLayer(msg)
	}

	if strings.HasPrefix(fullname, "effect.") {
		name := strings.TrimPrefix(fullname, "effect.")
		// Effect parameters get sent to Resolume
		player.sendEffectParam(name, value)
	}

	return nil
}

// ClearExternalScale xxx
func (player *Player) clearExternalScale() {
	DebugLogOfType("scale", "clearExternalScale", "pad", player.playerName)
	player.externalScale = MakeScale()
}

// SetExternalScale xxx
func (player *Player) setExternalScale(pitch int, on bool) {
	s := player.externalScale
	for p := pitch; p < 128; p += 12 {
		s.HasNote[p] = on
	}
}

/*
var uniqueIndex = 0

// ActiveNote is a currently active MIDI note
type ActiveNote struct {
	id     int
	noteOn *Note
	ce     CursorEvent // the one that triggered the note
}

func (player *Player) getActiveNote(id string) *ActiveNote {
	player.activeNotesMutex.RLock()
	a, ok := player.activeNotes[id]
	player.activeNotesMutex.RUnlock()
	if !ok {
		player.lastActiveID++
		a = &ActiveNote{
			id:     player.lastActiveID,
			noteOn: nil,
		}
		player.activeNotesMutex.Lock()
		player.activeNotes[id] = a
		player.activeNotesMutex.Unlock()
	}
	return a
}

func (player) *Player) terminateActiveNotes() {
	player.activeNotesMutex.RLock()
	for id, a := range player.activeNotes {
		if a != nil {
			player.sendNoteOff(a)
		} else {
			Warn("Hey, activeNotes entry for","id",id)
		}
	}
	player.activeNotesMutex.RUnlock()
}
*/

/*
func (player) *Player) clearGraphics() {
	// send an OSC message to Resolume
	player.toFreeFramePluginForLayer(osc.NewMessage("/clear"))
}
*/

func (player *Player) generateSprite(id string, x, y, z float32) {
	if !TheRouter().generateVisuals {
		return
	}
	// send an OSC message to Resolume
	msg := osc.NewMessage("/sprite")
	msg.Append(x)
	msg.Append(y)
	msg.Append(z)
	msg.Append(id)
	player.toFreeFramePluginForLayer(msg)
}

/*
func (player *Player) generateVisualsFromCursor(ce CursorEvent) {
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
	player.toFreeFramePluginForLayer(msg)
}
*/

func (player *Player) generateSpriteFromPhraseElement(pe *PhraseElement) {

	var pitch uint8
	var velocity uint8

	switch v := pe.Value.(type) {
	case *NoteOn:
		pitch = v.Pitch
		velocity = v.Velocity
	case *NoteOff:
		pitch = v.Pitch
		velocity = v.Velocity
	case *NoteFull:
		pitch = v.Pitch
		velocity = v.Velocity
	default:
		return
	}

	pitchmin := uint8(player.params.ParamIntValue("sound.pitchmin"))
	pitchmax := uint8(player.params.ParamIntValue("sound.pitchmax"))
	if pitch < pitchmin || pitch > pitchmax {
		Warn("Unexpected value", "pitch", pitch)
		return
	}

	var x float32
	var y float32
	switch player.params.ParamStringValue("visual.placement", "random") {
	case "random":
		x = rand.Float32()
		y = rand.Float32()
	case "linear":
		y = 0.5
		x = float32(pitch-pitchmin) / float32(pitchmax-pitchmin)
	case "cursor":
		x = rand.Float32()
		y = rand.Float32()
	case "top":
		y = 1.0
		x = float32(pitch-pitchmin) / float32(pitchmax-pitchmin)
	case "bottom":
		y = 0.0
		x = float32(pitch-pitchmin) / float32(pitchmax-pitchmin)
	case "left":
		y = float32(pitch-pitchmin) / float32(pitchmax-pitchmin)
		x = 0.0
	case "right":
		y = float32(pitch-pitchmin) / float32(pitchmax-pitchmin)
		x = 1.0
	default:
		x = rand.Float32()
		y = rand.Float32()
	}

	// send an OSC message to Resolume
	msg := osc.NewMessage("/sprite")
	msg.Append(x)
	msg.Append(y)
	msg.Append(float32(velocity) / 127.0)

	// Someday localhost should be changed to the actual IP address.
	// XXX - Set sprite ID to pitch, is this right?
	msg.Append(fmt.Sprintf("%d@localhost", pitch))

	player.toFreeFramePluginForLayer(msg)
}

func (player *Player) toFreeFramePluginForLayer(msg *osc.Message) {
	player.freeframeClient.Send(msg)
	DebugLogOfType("osc", "Player.toFreeFramePlugin", "layer", player.resolumeLayer, "port", player.freeframeClient.Port(), "osc.message", msg)
}

func (player *Player) toResolume(msg *osc.Message) {
	player.resolumeClient.Send(msg)
	DebugLogOfType("osc", "Player.toResolume", "msg", msg)
}

func (player *Player) handleMIDISetScaleNote(e MidiEvent) {
	Info("handleMIDISetScaleNote needs work")
	/*
		status := e.Status & 0xf0
		pitch := int(e.Data1)
		if status == 0x90 {
			// If there are no notes held down (i.e. this is the first), clear the scale
			if player.MIDINumDown < 0 {
				// this can happen when there's a Read error that misses a noteon
				player.MIDINumDown = 0
			}
			if player.MIDINumDown == 0 {
				player.clearExternalScale()
			}
			player.setExternalScale(pitch%12, true)
			player.MIDINumDown++
			if pitch < 60 {
				player.MIDIOctaveShift = -1
			} else if pitch > 72 {
				player.MIDIOctaveShift = 1
			} else {
				player.MIDIOctaveShift = 0
			}
		} else if status == 0x80 {
			player.MIDINumDown--
		}
	*/
}

// to avoid unused warning
var p *Player
var _ = p.handleMIDISetScaleNote

// getScale xxx
func (player *Player) getScale() *Scale {
	var scaleName string
	var scale *Scale
	if player.MIDIUseScale {
		scale = player.externalScale
	} else {
		scaleName = player.params.ParamStringValue("misc.scale", "newage")
		scale = GlobalScale(scaleName)
	}
	return scale
}

/*
func (player *Player) generateSoundFromCursor(ce CursorEvent) {
	if !TheRouter().generateSound {
		return
	}
	a := player.getActiveNote(ce.ID)
		if Debug.Transpose {
			if a == nil {
				Warn("a is nil")
			} else {
				if a.noteOn == nil {
					Warn("a.noteOn=nil","a.id", a.id)
				} else {
					s := fmt.Sprintf("   a.id=%d a.noteOn=%+v\n", a.id, *(a.noteOn))
					if strings.Contains(s, "PANIC") {
						Warn("HEY, PANIC?")
					} else {
						Warn(s)

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
			player.sendNoteOff(a)
		}
		a.noteOn = player.cursorToNoteOn(ce)
		a.ce = ce
		player.sendNoteOn(a)
	case "drag":
		if a.noteOn == nil {
			// if we turn on playing in the middle of an existing loop,
			// we may see some drag events without a down.
			// Also, I'm seeing this pretty commonly in other situations,
			// not really sure what the underlying reason is,
			// but it seems to be harmless at the moment.
			if Debug.Router {
				Warn("=============== HEY! drag event, a.currentNoteOn == nil?\n")
			}
			return
		}
		newNoteOn := player.cursorToNoteOn(ce)
		oldpitch := a.noteOn.Pitch
		newpitch := newNoteOn.Pitch
		// We only turn off the existing note (for a given Cursor ID)
		// and start the new one if the pitch changes

		// Also do this if the Z/Velocity value changes more than the trigger value

		// NOTE: this could and perhaps should use a.ce.Z now that we're
		// saving a.ce, like the deltay value

		dz := float64(int(a.noteOn.Velocity) - int(newNoteOn.Velocity))
		deltaz := float32(math.Abs(dz) / 128.0)
		deltaztrig := player.params.ParamFloatValue("sound._deltaztrig")

		deltay := float32(math.Abs(float64(a.ce.Y - ce.Y)))
		deltaytrig := player.params.ParamFloatValue("sound._deltaytrig")

		if player.params.ParamStringValue("sound.controllerstyle", "nothing") == "modulationonly" {
			zmin := player.params.ParamFloatValue("sound._controllerzmin")
			zmax := player.params.ParamFloatValue("sound._controllerzmax")
			cmin := player.params.ParamIntValue("sound._controllermin")
			cmax := player.params.ParamIntValue("sound._controllermax")
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
			player.sendNoteOff(a)
			a.noteOn = newNoteOn
			a.ce = ce
			if Debug.Transpose {
				s := fmt.Sprintf("r=%s drag Setting currentNoteOn to %+v\n", player.padName, *(a.noteOn))
				if strings.Contains(s, "PANIC") {
					Warn("PANIC? setting currentNoteOn")
				} else {
					Warn(s)
				}
				Info("generateMIDI sending NoteOn")
			}
			player.sendNoteOn(a)
		}
	case "up":
		if a.noteOn == nil {
			// not sure why this happens, yet
			Warn("Unexpected UP when currentNoteOn is nil?", "player",player.padName)
		} else {
			player.sendNoteOff(a)

			a.noteOn = nil
			a.ce = ce // Hmmmm, might be useful, or wrong
		}
		player.activeNotesMutex.Lock()
		delete(player.activeNotes, ce.ID)
		player.activeNotesMutex.Unlock()
	}
}
*/

/*
func (player *Player) nextQuant(t Clicks, q Clicks) Clicks {
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

func (player *Player) sendNoteOn(a *ActiveNote) {

	player.SendPhraseElementToSynth(a.noteOn)

	ss := player.params.ParamStringValue("visual.spritesource", "")
	if ss == "midi" {
		n := a.noteOn
		player.generateSpriteFromNote(n)
	}
}

// func (player *Player) sendController(a *ActiveNote) {
// }

func (player *Player) sendNoteOff(a *ActiveNote) {
	n := a.noteOn
	if n == nil {
		// Not sure why this sometimes happens
		return
	}
	// the Note coming in should be a noteon or noteoff
	if n.TypeOf != "noteon" && n.TypeOf != "noteoff" {
		Warn("HEY! sendNoteOff didn't get a noteon or noteoff!?")
		return
	}

	noteOff := NewNoteOff(n.Pitch, n.Velocity, n.Synth)
	player.SendPhraseElementToSynth(noteOff)
}
*/

func (player *Player) sendANO() {
	if !TheRouter().generateSound {
		return
	}
	synth := player.params.ParamStringValue("sound.synth", defaultSynth)
	SendANOToSynth(synth)
}

/*
func (r *Player) paramStringValue(paramname string, def string) string {
	r.paramsMutex.RLock()
	param, ok := r.params[paramname]
	r.paramsMutex.RUnlock()
	if !ok {
		return def
	}
	return r.params.param).(paramValString).value
}

func (r *Player) paramIntValue(paramname string) int {
	r.paramsMutex.RLock()
	param, ok := r.params[paramname]
	r.paramsMutex.RUnlock()
	if !ok {
		Warn("No param named","paramname", paramname)
		return 0
	}
	return (param).(paramValInt).value
}
*/

/*
func (player) *Player) cursorToNoteOn(ce CursorStepEvent) *Note {
	pitch := player.cursorToPitch(ce)
	velocity := player.cursorToVelocity(ce)
	synth := player.params.ParamStringValue("sound.synth", defaultSynth)
	return NewNoteOn(pitch, velocity, synth)
}
*/

/*
func (player) *Player) cursorToPitch(ce CursorStepEvent) uint8 {
	pitchmin := player.params.ParamIntValue("sound.pitchmin")
	pitchmax := player.params.ParamIntValue("sound.pitchmax")
	dp := pitchmax - pitchmin + 1
	p1 := int(ce.X * float32(dp))
	p := uint8(pitchmin + p1%dp)
	chromatic := player.params.ParamBoolValue("sound.chromatic")
	if !chromatic {
		scale := player.getScale()
		p = scale.ClosestTo(p)
		// MIDIOctaveShift might be negative
		i := int(p) + 12*player.MIDIOctaveShift
		for i < 0 {
			i += 12
		}
		for i > 127 {
			i -= 12
		}
		p = uint8(i + player.TransposePitch)
	}
	return p
}
*/

/*
func (player) *Player) cursorToVelocity(ce CursorStepEvent) uint8 {
	vol := player.params.ParamStringValue("misc.vol", "fixed")
	velocitymin := player.params.ParamIntValue("sound.velocitymin")
	velocitymax := player.params.ParamIntValue("sound.velocitymax")
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
		Warn("Unrecognized vol value","vol",vol)
	}
	dv := velocitymax - velocitymin + 1
	p1 := int(v * float32(dv))
	vel := uint8(velocitymin + p1%dv)
	return uint8(vel)
}
*/

/*
func (player) *Player) cursorToDuration(ce CursorStepEvent) int {
	return 92
}

func (player) *Player) cursorToQuant(ce CursorStepEvent) Clicks {
	quant := player.params.ParamStringValue("misc.quant", "fixed")

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
		Warn("Unrecognized quant","quant", quant)
	}
	q = Clicks(float64(q) / TempoFactor)
	return q
}
*/

/*
func (player) *Player) loopComb() {

	player.loop.stepsMutex.Lock()
	defer player.loop.stepsMutex.Unlock()

	// Create a map of the UP cursor events, so we only do completed notes
	upEvents := make(map[string]CursorStepEvent)
	for _, step := range player.loop.steps {
		if step.events != nil && len(step.events) > 0 {
			for _, event := range step.events {
				if event.cursorStepEvent.Ddu == "up" {
					upEvents[event.cursorStepEvent.ID] = event.cursorStepEvent
				}
			}
		}
	}
	combme := 0
	combmod := 2 // should be a parameter
	for id := range upEvents {
		if combme == 0 {
			player.loop.ClearID(id)
			player.generateSoundFromCursor(upEvents[id])
		}
		combme = (combme + 1) % combmod
	}
}

func (player) *Player) loopQuant() {

	player.loop.stepsMutex.Lock()
	defer player.loop.stepsMutex.Unlock()

	// XXX - Need to make sure we have mutex for changing loop steps
	// XXX - DOES THIS EVEN WORK?

	Info("DOES LOOPQUANT WORK????")

	// Create a map of the UP cursor events, so we only do completed notes
	// Create a map of the DOWN events so we know how much to shift that cursor.
	quant := oneBeat / 2
	upEvents := make(map[string]CursorStepEvent)
	downEvents := make(map[string]CursorStepEvent)
	shiftOf := make(map[string]Clicks)
	for stepnum, step := range player.loop.steps {
		if step.events != nil && len(step.events) > 0 {
			for _, e := range step.events {
				switch e.cursorStepEvent.Ddu {
				case "up":
					upEvents[e.cursorStepEvent.ID] = e.cursorStepEvent
				case "down":
					downEvents[e.cursorStepEvent.ID] = e.cursorStepEvent
					shift := player.nextQuant(Clicks(stepnum), quant)
					shiftOf[e.cursorStepEvent.ID] = Clicks(shift)
				}
			}
		}
	}

	if len(shiftOf) == 0 {
		return
	}

	// We're going to create a brand new steps array
	newsteps := make([]*Step, len(player.loop.steps))
	for stepnum, step := range player.loop.steps {
		if step.events == nil || len(step.events) == 0 {
			continue
		}
		for _, e := range step.events {
			if !e.hasCursor {
				newsteps[stepnum].events = append(newsteps[stepnum].events, e)
			} else {
				Info("IS THIS CODE EVER EXECUTED?")
				id := e.cursorStepEvent.ID
				newstepnum := stepnum
				shift, ok := shiftOf[id]
				// It's shifted
				if ok {
					shiftStep := Clicks(stepnum) + shift
					newstepnum = int(shiftStep) % len(player.loop.steps)
				}
				newsteps[newstepnum].events = append(newsteps[newstepnum].events, e)
			}
		}
	}
}
*/

func (player *Player) sendEffectParam(name string, value string) {
	// Effect parameters that have ":" in their name are plugin parameters
	i := strings.Index(name, ":")
	if i > 0 {
		effectName := name[0:i]
		paramName := name[i+1:]
		player.sendPadOneEffectParam(effectName, paramName, value)
	} else {
		onoff, err := strconv.ParseBool(value)
		if err != nil {
			LogError(err)
			onoff = false
		}
		player.sendPadOneEffectOnOff(name, onoff)
	}
}

// getEffectMap returns the resolume.json map for a given effect
// and map type ("on", "off", or "params")
func (player *Player) getEffectMap(effectName string, mapType string) (map[string]interface{}, string, int, error) {
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

func (player *Player) addLayerAndClipNums(addr string, layerNum int, clipNum int) string {
	if addr[0] != '/' {
		Warn("addr in resolume.json doesn't start with /", "addr", addr)
		addr = "/" + addr
	}
	addr = fmt.Sprintf("/composition/layers/%d/clips/%d/video/effects%s", layerNum, clipNum, addr)
	return addr
}

func (player *Player) resolumeEffectNameOf(name string, num int) string {
	if num == 1 {
		return name
	}
	return fmt.Sprintf("%s%d", name, num)
}

func (player *Player) sendPadOneEffectParam(effectName string, paramName string, value string) {
	fullName := "effect" + "." + effectName + ":" + paramName
	paramsMap, realEffectName, realEffectNum, err := player.getEffectMap(effectName, "params")
	if err != nil {
		LogError(err)
		return
	}
	if paramsMap == nil {
		Warn("No params value for", "effecdt", effectName)
		return
	}
	oneParam, ok := paramsMap[paramName]
	if !ok {
		Warn("No params value for", "param", paramName, "effect", effectName)
		return
	}

	oneDef, ok := ParamDefs[fullName]
	if !ok {
		Warn("No paramdef value for", "param", paramName, "effect", effectName)
		return
	}

	addr := oneParam.(string)
	resEffectName := player.resolumeEffectNameOf(realEffectName, realEffectNum)
	addr = strings.Replace(addr, realEffectName, resEffectName, 1)
	addr = player.addLayerAndClipNums(addr, player.resolumeLayer, 1)

	msg := osc.NewMessage(addr)

	// Append the value to the message, depending on the type of the parameter

	switch oneDef.typedParamDef.(type) {

	case paramDefInt:
		valint, err := strconv.Atoi(value)
		if err != nil {
			LogError(err)
			valint = 0
		}
		msg.Append(int32(valint))

	case paramDefBool:
		valbool, err := strconv.ParseBool(value)
		if err != nil {
			LogError(err)
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
			LogError(err)
			valfloat = 0.0
		}
		msg.Append(float32(valfloat))

	default:
		Warn("SetParamValueWithString: unknown type of ParamDef for", "name", fullName)
		return
	}

	player.toResolume(msg)
}

func (player *Player) addEffectNum(addr string, effect string, num int) string {
	if num == 1 {
		return addr
	}
	// e.g. "blur" becomes "blur2"
	return strings.Replace(addr, effect, fmt.Sprintf("%s%d", effect, num), 1)
}

func (player *Player) sendPadOneEffectOnOff(effectName string, onoff bool) {
	var mapType string
	if onoff {
		mapType = "on"
	} else {
		mapType = "off"
	}

	onoffMap, realEffectName, realEffectNum, err := player.getEffectMap(effectName, mapType)
	if err != nil {
		LogError(err)
		return
	}

	if onoffMap == nil {
		Warn("No onoffMap value for", "effect", effectName, "maptype", mapType, effectName)
		return
	}

	onoffAddr, ok := onoffMap["addr"]
	if !ok {
		Warn("No addr value in onoff", "effect", effectName)
		return
	}
	onoffArg, ok := onoffMap["arg"]
	if !ok {
		Warn("No arg valuei in onoff for", "effect", effectName)
		return
	}
	addr := onoffAddr.(string)
	addr = player.addEffectNum(addr, realEffectName, realEffectNum)
	addr = player.addLayerAndClipNums(addr, player.resolumeLayer, 1)
	onoffValue := int(onoffArg.(float64))

	msg := osc.NewMessage(addr)
	msg.Append(int32(onoffValue))
	player.toResolume(msg)
}
