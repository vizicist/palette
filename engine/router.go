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
var AliveOutputPort = 3331

// var FFGLPort = 3334
var LocalAddress = "127.0.0.1"
var ResolumePort = 7000
var GuiPort = 3943
var PloguePort = 3210

// Router takes events and routes them
type Router struct {
	PlayerManager *PlayerManager
	playerLetters string

	OSCInput      chan OSCEvent
	midiInputChan chan MidiEvent
	NoteInput     chan *Note
	CursorInput   chan CursorEvent

	CursorManager *CursorManager
	killme        bool

	AliveWaiters map[string]chan string

	layerMap map[string]int // map of pads to resolume layer numbers

	midiEventHandler MIDIEventHandler
	responderManager *ResponderManager

	resolumeClient *osc.Client
	guiClient      *osc.Client
	plogueClient   *osc.Client

	myHostname           string
	generateVisuals      bool
	generateSound        bool
	playerAssignedToNUID map[string]string
	inputEventMutex      sync.RWMutex
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

// DeviceCursor purpose is to know when it
// hasn't been seen for a while,
// in order to generate an UP event
type DeviceCursor struct {
	lastTouch time.Time
	downed    bool // true if cursor down event has been generated
}

/*
// APIEvent is an API invocation
type APIEvent struct {
	apiType string // sound, visual, effect, global
	pad     string
	method  string
	args    []string
}

// PlaybackEvent is a time-tagged cursor or API event
type PlaybackEvent struct {
	time      float64
	eventType string
	pad       string
	method    string
	args      map[string]string
	rawargs   string
}
func recordingsFile(nm string) string {
	return ConfigFilePath(filepath.Join("recordings", nm))
}

*/

type APIExecutorFunc func(api string, nuid string, rawargs string) (result interface{}, err error)

func NewRouter() *Router {

	r := Router{}

	r.CursorManager = NewCursorManager()
	r.PlayerManager = NewPlayerManager()

	r.playerLetters = ConfigValue("pads")
	if r.playerLetters == "" {
		DebugLogOfType("morph", "No value for pads, assuming ABCD")
		r.playerLetters = "ABCD"
	}

	err := LoadParamEnums()
	if err != nil {
		Warn("LoadParamEnums", "err", err)
		// might be fatal, but try to continue
	}
	err = LoadParamDefs()
	if err != nil {
		Warn("LoadParamDefs", "err", err)
		// might be fatal, but try to continue
	}
	err = LoadResolumeJSON()
	if err != nil {
		Warn("LoadResolumeJSON", "err", err)
		// might be fatal, but try to continue
	}

	r.playerAssignedToNUID = make(map[string]string)

	r.resolumeClient = osc.NewClient(LocalAddress, ResolumePort)
	r.guiClient = osc.NewClient(LocalAddress, GuiPort)
	r.plogueClient = osc.NewClient(LocalAddress, PloguePort)

	/*
		for i, c := range r.playerLetters {
			resolumeLayer := r.ResolumeLayerForPad(string(c))
			ffglPort := ResolumePort + i
			ch := string(c)
			freeframeClient := osc.NewClient("127.0.0.1", ffglPort)
			Info("OSC freeframeClient", "ffglPort", ffglPort, "resolumeLayer", resolumeLayer)
			r.players[ch] = NewPlayer(ch, resolumeLayer, freeframeClient, r.resolumeClient, r.guiClient)
			Info("Pad created", "pad", ch, "resolumeLayer", resolumeLayer, "resolumePort", resolumePort)
		}
	*/

	r.OSCInput = make(chan OSCEvent)
	r.midiInputChan = make(chan MidiEvent)
	r.NoteInput = make(chan *Note)
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
	r.generateVisuals = ConfigBoolWithDefault("generatevisuals", true)
	r.generateSound = ConfigBoolWithDefault("generatesound", true)

	r.responderManager = NewResponderManager()

	return &r
}

func (r *Router) Start() {

	go r.notifyGUI("restart")
	go r.InputListener()

	err := LoadMorphs()
	if err != nil {
		Warn("StartCursorInput: LoadMorphs", "err", err)
	}

	go StartMorph(r.CursorManager.handleCursorEvent, 1.0)
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
			r.PlayerManager.handleMidiEvent(event)
		case event := <-r.CursorInput:
			r.CursorManager.handleCursorEvent(event)
		default:
			time.Sleep(time.Millisecond)
		}
	}
	Info("InputListener is being killed")
}

/*
func (r *Router) handleCursorEvent(ce CursorEvent) {
	Info("Router.handleCursorEvent needs work", "ce", ce)
	r.responderManager.handleCursorEvent(ce)
}
*/

/*
func (r *Router) handleMidiEvent(me MidiEvent) {
	if EraeEnabled {
		HandleEraeMIDI(me)
	}
	r.responderManager.handleMidiEvent(me)
}
*/

func (r *Router) ResolumeLayerForText() int {
	defLayer := "5"
	s := ConfigStringWithDefault("textlayer", defLayer)
	layernum, err := strconv.Atoi(s)
	if err != nil {
		LogError(err)
		layernum, _ = strconv.Atoi(defLayer)
	}
	return layernum
}

func (r *Router) ResolumeLayerForPad(pad string) int {
	if r.layerMap == nil {
		playerLayers := "1,2,3,4"
		s := ConfigStringWithDefault("playerLayers", playerLayers)
		layers := strings.Split(s, ",")
		if len(layers) != 4 {
			Warn("ResolumeLayerForPad: playerLayers value needs 4 values")
			layers = strings.Split(playerLayers, ",")
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
	id := args["id"]
	source := args["source"]
	event := strings.TrimPrefix(args["event"], "cursor_")
	x := ArgToFloat("x", args)
	y := ArgToFloat("y", args)
	z := ArgToFloat("z", args)
	ce := CursorEvent{
		ID:        id,
		Source:    source,
		Timestamp: time.Now(),
		Ddu:       event,
		X:         x,
		Y:         y,
		Z:         z,
		Area:      0.0,
	}
	return ce
}

// HandleInputEvent xxx
func (r *Router) HandleInputEvent(playerName string, args map[string]string) error {

	r.inputEventMutex.Lock()
	defer func() {
		r.inputEventMutex.Unlock()
	}()

	event, err := needStringArg("event", "HandleEvent", args)
	if err != nil {
		return err
	}

	DebugLogOfType("router", "Router.HandleEvent", "player", playerName, "event", event)

	// XXX - player value should allow "*" and other multi-player values
	player, err := r.PlayerManager.GetPlayer(playerName)
	if err != nil {
		return err
	}

	switch event {

	case "engine":
		Info("Router: ignoring engine event")
		return nil

	case "cursor_down", "cursor_drag", "cursor_up":
		ce := ArgsToCursorEvent(args)
		player.HandleCursorEvent(ce)

	case "sprite":

		x, y, z, err := GetArgsXYZ(args)
		if err != nil {
			return nil
		}
		player.generateSprite("dummy", x, y, z)

	case "midi_reset":
		Info("HandleEvent: midi_reset, sending ANO")
		player.HandleMIDITimeReset()
		player.sendANO()

	case "audio_reset":
		Info("HandleEvent: audio_reset!!")
		go r.audioReset()

	case "note":
		notestr, err := needStringArg("note", "HandleEvent", args)
		if err != nil {
			return err
		}
		clickstr, err := needStringArg("clicks", "HandleEvent", args)
		if err != nil {
			return err
		}
		Info("HandleInputEvent", "notestr", notestr, "clickstr", clickstr)
		note, err := NoteFromString(notestr)
		if err != nil {
			return err
		}
		if note.Synth == "" {
			note.Synth = "default"
		}
		SendNoteToSynth(note)

	default:
		Info("HandleInputEvent: event not handled", "event", event)
	}

	return nil
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

	DebugLogOfType("osc", "Router.HandleOSCInput", "msg", e.Msg.String())
	switch e.Msg.Address {

	case "/clientrestart":
		r.handleClientRestart(e.Msg)

	case "/event": // These messages encode the arguments as JSON
		r.handleOSCEvent(e.Msg)

	case "/cursor":
		r.handleMMTTCursor(e.Msg)

	case "/patchxr":
		r.handlePatchXREvent(e.Msg)

	default:
		Warn("Router.HandleOSCInput: Unrecognized OSC message", "source", e.Source, "msg", e.Msg)
	}
}

func (r *Router) notifyGUI(eventName string) {
	if !ConfigBool("notifygui") {
		return
	}
	msg := osc.NewMessage("/notify")
	msg.Append(eventName)
	r.guiClient.Send(msg)
	DebugLogOfType("osc", "Router.notifyGUI", "msg", msg)
}

func (r *Router) handleMMTTButton(butt string) {
	presetName := ConfigStringWithDefault(butt, "")
	if presetName == "" {
		Warn("No Preset assigned to BUTTON, using Perky_Shapes", "butt", butt)
		presetName = "Perky_Shapes"
		return
	}
	preset := GetPreset("quad." + presetName)
	Info("Router.handleMMTTButton", "butt", butt, "preset", presetName)
	err := preset.loadQuadPreset("*")
	if err != nil {
		Warn("handleMMTTButton", "preset", presetName, "err", err)
	}

	text := strings.ReplaceAll(presetName, "_", "\n")
	go r.showText(text)
}

func (r *Router) showText(text string) {

	textLayerNum := r.ResolumeLayerForText()

	// disable the text display by bypassing the layer
	bypassLayer(r.resolumeClient, textLayerNum, true)

	addr := fmt.Sprintf("/composition/layers/%d/clips/1/video/source/textgenerator/text/params/lines", textLayerNum)
	msg := osc.NewMessage(addr)
	msg.Append(text)
	_ = r.resolumeClient.Send(msg)

	// give it time to "sink in", otherwise the previous text displays briefly
	time.Sleep(150 * time.Millisecond)

	connectClip(r.resolumeClient, textLayerNum, 1)
	bypassLayer(r.resolumeClient, textLayerNum, false)
}

func (r *Router) handleClientRestart(msg *osc.Message) {

	tags, _ := msg.TypeTags()
	_ = tags
	nargs := msg.CountArguments()
	if nargs < 1 {
		Warn("Router.handleOSCEvent: too few arguments")
		return
	}
	// Even though the argument is an integer port number,
	// it's a string in the OSC message sent from the Palette FFGL plugin.
	s, err := argAsString(msg, 0)
	if err != nil {
		LogError(err)
		return
	}
	portnum, err := strconv.Atoi(s)
	if err != nil {
		LogError(err)
		return
	}
	var found *Player
	r.PlayerManager.ApplyToAllPlayers(func(player *Player) {
		if player.freeframeClient.Port() == portnum {
			found = player
		}
	})
	if found == nil {
		Warn("handleClientRestart unable to find Player with", "portnum", portnum)
	} else {
		found.sendAllParameters()
	}
}

// handleMMTTCursor handles messages from MMTT, reformating them as a standard cursor event
func (r *Router) handleMMTTCursor(msg *osc.Message) {

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
	playerName := "A"
	words := strings.Split(cid, ".")
	if len(words) > 1 {
		playerName = words[0]
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

	player, err := r.PlayerManager.GetPlayer(playerName)
	if err != nil {
		// If it's not a player, it's a button.
		buttonDepth := ConfigFloatWithDefault("mmttbuttondepth", 0.002)
		if z > buttonDepth {
			Warn("NOT triggering button too deep", "z", z, "buttonDepth", buttonDepth)
			return
		}
		if ddu == "down" {
			DebugLogOfType("mmtt", "MMT BUTTON TRIGGERED", "buttonDepth", buttonDepth, "z", z)
			r.handleMMTTButton(playerName)
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

	DebugLogOfType("mmtt", "MMTT Cursor", "source", ce.Source, "ddu", ce.Ddu, "x", ce.X, "y", ce.Y, "z", ce.Z)

	player.HandleCursorEvent(ce)
}

func (r *Router) handleOSCEvent(msg *osc.Message) {
	tags, _ := msg.TypeTags()
	_ = tags
	nargs := msg.CountArguments()
	if nargs < 1 {
		Warn("Router.handleOSCEvent: too few arguments")
		return
	}
	rawargs, err := argAsString(msg, 0)
	if err != nil {
		Warn("Router.handleOSCEvent", "err", err)
		return
	}
	r.handleInputEventRaw(rawargs)
}

func (r *Router) handleInputEventRaw(rawargs string) {

	args, err := StringMap(rawargs)
	if err != nil {
		return
	}
	playerName := extractPlayer(args)
	r.HandleInputEvent(playerName,args)
}

func (r *Router) handlePatchXREvent(msg *osc.Message) {

	tags, _ := msg.TypeTags()
	_ = tags
	nargs := msg.CountArguments()
	if nargs < 1 {
		Warn("Router.handlePatchXREvent: too few arguments")
		return
	}
	var err error
	if nargs < 3 {
		Warn("handlePatchXREvent: not enough arguments")
		return
	}
	action, _ := argAsString(msg, 0)
	playerName, _ := argAsString(msg, 1)
	switch action {
	case "cursordown":
		Warn("handlePatchXREvent: cursordown ignored")
		return
	case "cursorup":
		Warn("handlePatchXREvent: cursorup ignored")
		return
	}
	// drag
	if nargs < 5 {
		Warn("Not enough arguments for drag")
		return
	}
	x, _ := argAsFloat32(msg, 2)
	y, _ := argAsFloat32(msg, 3)
	z, _ := argAsFloat32(msg, 4)
	s := fmt.Sprintf("\"event\": \"cursor_drag\", \"x\": \"%f\", \"y\": \"%f\", \"z\": \"%f\"", x, y, z)
	var args map[string]string
	args, err = StringMap(s)
	if err != nil {
		return
	}

	// we didn't add "player" to the args, no need for extractPlayer
	err = r.HandleInputEvent(playerName, args)
	if err != nil {
		Warn("Router.handlePatchXREvent", "err", err)
		return
	}
}

func (r *Router) handleOSCSpriteEvent(msg *osc.Message) {

	tags, _ := msg.TypeTags()
	_ = tags
	nargs := msg.CountArguments()
	if nargs < 1 {
		Warn("Router.handleOSCSpriteEvent: too few arguments")
		return
	}
	rawargs, err := argAsString(msg, 0)
	if err != nil {
		Warn("Router.handleOSCSpriteEvent", "err", err)
		return
	}
	if len(rawargs) == 0 || rawargs[0] != '{' {
		Warn("Router.handleOSCSpriteEvent: first char of args must be curly brace")
		return
	}

	args, err := StringMap(rawargs)
	if err != nil {
		Warn("Router.handleOSCSpriteEvent: Unable to process", "args", rawargs)
		return
	}

	playerName := extractPlayer(args)
	err = r.HandleInputEvent(playerName, args)
	if err != nil {
		Warn("Router.handleOSCSpriteEvent", "err", err)
		return
	}
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

var audioResetMutex sync.Mutex

func (r *Router) audioReset() {

	audioResetMutex.Lock()
	defer audioResetMutex.Unlock()

	msg := osc.NewMessage("/play")
	msg.Append(int32(0))
	r.plogueClient.Send(msg)
	// Give Plogue time to react
	time.Sleep(400 * time.Millisecond)
	msg = osc.NewMessage("/play")
	msg.Append(int32(1))
	r.plogueClient.Send(msg)
}

// This silliness is to avoid unused function errors from go-staticcheck
var rr *Router

// var _ = rr.recordPadAPI
var _ = rr.handleOSCSpriteEvent
var _ = argAsInt
var _ = argAsFloat32
