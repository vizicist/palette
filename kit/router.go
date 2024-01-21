package kit

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/hypebeast/go-osc/osc"
	midi "gitlab.com/gomidi/midi/v2"
)

var TheRouter *Router

// Router takes events and routes them
type Router struct {
	oscInputChan  chan OscEvent
	midiInputChan chan MidiEvent
	cursorInput   chan CursorEvent

	killme bool

	midiNumDown          int
	midiOctaveShift      int
	midisetexternalscale bool
	midithru             bool
	midiThruScadjust     bool

	guiClient *osc.Client

	inputEventMutex sync.RWMutex
}

func NewRouter() *Router {

	r := Router{}

	r.guiClient = osc.NewClient(LocalAddress, GuiPort)
	r.oscInputChan = make(chan OscEvent)
	r.midiInputChan = make(chan MidiEvent)

	// By default, the engine handles Cursor events internally.
	// However, if publishcursor is set, it ONLY publishes them,
	// so the expectation is that a plugin will handle it.

	return &r
}

func (r *Router) Start() {

	go r.InputListener()

	err := LoadMorphs()
	if err != nil {
		LogWarn("StartCursorInput: LoadMorphs", "err", err)
	}

	go StartMorph(r.ScheduleCursorEvent, 1.0)

	go r.notifyGUI("restart")
}

// InputListener listens for local device inputs (OSC, MIDI)
// them in a single select eliminates some need for locking.
func (r *Router) InputListener() {

	for !r.killme {
		r.InputListenOnce()
	}
	LogInfo("InputListener is being killed")
}

func (r *Router) InputListenOnce() {
	defer func() {
		if err := recover(); err != nil {
			LogWarn("Panic in InputListener", "err", err, "stacktrace", string(debug.Stack()))
		}
	}()
	select {
	case msg := <-r.oscInputChan:
		r.handleOscInput(msg)
		TheEngine.RecordOscEvent(&msg)
	case event := <-r.midiInputChan:
		r.HandleMidiEvent(event)
		TheEngine.RecordMidiEvent(&event)
	case event := <-r.cursorInput:
		ScheduleAt(CurrentClick(), event.Tag, event)
	default:
		time.Sleep(time.Millisecond)
	}

}

func (r *Router) ScheduleCursorEvent(ce CursorEvent) {

	// schedule CursorEvents rather than handle them right away.
	// This makes it easier to do looping, among other benefits.

	ScheduleAt(CurrentClick(), ce.Tag, ce)
}

func (r *Router) HandleMidiEvent(me MidiEvent) {
	TheEngine.sendToOscClients(MidiToOscMsg(me))

	if r.midithru {
		LogOfType("midi", "PassThruMIDI", "msg", me.Msg)
		if r.midiThruScadjust {
			LogWarn("PassThruMIDI, midiThruScadjust needs work", "msg", me.Msg)
		}
		ScheduleAt(CurrentClick(), "midi", me.Msg)
	}
	if r.midisetexternalscale {
		r.handleMIDISetScaleNote(me)
	}

	err := TheQuad.onMidiEvent(me)
	LogIfError(err)

	if TheErae.enabled {
		TheErae.onMidiEvent(me)
	}
}

func (r *Router) handleMIDISetScaleNote(me MidiEvent) {
	if !me.HasPitch() {
		return
	}
	pitch := me.Pitch()
	if me.Msg.Is(midi.NoteOnMsg) {

		if r.midiNumDown < 0 {
			// may happen when there's a Read error that misses a noteon
			r.midiNumDown = 0
		}

		// If there are no notes held down (i.e. this is the first), clear the scale
		if r.midiNumDown == 0 {
			LogOfType("scale", "Clearing external scale")
			ClearExternalScale()
		}
		SetExternalScale(pitch, true)
		r.midiNumDown++

		// adjust the octave shift based on incoming MIDI
		newshift := 0
		if pitch < 60 {
			newshift = -1
		} else if pitch > 72 {
			newshift = 1
		}
		if newshift != r.midiOctaveShift {
			r.midiOctaveShift = newshift
			LogOfType("midi", "MidiOctaveShift changed", "octaveshift", r.midiOctaveShift)
		}
	} else if me.Msg.Is(midi.NoteOffMsg) {
		r.midiNumDown--
	}
}

/*
func (r *Router) SetMIDIEventHandler(handler MIDIEventHandler) {
	r.midiEventHandler = handler
}
*/

func ArgToFloat(nm string, args map[string]string) float32 {
	v, err := strconv.ParseFloat(args[nm], 32)
	if err != nil {
		LogIfError(err)
		v = 0.0
	}
	return float32(v)
}

func ArgToInt(nm string, args map[string]string) int {
	v, err := strconv.ParseInt(args[nm], 10, 64)
	if err != nil {
		LogIfError(err)
		v = 0.0
	}
	return int(v)
}

/*
func ArgsToCursorEvent(args map[string]string) CursorEvent {
	gid := ArgToInt("gid", args)
	// source := args["source"]
	ddu := strings.TrimPrefix(args["event"], "cursor_")
	x := ArgToFloat("x", args)
	y := ArgToFloat("y", args)
	z := ArgToFloat("z", args)
	pos := CursorPos{x, y, z}
	ce := NewCursorEvent(gid, ddu, pos)
	return ce
}
*/

func GetArgsXYZ(args map[string]string) (x, y, z float32, err error) {

	api := "GetArgsXYZ"
	x, err = needFloatArg("x", api, args)
	if err != nil {
		return x, y, z, err
	}

	y, err = needFloatArg("y", api, args)
	if err != nil {
		return x, y, z, err
	}

	z, err = needFloatArg("z", api, args)
	if err != nil {
		return x, y, z, err
	}
	return x, y, z, err
}

// HandleOscInput xxx
func (r *Router) handleOscInput(e OscEvent) {

	LogOfType("osc", "Router.HandleOscInput", "msg", e.Msg.String())
	switch e.Msg.Address {

	case "/clientrestart":
		// This message currently comes from the FFGL plugins in Resolume
		err := r.oscHandleClientRestart(e.Msg)
		LogIfError(err)

	case "/event": // These messages encode the arguments as JSON
		LogIfError(fmt.Errorf("/event OSC message should no longer be used"))
		// r.handleOscEvent(e.Msg)

	// case "/button":
	// 	r.oscHandleButton(e.Msg)

	case "/cursor":
		// This message comes from the mmtt_kinect process,
		// and is the way other cursor servers can convey
		// cursor down/drag/up input.  This is kinda like TUIO,
		// except that it's not a polling interface - servers
		// need to send explicit events, including a reliable "up".
		r.oscHandleCursor(e.Msg)

		/*
			case "/sprite":
				r.handleSpriteMsg(e.Msg)
		*/

	case "/api":
		err := r.oscHandleApi(e.Msg)
		LogIfError(err)

	default:
		LogWarn("Router.HandleOSCInput: Unrecognized OSC message", "source", e.Source, "msg", e.Msg)
	}
}

// ArgAsString xxx
func ArgAsString(msg *osc.Message, index int) (s string, err error) {
	arg := msg.Arguments[index]
	switch arg.(type) {
	case string:
		s = arg.(string)
	default:
		err = fmt.Errorf("Expected a string in OSC argument index=%d", index)
	}
	return s, err
}

func (r *Router) notifyGUI(eventName string) {
	b, err := GetParamBool("global.notifygui")
	LogIfError(err)
	if !b {
		return
	}
	msg := osc.NewMessage("/notify")
	msg.Append(eventName)
	TheEngine.SendOsc(r.guiClient, msg)
}

/*
func (r *Router) oscHandleButton(msg *osc.Message) {
	buttName, err := argAsString(msg, 0)
	if err != nil {
		LogIfError(err)
		return
	}
	text := strings.ReplaceAll(buttName, "_", "\n")
	go TheResolume().showText(text)
}
*/

func (r *Router) oscHandleClientRestart(msg *osc.Message) error {

	nargs := msg.CountArguments()
	if nargs < 1 {
		return fmt.Errorf("Router.handleOscEvent: too few arguments")
	}
	// Even though the argument is an integer port number,
	// it's a string in the OSC message sent from the Palette FFGL plugin.
	portnum, err := argAsString(msg, 0)
	if err != nil {
		return err
	}
	ffglportnum, err := strconv.Atoi(portnum)
	if err != nil {
		return err
	}
	TheQuad.onClientRestart(ffglportnum)
	return nil
}

// handleMMTTCursor handles messages from MMTT, reformating them as a standard cursor event
func (r *Router) oscHandleCursor(msg *osc.Message) {

	nargs := msg.CountArguments()
	if nargs < 1 {
		LogWarn("Router.handleMMTTCursor: too few arguments")
		return
	}
	ddu, err := argAsString(msg, 0)
	if err != nil {
		LogIfError(err)
		return
	}
	source, err := argAsString(msg, 1)
	if err != nil {
		LogIfError(err)
		return
	}
	gid, err := argAsInt(msg, 2)
	if err != nil {
		LogIfError(err)
		return
	}
	x, err := argAsFloat32(msg, 3)
	if err != nil {
		LogIfError(err)
		return
	}
	y, err := argAsFloat32(msg, 4)
	if err != nil {
		LogIfError(err)
		return
	}
	z, err := argAsFloat32(msg, 5)
	if err != nil {
		LogIfError(err)
		return
	}

	pos := CursorPos{x, y, z}

	ce := NewCursorEvent(gid, source, ddu, pos)

	// XXX - HACK!!
	xexpand, err := GetParamFloat("global.mmtt_xexpand")
	if err != nil {
		LogIfError(err)
		return
	}
	yexpand, err := GetParamFloat("global.mmtt_yexpand")
	if err != nil {
		LogIfError(err)
		return
	}
	zexpand, err := GetParamFloat("global.mmtt_zexpand")
	if err != nil {
		LogIfError(err)
		return
	}

	// The X, Y, Z values are 0.0 to 1.0.
	// We want to expand the X and Y values a bit around their center
	// (hence the 0.5 stuff), since the values from mmtt_kinect are inset a bit.
	newPos := ce.Pos
	newPos.X = boundValueZeroToOne(((float64(ce.Pos.X) - 0.5) * xexpand) + 0.5)
	newPos.Y = boundValueZeroToOne(((float64(ce.Pos.Y) - 0.5) * yexpand) + 0.5)
	// For Z, we want to expand the value only in the positive direction.
	newPos.Z = boundValueZeroToOne(float64(ce.Pos.Z) * zexpand)

	LogOfType("cursor", "MMTT Cursor", "ddu", ce.Ddu, "newPos", newPos)

	ce.Pos = newPos

	if ce.Ddu != "up" && ce.Pos.Z < 0.0001 {
		LogWarn("Hmmmmmmmm, OSC down/drag cursor event has zero Z?", "ce", ce)
		ce.Pos.Z = TheCursorManager.LoopThreshold
	}
	TheCursorManager.ExecuteCursorEvent(ce)
}

// handleApiMsg
func (r *Router) oscHandleApi(msg *osc.Message) error {
	apijson, err := argAsString(msg, 0)
	if err != nil {
		return err
	}
	var f any
	err = json.Unmarshal([]byte(apijson), &f)
	if err != nil {
		return fmt.Errorf("unable to Unmarshal apijson=%s", apijson)
	}
	LogInfo("Router.handleApiMsg", "apijson", apijson)
	LogInfo("Router.handleApiMsg", "f", f)
	// r.cursorManager.HandleCursorEvent(ce)
	return nil
}

func argAsInt(msg *osc.Message, index int) (i int, err error) {
	if index >= len(msg.Arguments) {
		return 0, fmt.Errorf("not enough arguments, looking for index=%d", index)
	}
	arg := msg.Arguments[index]
	switch v := arg.(type) {
	case int32:
		i = int(v)
	case int64:
		i = int(v)
	default:
		err = fmt.Errorf("expected an int in OSC argument index=%d", index)
	}
	return i, err
}

func argAsFloat32(msg *osc.Message, index int) (f float32, err error) {
	if index >= len(msg.Arguments) {
		return f, fmt.Errorf("not enough arguments, looking for index=%d", index)
	}
	arg := msg.Arguments[index]
	switch v := arg.(type) {
	case float32:
		f = v
	case float64:
		f = float32(v)
	default:
		err = fmt.Errorf("expected a float in OSC argument index=%d", index)
	}
	return f, err
}

func argAsString(msg *osc.Message, index int) (s string, err error) {
	if index >= len(msg.Arguments) {
		return s, fmt.Errorf("not enough arguments, looking for index=%d", index)
	}
	arg := msg.Arguments[index]
	switch v := arg.(type) {
	case string:
		s = v
	default:
		err = fmt.Errorf("expected a string in OSC argument index=%d", index)
	}
	return s, err
}

// This silliness is to avoid unused function errors from go-staticcheck
var _ = argAsInt
var _ = argAsFloat32
