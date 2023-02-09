package engine

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hypebeast/go-osc/osc"
	midi "gitlab.com/gomidi/midi/v2"
)

var HTTPPort = 3330
var OSCPort = 3333
var EventClientPort = 6666
var GuiPort = 3943

var LocalAddress = "127.0.0.1"

var TheRouter *Router

// Router takes events and routes them
type Router struct {
	OSCInput      chan OSCEvent
	midiInputChan chan MidiEvent
	cursorInput   chan CursorEvent

	killme bool

	MIDINumDown     int
	MidiOctaveShift int

	midisetexternalscale bool
	midithru             bool
	MIDIThruScadjust     bool

	guiClient *osc.Client

	inputEventMutex sync.RWMutex
}

type MIDIEventHandler func(MidiEvent)

// OSCEvent is an OSC message
type OSCEvent struct {
	Msg    *osc.Message
	Source string
}

type HeartbeatEvent struct {
	UpTime      time.Duration
	CursorCount int
}

type APIExecutorFunc func(api string, nuid string, rawargs string) (result any, err error)

func NewRouter() *Router {

	r := Router{}

	err := LoadParamEnums()
	if err != nil {
		LogWarn("LoadParamEnums", "err", err)
		// might be fatal, but try to continue
	}
	err = LoadParamDefs()
	if err != nil {
		LogWarn("LoadParamDefs", "err", err)
		// might be fatal, but try to continue
	}

	r.guiClient = osc.NewClient(LocalAddress, GuiPort)
	r.OSCInput = make(chan OSCEvent)
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

	go StartMorph(r.HandleCursorEvent, 1.0)

	go r.notifyGUI("restart")
}

func (r *Router) HandleCursorEvent(ce CursorEvent) {
	TheEngine.sendToOscClients(CursorToOscMsg(ce))
	TheCursorManager.HandleCursorEvent(ce)
}

func (r *Router) HandleMidiEvent(me MidiEvent) {
	TheEngine.sendToOscClients(MidiToOscMsg(me))

	if r.midithru {
		LogInfo("PassThruMIDI", "msg", me.Msg)
		ScheduleAt(me.Msg, CurrentClick())
	}
	if r.midisetexternalscale {
		r.handleMIDISetScaleNote(me)
	}

	err := TheQuadPro.onMidiEvent(me)
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

		if r.MIDINumDown < 0 {
			// may happen when there's a Read error that misses a noteon
			r.MIDINumDown = 0
		}

		// If there are no notes held down (i.e. this is the first), clear the scale
		if r.MIDINumDown == 0 {
			LogOfType("scale", "Clearing external scale")
			ClearExternalScale()
		}
		SetExternalScale(pitch, true)
		r.MIDINumDown++

		// adjust the octave shift based on incoming MIDI
		newshift := 0
		if pitch < 60 {
			newshift = -1
		} else if pitch > 72 {
			newshift = 1
		}
		if newshift != r.MidiOctaveShift {
			r.MidiOctaveShift = newshift
			LogInfo("MidiOctaveShift changed", "octaveshift", r.MidiOctaveShift)
		}
	} else if me.Msg.Is(midi.NoteOffMsg) {
		r.MIDINumDown--
	}
}

// InputListener listens for local device inputs (OSC, MIDI)
// them in a single select eliminates some need for locking.
// We could have separate goroutines for the different inputs, but doing
func (r *Router) InputListener() {
	for !r.killme {
		select {
		case msg := <-r.OSCInput:
			r.handleOSCInput(msg)
		case event := <-r.midiInputChan:
			r.HandleMidiEvent(event)
		case event := <-r.cursorInput:
			r.HandleCursorEvent(event)
		default:
			time.Sleep(time.Millisecond)
		}
	}
	LogInfo("InputListener is being killed")
}

/*
func (r *Router) SetMIDIEventHandler(handler MIDIEventHandler) {
	r.midiEventHandler = handler
}
*/

func ArgToFloat(nm string, args map[string]string) float32 {
	f, err := strconv.ParseFloat(args[nm], 32)
	if err != nil {
		LogError(err)
		f = 0.0
	}
	return float32(f)
}

func ArgsToCursorEvent(args map[string]string) CursorEvent {
	cid := args["cid"]
	// source := args["source"]
	event := strings.TrimPrefix(args["event"], "cursor_")
	x := ArgToFloat("x", args)
	y := ArgToFloat("y", args)
	z := ArgToFloat("z", args)
	ce := CursorEvent{
		Cid: cid,
		// Source: source,
		// Timestamp: time.Now(),
		Ddu:  event,
		X:    x,
		Y:    y,
		Z:    z,
		Area: 0.0,
	}
	return ce
}

/*
// makeMIDIEvent xxx
func (r *Router) makeMIDIEvent(source string, bytes string, args map[string]string) (*MidiEvent, error) {

	var timestamp int64
	s := optionalStringArg("time", args, "")
	if s != "" {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, fmt.Errorf("makeMIDIEvent: unable to parse time value (%s)", s)
		}
		timestamp = int64(f * 1000.0)
	}

	// We expect the string to start with 0x
	if len(bytes) < 2 || bytes[0:2] != "0x" {
		return nil, fmt.Errorf("makeMIDIEvent: invalid bytes value - %s", bytes)
	}
	hexstring := bytes[2:]
	src := []byte(hexstring)
	bytearr := make([]byte, hex.DecodedLen(len(src)))
	nbytes, err := hex.Decode(bytearr, src)
	if err != nil {
		return nil, fmt.Errorf("makeMIDIEvent: unable to decode hex bytes = %s", bytes)
	}
	var event *MidiEvent
	switch nbytes {
	case 0:
		return nil, fmt.Errorf("makeMIDIEvent: unable to handle midi bytes len=%d", nbytes)
	case 1:
		event = &MidiEvent{
			Timestamp: Timestamp(timestamp),
			Status:    int64(bytearr[0]),
			Data1:     int64(0),
			Data2:     int64(0),
			SysEx:     []byte{},
			source:    source,
		}
	case 2:
		// return nil, fmt.Errorf("makeMIDIEvent: unable to handle midi bytes len=%d", nbytes)
		event = &MidiEvent{
			Timestamp: Timestamp(timestamp),
			Status:    int64(bytearr[0]),
			Data1:     int64(bytearr[1]),
			Data2:     int64(0),
			SysEx:     []byte{},
			source:    source,
		}
	case 3:
		event = &MidiEvent{
			Timestamp: Timestamp(timestamp),
			Status:    int64(bytearr[0]),
			Data1:     int64(bytearr[1]),
			Data2:     int64(bytearr[2]),
			SysEx:     []byte{},
			source:    source,
		}
	default:
		event = &MidiEvent{
			Timestamp: Timestamp(timestamp),
			Status:    int64(bytearr[0]),
			Data1:     int64(0),
			Data2:     int64(0),
			SysEx:     bytearr,
		}
	}

	return event, nil
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

// HandleOSCInput xxx
func (r *Router) handleOSCInput(e OSCEvent) {

	LogOfType("osc", "Router.HandleOSCInput", "msg", e.Msg.String())
	switch e.Msg.Address {

	case "/clientrestart":
		// This message currently comes from the FFGL plugins in Resolume
		err := r.oscHandleClientRestart(e.Msg)
		LogIfError(err)

	case "/event": // These messages encode the arguments as JSON
		LogError(fmt.Errorf("/event OSC message should no longer be used"))
		// r.handleOSCEvent(e.Msg)

	case "/button":
		r.oscHandleButton(e.Msg)

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

func (r *Router) notifyGUI(eventName string) {
	if !TheEngine.ParamBool("engine.notifygui") {
		return
	}
	msg := osc.NewMessage("/notify")
	msg.Append(eventName)
	LogError(r.guiClient.Send(msg))
	LogOfType("osc", "Router.notifyGUI", "msg", msg)
}

func (r *Router) oscHandleButton(msg *osc.Message) {
	buttName, err := argAsString(msg, 0)
	if err != nil {
		LogError(err)
		return
	}
	text := strings.ReplaceAll(buttName, "_", "\n")
	go TheResolume().showText(text)
}

func (r *Router) oscHandleClientRestart(msg *osc.Message) error {

	nargs := msg.CountArguments()
	if nargs < 1 {
		return fmt.Errorf("Router.handleOSCEvent: too few arguments")
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
	TheQuadPro.onClientRestart(ffglportnum)
	return nil
}

func (r *Router) getXYZargs(msg *osc.Message) (x, y, z float32, err error) {
	x, err = argAsFloat32(msg, 2)
	if err != nil {
		return
	}
	y, err = argAsFloat32(msg, 3)
	if err != nil {
		return
	}
	z, err = argAsFloat32(msg, 4)
	if err != nil {
		return
	}
	return
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
		LogError(err)
		return
	}
	cid, err := argAsString(msg, 1)
	if err != nil {
		LogError(err)
		return
	}
	x, y, z, err := r.getXYZargs(msg)
	if err != nil {
		LogError(err)
		return
	}
	ce := CursorEvent{
		Cid:   cid + ",mmtt", // NOTE: we add an mmmtt tag
		Click: CurrentClick(),
		Ddu:   ddu,
		X:     x,
		Y:     y,
		Z:     z,
		Area:  0.0,
	}

	// XXX - HACK!!
	zfactor := TheEngine.EngineParamFloatWithDefault("mmttzfactor", 5.0)
	ahack := TheEngine.EngineParamFloatWithDefault("mmttahack", 20.0)
	ce.Z = boundval32(ahack * zfactor * float64(ce.Z))

	xexpand := TheEngine.EngineParamFloatWithDefault("mmttxexpand", 1.25)
	ce.X = boundval32(((float64(ce.X) - 0.5) * xexpand) + 0.5)

	yexpand := TheEngine.EngineParamFloatWithDefault("mmttyexpand", 1.25)
	ce.Y = boundval32(((float64(ce.Y) - 0.5) * yexpand) + 0.5)

	LogOfType("mmtt", "MMTT Cursor", "source", ce.Source, "ddu", ce.Ddu, "x", ce.X, "y", ce.Y, "z", ce.Z)

	TheCursorManager.HandleCursorEvent(ce)
}

/*
// handleSpriteMsg
func (r *Router) handleSpriteMsg(msg *osc.Message) {
	x, y, z, err := r.getXYZargs(msg)
	if err != nil {
		LogError(err)
		return
	}
	LogInfo("Router.handleSpriteMsg", "x", x, "y", y, "z", z)
	// r.cursorManager.HandleCursorEvent(ce)
}
*/

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

/*
func (r *Router) handleInputEventRaw(rawargs string) {

	args, err := StringMap(rawargs)
	if err != nil {
		return
	}
	pluginName := ExtractAndRemoveValueOf("plugin", args)
	r.CursorManager.HandleCursorEvent(args)
}

func (r *Router) handleOSCEvent(msg *osc.Message) {
	tags, _ := msg.TypeTags()
	_ = tags
	nargs := msg.CountArguments()
	if nargs < 1 {
		LogWarn("Router.handleOSCEvent: too few arguments")
		return
	}
	rawargs, err := argAsString(msg, 0)
	if err != nil {
		LogWarn("Router.handleOSCEvent", "err", err)
		return
	}
	r.handleInputEventRaw(rawargs)
}
*/

func argAsInt(msg *osc.Message, index int) (i int, err error) {
	if index >= len(msg.Arguments) {
		return 0, fmt.Errorf("Not enough arguments, looking for index=%d", index)
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
		return f, fmt.Errorf("Not enough arguments, looking for index=%d", index)
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
		return s, fmt.Errorf("Not enough arguments, looking for index=%d", index)
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
