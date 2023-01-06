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

	layerMap map[string]int // map of pads to resolume layer numbers

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

func (r *Router) ResolumeLayerForPad(pad string) int {
	if r.layerMap == nil {
		layerLayers := "1,2,3,4"
		s := ConfigStringWithDefault("layerLayers", layerLayers)
		layers := strings.Split(s, ",")
		if len(layers) != 4 {
			LogWarn("ResolumeLayerForPad: layerLayers value needs 4 values")
			layers = strings.Split(layerLayers, ",")
		}
		r.layerMap = make(map[string]int)
		r.layerMap["A"], _ = strconv.Atoi(layers[0])
		r.layerMap["B"], _ = strconv.Atoi(layers[1])
		r.layerMap["C"], _ = strconv.Atoi(layers[2])
		r.layerMap["D"], _ = strconv.Atoi(layers[3])

	}
	return r.layerMap[pad]
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
		r.handleOSCEvent(e.Msg)

		/*
			case "/cursor":
				r.handleMMTTCursor(e.Msg)
		*/

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
	r.guiClient.Send(msg)
	LogOfType("osc", "Router.notifyGUI", "msg", msg)
}

/*
func (r *Router) handleMMTTButton(butt string) {
	savedName := ConfigStringWithDefault(butt, "")
	if savedName == "" {
		Warn("No Saved assigned to BUTTON, using Perky_Shapes", "butt", butt)
		savedName = "Perky_Shapes"
		return
	}
	saved, err := LoadSaved("saved." + savedName)
	if err != nil {
		LogError(err)
		return
	}
	Info("Router.handleMMTTButton", "butt", butt, "saved", savedName)
	err = saved.applyPresetSavedToLayer("*")
	if err != nil {
		Warn("handleMMTTButton", "saved", savedName, "err", err)
	}

	text := strings.ReplaceAll(savedName, "_", "\n")
	go r.showText(text)
}
*/

func (r *Router) handleClientRestart(msg *osc.Message) {

	tags, _ := msg.TypeTags()
	_ = tags
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
	LogWarn("Router.handleMMTTCursor needs work")
	/*
		tags, _ := msg.TypeTags()
		_ = tags
		nargs := msg.CountArguments()
		if nargs < 1 {
			Warn("Router.handleMMTTCursor: too few arguments")
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
		layerName := "A"
		words := strings.Split(cid, ".")
		if len(words) > 1 {
			layerName = words[0]
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

		layer, err := r.LayerManager.GetLayer(layerName)
		if err != nil {
			// If it's not a layer, it's a button.
			buttonDepth := ConfigFloatWithDefault("mmttbuttondepth", 0.002)
			if z > buttonDepth {
				Warn("NOT triggering button too deep", "z", z, "buttonDepth", buttonDepth)
				return
			}
			if ddu == "down" {
				LogOfType("mmtt", "MMT BUTTON TRIGGERED", "buttonDepth", buttonDepth, "z", z)
				r.handleMMTTButton(layerName)
			}
			return
		}

		ce := CursorEvent{
			ID:        cid,
			Source:    "mmtt",
			Timestamp: time.Now(),
			Ddu:       ddu,
			X:         x,
			Y:         y,
			Z:         z,
			Area:      0.0,
		}

		// XXX - HACK!!
		zfactor := ConfigFloatWithDefault("mmttzfactor", 5.0)
		ahack := ConfigFloatWithDefault("mmttahack", 20.0)
		ce.Z = boundval(ahack * zfactor * ce.Z)

		xexpand := ConfigFloatWithDefault("mmttxexpand", 1.25)
		ce.X = boundval(((ce.X - 0.5) * xexpand) + 0.5)

		yexpand := ConfigFloatWithDefault("mmttyexpand", 1.25)
		ce.Y = boundval(((ce.Y - 0.5) * yexpand) + 0.5)

		LogOfType("mmtt", "MMTT Cursor", "source", ce.Source, "ddu", ce.Ddu, "x", ce.X, "y", ce.Y, "z", ce.Z)

		layer.HandleCursorEvent(ce)
	*/
}

func (r *Router) handleOSCEvent(msg *osc.Message) {
	tags, _ := msg.TypeTags()
	_ = tags
	nargs := msg.CountArguments()
	if nargs < 1 {
		LogWarn("Router.handleOSCEvent: too few arguments")
		return
	}
	LogWarn("Router.handleOSCEvent needs work")
	// rawargs, err := argAsString(msg, 0)
	// if err != nil {
	// 	LogWarn("Router.handleOSCEvent", "err", err)
	// 	return
	// }
	// r.handleInputEventRaw(rawargs)
}

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
