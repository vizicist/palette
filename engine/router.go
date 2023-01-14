package engine

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hypebeast/go-osc/osc"
)

var HTTPPort = 3330
var OSCPort = 3333

var LocalAddress = "127.0.0.1"
var GuiPort = 3943

// Router takes events and routes them
type Router struct {
	OSCInput      chan OSCEvent
	midiInputChan chan MidiEvent
	cursorInput   chan CursorEvent

	cursorManager *CursorManager

	killme bool

	AliveWaiters map[string]chan string

	// layerMap map[string]int // map of pads to resolume layer numbers

	midiEventHandler MIDIEventHandler

	// resolumeClient *osc.Client
	guiClient *osc.Client

	myHostname string
	// generateVisuals     bool
	// generateSound       bool
	layerAssignedToNUID map[string]string
	inputEventMutex     sync.RWMutex
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

	r.cursorManager = NewCursorManager()

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

	r.layerAssignedToNUID = make(map[string]string)
	r.guiClient = osc.NewClient(LocalAddress, GuiPort)
	r.OSCInput = make(chan OSCEvent)
	r.midiInputChan = make(chan MidiEvent)
	// r.recordingOn = false

	r.myHostname = ConfigValue("hostname")
	if r.myHostname == "" {
		hostname, err := os.Hostname()
		if err != nil {
			LogError(err)
			hostname = "unknown"
		}
		r.myHostname = hostname
	}

	// By default, the engine handles Cursor events internally.
	// However, if publishcursor is set, it ONLY publishes them,
	// so the expectation is that a plugin will handle it.

	return &r
}

func (r *Router) Start() {

	go r.notifyGUI("restart")
	go r.InputListener()

	err := LoadMorphs()
	if err != nil {
		LogWarn("StartCursorInput: LoadMorphs", "err", err)
	}

	// go StartMorph(r.cursorManager.HandleCursorEvent, 1.0)
	go StartMorph(r.cursorManager.HandleCursorEvent, 1.0)
}

// InputListener listens for local device inputs (OSC, MIDI)
// We could have separate goroutines for the different inputs, but doing
// them in a single select eliminates some need for locking.
func (r *Router) InputListener() {
	for !r.killme {
		select {
		case msg := <-r.OSCInput:
			r.handleOSCInput(msg)
		case event := <-r.midiInputChan:
			ThePluginManager().handleMidiEvent(event)
		case event := <-r.cursorInput:
			r.cursorManager.HandleCursorEvent(event)
		default:
			time.Sleep(time.Millisecond)
		}
	}
	LogInfo("InputListener is being killed")
}

func (r *Router) SetMIDIEventHandler(handler MIDIEventHandler) {
	r.midiEventHandler = handler
}

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
		r.handleClientRestart(e.Msg)

	case "/event": // These messages encode the arguments as JSON
		LogError(fmt.Errorf("/event OSC message should no longer be used"))
		// r.handleOSCEvent(e.Msg)

	case "/button":
		r.handleMMTTButton(e.Msg)

	case "/cursor":
		// This message comes from the mmtt_kinect process,
		// and is the way other cursor servers can convey
		// cursor down/drag/up input.  This is kinda like TUIO,
		// except that it's not a polling interface - servers
		// need to send explicit events, including a reliable "up".
		r.handleMMTTCursor(e.Msg)

	default:
		LogWarn("Router.HandleOSCInput: Unrecognized OSC message", "source", e.Source, "msg", e.Msg)
	}
}

func (r *Router) notifyGUI(eventName string) {
	if !ConfigBool("notifygui") {
		return
	}
	msg := osc.NewMessage("/notify")
	msg.Append(eventName)
	LogError(r.guiClient.Send(msg))
	LogOfType("osc", "Router.notifyGUI", "msg", msg)
}

func (r *Router) handleMMTTButton(msg *osc.Message) {
	buttName, err := argAsString(msg, 0)
	if err != nil {
		LogError(err)
		return
	}
	text := strings.ReplaceAll(buttName, "_", "\n")
	go TheResolume().showText(text)
}

func (r *Router) handleClientRestart(msg *osc.Message) {

	nargs := msg.CountArguments()
	if nargs < 1 {
		LogWarn("Router.handleOSCEvent: too few arguments")
		return
	}
	// Even though the argument is an integer port number,
	// it's a string in the OSC message sent from the Palette FFGL plugin.
	s, err := argAsString(msg, 0)
	if err != nil {
		LogError(err)
		return
	}
	CallApiOnAllPlugins("event", map[string]string{"event": "clientrestart", "portnum": s})
}

// handleMMTTCursor handles messages from MMTT, reformating them as a standard cursor event
func (r *Router) handleMMTTCursor(msg *osc.Message) {

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
	x, err := argAsFloat32(msg, 2)
	if err != nil {
		LogError(err)
		return
	}
	y, err := argAsFloat32(msg, 3)
	if err != nil {
		LogError(err)
		return
	}
	z, err := argAsFloat32(msg, 4)
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
	zfactor := ConfigFloatWithDefault("mmttzfactor", 5.0)
	ahack := ConfigFloatWithDefault("mmttahack", 20.0)
	ce.Z = boundval32(ahack * zfactor * float64(ce.Z))

	xexpand := ConfigFloatWithDefault("mmttxexpand", 1.25)
	ce.X = boundval32(((float64(ce.X) - 0.5) * xexpand) + 0.5)

	yexpand := ConfigFloatWithDefault("mmttyexpand", 1.25)
	ce.Y = boundval32(((float64(ce.Y) - 0.5) * yexpand) + 0.5)

	LogOfType("mmtt", "MMTT Cursor", "source", ce.Source, "ddu", ce.Ddu, "x", ce.X, "y", ce.Y, "z", ce.Z)

	r.cursorManager.HandleCursorEvent(ce)
}

/*
func (r *Router) handleInputEventRaw(rawargs string) {

	args, err := StringMap(rawargs)
	if err != nil {
		return
	}
	pluginName := ExtractAndRemoveValue("plugin", args)
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
