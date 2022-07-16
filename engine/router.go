package engine

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hypebeast/go-osc/osc"
	nats "github.com/nats-io/nats.go"
)

// CurrentMilli is the time from the start, in milliseconds
const defaultClicksPerSecond = 192
const minClicksPerSecond = (defaultClicksPerSecond / 16)
const maxClicksPerSecond = (defaultClicksPerSecond * 16)

var defaultSynth = "P_01_C_01"
var loopForever = 999999

var currentMilli int64
var currentMilliMutex sync.Mutex
var currentMilliOffset int64
var currentClickOffset Clicks
var clicksPerSecond int
var currentClick Clicks
var oneBeat Clicks
var currentClickMutex sync.Mutex

// Bits for Events
const EventMidiInput = 0x01
const EventNoteOutput = 0x02
const EventCursor = 0x04
const EventAll = EventMidiInput | EventNoteOutput | EventCursor

func CurrentClick() Clicks {
	currentClickMutex.Lock()
	defer currentClickMutex.Unlock()
	return currentClick
}

func SetCurrentClick(clk Clicks) {
	currentClickMutex.Lock()
	currentClick = clk
	currentClickMutex.Unlock()
}

// TempoFactor xxx
var TempoFactor = float64(1.0)

func CurrentMilli() int64 {
	currentMilliMutex.Lock()
	defer currentMilliMutex.Unlock()
	return int64(currentMilli)
}

func SetCurrentMilli(m int64) {
	currentMilliMutex.Lock()
	currentMilli = m
	currentMilliMutex.Unlock()
}

// Router takes events and routes them
type Router struct {
	regionLetters string
	motors        map[string]*Motor
	// inputs        []*osc.Client
	OSCInput  chan OSCEvent
	MIDIInput chan MidiEvent
	NoteInput chan *Note

	block        map[string]Block
	blockContext map[string]*EContext
	layerMap     map[string]int // map of pads to resolume layer numbers

	midiEventHandler MIDIEventHandler
	killme           bool // true if Router should be stopped
	lastClick        Clicks
	control          chan Command
	time             time.Time
	time0            time.Time
	killPlayback     bool
	transposeAuto    bool
	transposeNext    Clicks
	transposeBeats   Clicks // time between auto transpose changes
	transposeIndex   int    // current place in tranposeValues
	transposeValues  []int
	recordingOn      bool
	recordingFile    *os.File
	recordingBegun   time.Time
	resolumeClient   *osc.Client
	guiClient        *osc.Client
	plogueClient     *osc.Client
	cursorCount      int
	publishCursor    bool
	publishMIDI      bool
	myHostname       string
	generateVisuals  bool
	generateSound    bool
	// regionForMorph       map[string]string // for all known Morph serial#'s
	regionAssignedToNUID map[string]string
	regionAssignedMutex  sync.RWMutex // covers both regionForMorph and regionAssignedToNUID
	eventMutex           sync.RWMutex
	plugin               map[string]*PluginRef
}

// OSCEvent is an OSC message
type OSCEvent struct {
	Msg    *osc.Message
	Source string
}

// Command is sent on the control channel of the Router
type Command struct {
	Action string // e.g. "addmidi"
	Arg    interface{}
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
*/

// PlaybackEvent is a time-tagged cursor or API event
type PlaybackEvent struct {
	time      float64
	eventType string
	pad       string
	method    string
	args      map[string]string
	rawargs   string
}

var onceRouter sync.Once
var oneRouter Router

// APIExecutorFunc xxx
type APIExecutorFunc func(api string, nuid string, rawargs string) (result interface{}, err error)

// CallAPI calls an API in the Palette Freeframe plugin running in Resolume
func CallAPI(method string, args []string) {
	log.Printf("CallAPI method=%s", method)
}

func recordingsFile(nm string) string {
	return ConfigFilePath(filepath.Join("recordings", nm))
}

// TheRouter returns a pointer to the one-and-only Router
func TheRouter() *Router {
	onceRouter.Do(func() {

		oneRouter.regionLetters = ConfigValue("pads")
		if oneRouter.regionLetters == "" {
			log.Printf("No value for pads, assuming ABCD")
			oneRouter.regionLetters = "ABCD"
		}

		err := LoadParamEnums()
		if err != nil {
			log.Printf("LoadParamEnums: err=%s\n", err)
			// might be fatal, but try to continue
		}
		err = LoadParamDefs()
		if err != nil {
			log.Printf("LoadParamDefs: err=%s\n", err)
			// might be fatal, but try to continue
		}
		err = LoadResolumeJSON()
		if err != nil {
			log.Printf("LoadResolumeJSON: err=%s\n", err)
			// might be fatal, but try to continue
		}

		oneRouter.motors = make(map[string]*Motor)
		// oneRouter.regionForMorph = make(map[string]string)
		oneRouter.regionAssignedToNUID = make(map[string]string)
		oneRouter.block = make(map[string]Block)
		oneRouter.blockContext = make(map[string]*EContext)
		oneRouter.plugin = make(map[string]*PluginRef)

		resolumePort := 7000
		guiPort := 3943
		ploguePort := 3210

		oneRouter.resolumeClient = osc.NewClient("127.0.0.1", resolumePort)
		oneRouter.guiClient = osc.NewClient("127.0.0.1", guiPort)
		oneRouter.plogueClient = osc.NewClient("127.0.0.1", ploguePort)

		log.Printf("OSC client ports: resolume=%d gui=%d plogue=%d\n", resolumePort, guiPort, ploguePort)

		for i, c := range oneRouter.regionLetters {
			resolumeLayer := oneRouter.ResolumeLayerForPad(string(c))
			ffglPort := 3334 + i
			ch := string(c)
			freeframeClient := osc.NewClient("127.0.0.1", ffglPort)
			log.Printf("OSC freeframeClient: port=%d layer=%d\n", ffglPort, resolumeLayer)
			oneRouter.motors[ch] = NewMotor(ch, resolumeLayer, freeframeClient, oneRouter.resolumeClient, oneRouter.guiClient)
			// log.Printf("Pad %s created, resolumeLayer=%d resolumePort=%d\n", ch, resolumeLayer, resolumePort)
		}

		oneRouter.OSCInput = make(chan OSCEvent)
		oneRouter.MIDIInput = make(chan MidiEvent)
		oneRouter.NoteInput = make(chan *Note)
		oneRouter.recordingOn = false

		// Transpose control
		oneRouter.transposeAuto = ConfigBoolWithDefault("transposeauto", true)
		log.Printf("oneRouter.transposeAuto=%v\n", oneRouter.transposeAuto)
		oneRouter.transposeBeats = Clicks(ConfigIntWithDefault("transposebeats", 48))
		oneRouter.transposeNext = oneRouter.transposeBeats * oneBeat // first one

		// NOTE: the order of values here needs to match the
		// order in palette.py
		oneRouter.transposeValues = []int{0, -2, 3, -5}

		oneRouter.myHostname = ConfigValue("hostname")
		if oneRouter.myHostname == "" {
			hostname, err := os.Hostname()
			if err != nil {
				log.Printf("os.Hostname: err=%s\n", err)
				hostname = "unknown"
			}
			oneRouter.myHostname = hostname
		}

		oneRouter.publishCursor = ConfigBool("publishcursor")
		oneRouter.publishMIDI = ConfigBool("publishmidi")
		oneRouter.generateVisuals = ConfigBool("generatevisuals")
		oneRouter.generateSound = ConfigBool("generatesound")

		for _, motor := range oneRouter.motors {
			motor.restoreCurrentSnap()
		}

		go oneRouter.notifyGUI("restart")
	})
	return &oneRouter
}

func (r *Router) ResolumeLayerForText() int {
	defLayer := "5"
	s := ConfigStringWithDefault("textlayer", defLayer)
	layernum, err := strconv.Atoi(s)
	if err != nil {
		log.Printf("Bad format for textlayer value")
		layernum, _ = strconv.Atoi(defLayer)
	}
	return layernum
}

func (r *Router) ResolumeLayerForPad(pad string) int {
	if r.layerMap == nil {
		regionLayers := "1,2,3,4"
		s := ConfigStringWithDefault("regionlayers", regionLayers)
		layers := strings.Split(s, ",")
		if len(layers) != 4 {
			log.Printf("ResolumeLayerForPad: regionlayers value needs 4 values\n")
			layers = strings.Split(regionLayers, ",")
		}
		r.layerMap = make(map[string]int)
		r.layerMap["A"], _ = strconv.Atoi(layers[0])
		r.layerMap["B"], _ = strconv.Atoi(layers[1])
		r.layerMap["C"], _ = strconv.Atoi(layers[2])
		r.layerMap["D"], _ = strconv.Atoi(layers[3])

		log.Printf("Router: Resolume layerMap = %+v\n", r.layerMap)
	}
	return r.layerMap[pad]
}

// StartOSC xxx
func (r *Router) StartOSC(source string) {

	handler := r.OSCInput

	d := osc.NewStandardDispatcher()

	err := d.AddMsgHandler("*", func(msg *osc.Message) {
		handler <- OSCEvent{Msg: msg, Source: source}
	})
	if err != nil {
		log.Printf("ERROR! %s\n", err.Error())
	}

	server := &osc.Server{
		Addr:       source,
		Dispatcher: d,
	}
	server.ListenAndServe()
}

// StartNATSClient xxx
func (r *Router) StartNATSClient() {

	// ConnectToNATSServer()
	// Hand all NATS messages to HandleAPI

	log.Printf("StartNATS: Subscribing to %s\n", PaletteAPISubject)
	SubscribeNATS(PaletteAPISubject, func(msg *nats.Msg) {
		data := string(msg.Data)
		response := r.handleAPIInput(data)
		msg.Respond([]byte(response))
	})

	log.Printf("StartNATS: subscribing to %s\n", PaletteEventSubject)
	SubscribeNATS(PaletteEventSubject, func(msg *nats.Msg) {
		data := string(msg.Data)
		args, err := StringMap(data)
		if err != nil {
			log.Printf("HandleSubscribedEvent: err=%s\n", err)
		}
		err = r.HandleEvent(args)
		if err != nil {
			log.Printf("HandleSubscribedEvent: err=%s\n", err)
		}
	})
}

// TimeString returns time and clicks
func TimeString() string {
	r := TheRouter()
	sofar := r.time.Sub(r.time0)
	click := Seconds2Clicks(sofar.Seconds())
	return fmt.Sprintf("sofar=%f click=%d", sofar.Seconds(), click)

}

// StartRealtime runs the looper and never returns
func (r *Router) StartRealtime() {

	log.Println("StartRealtime begins")

	// Wake up every 2 milliseconds and check looper events
	tick := time.NewTicker(2 * time.Millisecond)
	r.time0 = <-tick.C

	var nextAliveSecs int

	// Don't start checking processes right away, after killing them on a restart,
	// they may still be running for a bit
	var nextCheckSecs int = 2

	var aliveIntervalSeconds = ConfigIntWithDefault("aliveinterval", 30)
	var processcheckSeconds = ConfigIntWithDefault("processcheckinterval", 60)

	// By reading from tick.C, we wake up every 2 milliseconds
	for now := range tick.C {
		// log.Printf("Realtime loop now=%v\n", time.Now())
		r.time = now
		sofar := now.Sub(r.time0)
		secs := sofar.Seconds()
		newclick := Seconds2Clicks(secs)
		SetCurrentMilli(int64(secs * 1000.0))

		currentClick := CurrentClick()
		if newclick > currentClick {
			// log.Printf("ADVANCING CLICK now=%v click=%d\n", time.Now(), newclick)
			r.advanceTransposeTo(newclick)
			r.advanceClickTo(currentClick)
			SetCurrentClick(newclick)
		}

		// every so often send out an "alive" event
		// with a cursorCount since the last one
		if int(secs) > nextAliveSecs {
			nextAliveSecs += aliveIntervalSeconds
			PublishAliveEvent(secs, r.cursorCount)
			r.cursorCount = 0
		}

		// Every so often check to see if necessary
		// processes (like Resolume) are still running
		if processcheckSeconds > 0 && int(secs) > nextCheckSecs {
			nextCheckSecs += processcheckSeconds
			// Put it in background, so calling
			// tasklist or ps doesn't disrupt realtime
			go CheckProcessesAndRestartIfNecessary()
		}

		select {
		case cmd := <-r.control:
			_ = cmd
			log.Println("Realtime got command on control channel: ", cmd)
		default:
		}
	}
	log.Println("StartRealtime ends")
}

func (r *Router) advanceTransposeTo(newclick Clicks) {
	if r.transposeAuto && r.transposeNext < newclick {
		r.transposeNext += (r.transposeBeats * oneBeat)
		r.transposeIndex = (r.transposeIndex + 1) % len(r.transposeValues)
		transposePitch := r.transposeValues[r.transposeIndex]
		if Debug.Transpose {
			log.Printf("advanceTransposeTo: newclick=%d transposePitch=%d\n", newclick, transposePitch)
		}
		for _, motor := range r.motors {
			// motor.clearDown()
			if Debug.Transpose {
				log.Printf("  setting transposepitch in motor pad=%s trans=%d activeNotes=%d\n", motor.padName, transposePitch, len(motor.activeNotes))
			}
			motor.terminateActiveNotes()
			motor.TransposePitch = transposePitch
		}
	}
}

type MIDIEventHandler func(MidiEvent)

func (r *Router) SetMIDIEventHandler(handler MIDIEventHandler) {
	r.midiEventHandler = handler
}

// StartMIDI listens for MIDI events and sends their bytes to the MIDIInput chan
func (r *Router) StartMIDI() {
	if EraeEnabled {
		r.SetMIDIEventHandler(HandleEraeMIDI)
	}
	inputs := MIDI.InputMap()
	for {
		for nm, input := range inputs {
			hasinput, err := input.Poll()
			if err != nil {
				log.Printf("StartMIDI: Poll input=%s err=%s\n", nm, err)
				continue
			}
			if !hasinput {
				continue
			}
			events, err := input.ReadEvents()
			if err != nil {
				log.Printf("StartMIDI: ReadEvent input=%s err=%s\n", nm, err)
				continue
			}
			// Feed the MIDI bytes to r.MIDIInput one byte at a time
			for _, event := range events {
				r.MIDIInput <- event
				if r.midiEventHandler != nil {
					r.midiEventHandler(event)
				}
			}
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// InputListener listens for local device inputs (OSC, MIDI)
// We could have separate goroutines for the different inputs, but doing
// them in a single select eliminates some need for locking.
func (r *Router) InputListener() {
	for !r.killme {
		select {
		case msg := <-r.OSCInput:
			r.handleOSCInput(msg)
		case event := <-r.MIDIInput:
			if r.publishMIDI {
				log.Printf("InputListener: publicMIDI does nothing!!\n")
				if event.SysEx != nil {
					// we don't publish sysex
				} else {
					me := MIDIDeviceEvent{
						Timestamp: int64(event.Timestamp),
						Status:    event.Status,
						Data1:     event.Data1,
						Data2:     event.Data2,
					}
					err := PublishMIDIDeviceEvent(me)
					if err != nil {
						log.Printf("InputListener: me=%+v err=%s\n", me, err)
					}
				}
			}
			for _, motor := range r.motors {
				motor.HandleMIDIInput(event)
			}
			if Debug.MIDI {
				log.Printf("InputListener: MIDIInput event=0x%02x\n", event)
			}
		default:
			// log.Printf("Sleeping 1 ms - now=%v\n", time.Now())
			time.Sleep(time.Millisecond)
		}
	}
	log.Printf("InputListener is being killed\n")
}

// HandleSubscribedEventArgs xxx
func (r *Router) HandleEvent(args map[string]string) error {

	r.eventMutex.Lock()
	defer r.eventMutex.Unlock()

	// All Events should have nuid and event values

	fromNUID, err := needStringArg("nuid", "HandleEvent", args)
	if err != nil {
		return err
	}

	/*
		// Ignore anything from myself
		if fromNUID == MyNUID() {
			return nil
		}
	*/

	event, err := needStringArg("event", "HandleEvent", args)
	if err != nil {
		return err
	}

	// If no "region" argument, use one assigned to NUID
	region := optionalStringArg("region", args, "")
	if region == "" {
		region = r.getRegionForNUID(fromNUID)
	} else {
		// Remove it from the args
		delete(args, "region")
	}

	if Debug.Router {
		log.Printf("Router.HandleEvent: region=%s event=%s\n", region, event)
	}

	motor, ok := r.motors[region]
	if !ok {
		return fmt.Errorf("there is no region named %s", region)
	}

	switch event {

	case "engine":
		log.Printf("Router: ignoring engine event\n")
		return nil

	case "cursor_down", "cursor_drag", "cursor_up":

		// If we're publishing cursor events, we ignore ones from ourself
		if r.publishCursor && fromNUID == MyNUID() {
			return nil
		}

		cid := optionalStringArg("cid", args, "UnspecifiedCID")

		subEvent := event[7:] // assumes event is cursor_*
		switch subEvent {
		case "down", "drag", "up":
		default:
			return fmt.Errorf("handleSubscribedEvent: Unexpected cursor event type: %s", subEvent)
		}

		x, y, z, err := r.getXYZ("handleSubscribedEvent", args)
		if err != nil {
			return err
		}

		ce := CursorDeviceEvent{
			NUID:      fromNUID,
			Region:    region,
			CID:       cid,
			Timestamp: CurrentMilli(),
			Ddu:       subEvent,
			X:         x,
			Y:         y,
			Z:         z,
			Area:      0.0,
		}

		if r.publishCursor {
			err := PublishCursorDeviceEvent(ce)
			if err != nil {
				log.Printf("Router.routeCursorDeviceEvent: NATS publishing err=%s\n", err)
			}
		}

		motor.handleCursorDeviceEvent(ce)

	case "sprite":

		api := "event.sprite"
		x, y, z, err := r.getXYZ(api, args)
		if err != nil {
			return nil
		}
		motor.generateSprite("dummy", x, y, z)

	case "midi_reset":
		log.Printf("HandleEvent: midi_reset, sending ANO\n")
		motor.HandleMIDITimeReset()
		motor.sendANO()

	case "audio_reset":
		log.Printf("HandleEvent: audio_reset!!\n")
		go r.audioReset()

	case "midi":
		// If we're publishing midi events, we ignore ones from ourself
		if r.publishMIDI && fromNUID == MyNUID() {
			return nil
		}

		bytes, err := needStringArg("bytes", "HandleEvent", args)
		if err != nil {
			return err
		}
		me, err := r.makeMIDIEvent("subscribed", bytes, args)
		if err != nil {
			return err
		}
		motor.HandleMIDIInput(*me)
	}

	return nil
}

func (r *Router) handleCursorDeviceInput(e CursorDeviceEvent) {

	r.eventMutex.Lock()
	defer r.eventMutex.Unlock()

	if r.publishCursor {
		err := PublishCursorDeviceEvent(e)
		if err != nil {
			log.Printf("Router.routeCursorDeviceEvent: NATS publishing err=%s\n", err)
		}
	}

	motor, ok := r.motors[e.Region]
	if !ok {
		log.Printf("routeCursorDeviceEvent: no region named %s, unable to process ce=%+v\n", e.Region, e)
		return
	}
	r.cursorCount++
	motor.handleCursorDeviceEvent(e)
}

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

func (r *Router) getXYZ(api string, args map[string]string) (x, y, z float32, err error) {

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

// HandleAPIInput xxx
func (r *Router) handleAPIInput(data string) (response string) {

	r.eventMutex.Lock()
	defer r.eventMutex.Unlock()

	smap, err := StringMap(data)

	defer func() {
		if Debug.API {
			log.Printf("Router.HandleAPI: response=%s\n", response)
		}
	}()

	if err != nil {
		response = ErrorResponse(err)
		return
	}
	api, ok := smap["api"]
	if !ok {
		response = ErrorResponse(fmt.Errorf("missing api parameter"))
		return
	}
	fromNUID, ok := smap["nuid"]
	if !ok {
		response = ErrorResponse(fmt.Errorf("missing nuid parameter"))
		return
	}
	rawargs, ok := smap["params"]
	if !ok {
		response = ErrorResponse(fmt.Errorf("missing params parameter"))
		return
	}
	if Debug.API {
		log.Printf("Router.HandleAPI: api=%s args=%s\n", api, rawargs)
	}
	result, err := r.ExecuteAPI(api, fromNUID, rawargs)
	if err != nil {
		response = ErrorResponse(err)
	} else {
		response = ResultResponse(result)
	}
	return
}

// HandleOSCInput xxx
func (r *Router) handleOSCInput(e OSCEvent) {

	// r.eventMutex.Lock()
	// defer r.eventMutex.Unlock()

	if Debug.OSC {
		log.Printf("Router.HandleOSCInput: msg=%s\n", e.Msg.String())
	}
	switch e.Msg.Address {

	case "/clientrestart":
		r.handleClientRestart(e.Msg)

	case "/api":
		log.Printf("OSC /api is not implemented\n")
		// r.handleOSCAPI(e.Msg)

	case "/event": // These messages encode the arguments as JSON
		r.handleOSCEvent(e.Msg)

	case "/cursor":
		r.handleMMTTCursor(e.Msg)

	case "/patchxr":
		r.handlePatchXREvent(e.Msg)

	default:
		log.Printf("Router.HandleOSCInput: Unrecognized OSC message source=%s msg=%s\n", e.Source, e.Msg)
	}
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

func (r *Router) advanceClickTo(toClick Clicks) {

	// Don't let events get handled while we're advancing
	r.eventMutex.Lock()
	defer r.eventMutex.Unlock()

	for clk := r.lastClick; clk < toClick; clk++ {
		for _, motor := range r.motors {
			if (clk % oneBeat) == 0 {
				motor.checkCursorUp()
			}
			motor.AdvanceByOneClick()
		}
	}
	r.lastClick = toClick
}

func (r *Router) recordEvent(eventType string, pad string, method string, args string) {
	if !r.recordingOn {
		log.Printf("HEY! recordEvent called when recordingOn is false!?\n")
		return
	}
	if r.recordingFile == nil {
		log.Printf("HEY! recordEvent called when recordingFile is nil!?\n")
		return
	}
	if args[0] != '{' {
		log.Printf("HEY! first char of args in recordEvent needs to be a curly brace!\n")
		return
	}
	if r.recordingFile != nil {
		dt := time.Since(r.recordingBegun).Seconds()
		r.recordingFile.WriteString(
			fmt.Sprintf("%.6f %s %s %s ", dt, eventType, pad, method) + args + "\n")
		r.recordingFile.Sync()
	}
}

func (r *Router) recordPadAPI(data string) {
	if r.recordingOn {
		r.recordEvent("pad", "*", "api", data)
	}
}

func (r *Router) recordingSave(name string) error {
	if r.recordingFile != nil {
		r.recordingFile.Close()
		r.recordingFile = nil
	}

	source, err := os.Open(recordingsFile("LastRecording.json"))
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(recordingsFile(fmt.Sprintf("%s.json", name)))
	if err != nil {
		return err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	if err != nil {
		return err
	}
	log.Printf("nbytes copied = %d\n", nBytes)
	return nil
}

func (r *Router) recordingPlaybackStop() {
	r.killPlayback = true
	r.sendANO()
}

func (r *Router) sendANO() {
	for _, motor := range r.motors {
		motor.sendANO()
	}
}

func (r *Router) notifyGUI(eventName string) {
	if !ConfigBool("notifygui") {
		return
	}
	msg := osc.NewMessage("/notify")
	msg.Append(eventName)
	r.guiClient.Send(msg)
	if Debug.OSC {
		log.Printf("Router.notifyGUI: msg=%v\n", msg)
	}
}

func (r *Router) recordingPlayback(events []*PlaybackEvent) error {

	// XXX - WARNING, this code hasn't been exercised in a LONG time,
	// XXX - someday it'll probably be resurrected.
	log.Printf("recordingPlay, #events = %d\n", len(events))
	r.killPlayback = false

	r.notifyGUI("start")

	playbackBegun := time.Now()

	for _, pe := range events {
		if r.killPlayback {
			log.Printf("killPlayback!\n")
			r.sendANO()
			break
		}
		if pe != nil {
			eventTime := (*pe).time // in seconds
			for time.Since(playbackBegun).Seconds() < eventTime {
				time.Sleep(time.Millisecond)
			}
			eventType := (*pe).eventType
			pad := (*pe).pad
			var motor *Motor
			if pad != "*" {
				motor = r.motors[pad]
			}
			method := (*pe).method
			// time := (*pe).time
			args := (*pe).args
			rawargs := (*pe).rawargs
			// log.Printf("eventType=%s method=%s\n", eventType, method)
			switch eventType {
			case "cursor":
				id := args["id"]
				x := args["x"]
				y := args["y"]
				z := args["z"]
				ddu := method
				xf, _ := ParseFloat32(x, "cursor.x")
				yf, _ := ParseFloat32(y, "cursor.y")
				zf, _ := ParseFloat32(z, "cursor.z")
				motor.executeIncomingCursor(CursorStepEvent{
					ID:  id,
					X:   xf,
					Y:   yf,
					Z:   zf,
					Ddu: ddu,
				})
			case "api":
				// since we already have args
				motor.ExecuteAPI(method, args, rawargs)
			case "global":
				log.Printf("NOT doing anying for global playback, method=%s\n", method)
			default:
				log.Printf("Unknown eventType=%s in recordingPlay\n", eventType)
			}
		}
	}
	log.Printf("recordingPlay has finished!\n")
	r.notifyGUI("stop")
	return nil
}

func (r *Router) recordingLoad(name string) ([]*PlaybackEvent, error) {
	file, err := os.Open(recordingsFile(fmt.Sprintf("%s.json", name)))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	events := make([]*PlaybackEvent, 0)
	nlines := 0
	var line string
	for {
		line, err = reader.ReadString('\n')
		if err != nil {
			break
		}
		nlines++
		words := strings.SplitN(line, " ", 5)
		if len(words) < 4 {
			log.Printf("recordings line has less than 3 words? line=%s\n", line)
			continue
		}
		evTime, err := strconv.ParseFloat(words[0], 64)
		if err != nil {
			log.Printf("Unable to convert time in playback file: %s\n", words[0])
			continue
		}
		playbackType := words[1]
		switch playbackType {
		case "cursor", "sound", "visual", "effect", "global", "perform":
			pad := words[2]
			method := words[3]
			rawargs := words[4]
			args, err := StringMap(rawargs)
			if err != nil {
				log.Printf("Unable to parse JSON: %s\n", rawargs)
				continue
			}
			pe := &PlaybackEvent{time: evTime, eventType: playbackType, pad: pad, method: method, args: args, rawargs: rawargs}
			events = append(events, pe)
		default:
			log.Printf("Unknown playbackType in playback file: %s\n", playbackType)
			continue
		}
	}
	if err != io.EOF {
		return nil, err
	}
	log.Printf("Number of playback events is %d, lines is %d\n", len(events), nlines)
	return events, nil
}

func (r *Router) handleMMTTButton(butt string) {
	preset := ConfigStringWithDefault(butt, "")
	if preset == "" {
		log.Printf("No Preset assigned to BUTTON %s, using Perky_Shapes\n", butt)
		preset = "Perky_Shapes"
		return
	}
	log.Printf("Router.handleMMTTButton: butt=%s preset=%s\n", butt, preset)
	err := r.loadQuadPreset("quad."+preset, "*")
	if err != nil {
		log.Printf("handleMMTTButton: preset=%s err=%s\n", preset, err)
	}

	text := strings.ReplaceAll(preset, "_", "\n")
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
		log.Printf("Router.handleOSCEvent: too few arguments\n")
		return
	}
	// Even though the argument is an integer port number,
	// it's a string in the OSC message sent from the Palette FFGL plugin.
	s, err := argAsString(msg, 0)
	if err != nil {
		log.Printf("Router.handleOSCEvent: err=%s\n", err)
		return
	}
	portnum, err := strconv.Atoi(s)
	if err != nil {
		log.Printf("Router.handleOSCEvent: Atoi err=%s\n", err)
		return
	}
	var found *Motor
	for _, motor := range r.motors {
		if motor.freeframeClient.Port() == portnum {
			found = motor
			break
		}
	}
	if found == nil {
		log.Printf("handleClientRestart unable to find Motor with portnum=%d\n", portnum)
	} else {
		found.sendAllParameters()
	}
}

// handleMMTTCursor handles messages from MMTT and reformats them
// as a standard cursor event before sending them to r.HandleEvent
func (r *Router) handleMMTTCursor(msg *osc.Message) {

	tags, _ := msg.TypeTags()
	_ = tags
	nargs := msg.CountArguments()
	if nargs < 1 {
		log.Printf("Router.handleMMTTCursor: too few arguments\n")
		return
	}
	ddu, err := argAsString(msg, 0)
	if err != nil {
		log.Printf("Router.handleMMTTEvent: err=%s\n", err)
		return
	}
	cid, err := argAsString(msg, 1)
	if err != nil {
		log.Printf("Router.handleMMTTEvent: err=%s\n", err)
		return
	}
	region := "A"
	words := strings.Split(cid, ".")
	if len(words) > 1 {
		region = words[0]
	}
	x, err := argAsFloat32(msg, 2)
	if err != nil {
		log.Printf("Router.handleMMTTEvent: x err=%s\n", err)
		return
	}
	y, err := argAsFloat32(msg, 3)
	if err != nil {
		log.Printf("Router.handleMMTTEvent: y err=%s\n", err)
		return
	}
	z, err := argAsFloat32(msg, 4)
	if err != nil {
		log.Printf("Router.handleMMTTEvent: z err=%s\n", err)
		return
	}

	motor, mok := r.motors[region]
	if !mok {
		// If it's not a region, it's a button.
		buttonDepth := ConfigFloatWithDefault("mmttbuttondepth", 0.05)
		if z > buttonDepth {
			log.Printf("NOT triggering button too deep z=%f buttonDepth=%f\n", z, buttonDepth)
			return
		}
		if ddu == "down" {
			if Debug.MMTT {
				log.Printf("MMT BUTTON TRIGGERED buttonDepth=%f  z=%f\n", buttonDepth, z)
			}
			r.handleMMTTButton(region)
		}
		return
	}

	ce := CursorDeviceEvent{
		NUID:      MyNUID(),
		Region:    region,
		CID:       cid,
		Timestamp: CurrentMilli(),
		Ddu:       ddu,
		X:         x,
		Y:         y,
		Z:         z,
		Area:      0.0,
	}
	// XXX - HACK!!
	zfactor := ConfigFloatWithDefault("mmttzfactor", 3.0)
	ahack := ConfigFloatWithDefault("mmttahack", 1.0)
	ce.Z = boundval(ahack * zfactor * ce.Z)

	xexpand := ConfigFloatWithDefault("mmttxexpand", 1.0)
	ce.X = boundval(((ce.X - 0.5) * xexpand) + 0.5)

	yexpand := ConfigFloatWithDefault("mmttyexpand", 1.0)
	ce.Y = boundval(((ce.Y - 0.5) * yexpand) + 0.5)

	if Debug.MMTT && Debug.Cursor {
		log.Printf("MMTT Cursor %s %s xyz= %f %f %f\n", ce.CID, ce.Ddu, ce.X, ce.Y, ce.Z)
	}

	motor.handleCursorDeviceEvent(ce)
}

func boundval(v float32) float32 {
	if v < 0.0 {
		return 0.0
	}
	if v > 1.0 {
		return 1.0
	}
	return v
}

func (r *Router) handleOSCEvent(msg *osc.Message) {

	tags, _ := msg.TypeTags()
	_ = tags
	nargs := msg.CountArguments()
	if nargs < 1 {
		log.Printf("Router.handleOSCEvent: too few arguments\n")
		return
	}
	rawargs, err := argAsString(msg, 0)
	if err != nil {
		log.Printf("Router.handleOSCEvent: err=%s\n", err)
		return
	}
	if len(rawargs) == 0 || rawargs[0] != '{' {
		log.Printf("Router.handleOSCEvent: first char of args must be curly brace\n")
		return
	}

	// Add the required nuid argument, which OSC input doesn't provide.
	// It's "unknown" because we don't have access to info about the originator.
	fromNUID := "unknown"
	newrawargs := "{ \"nuid\": \"" + fromNUID + "\", " + rawargs[1:]
	var args map[string]string
	args, err = StringMap(newrawargs)
	if err != nil {
		return
	}

	err = r.HandleEvent(args)
	if err != nil {
		log.Printf("Router.handleOSCEvent: err=%s\n", err)
		return
	}
}

func (r *Router) handlePatchXREvent(msg *osc.Message) {

	tags, _ := msg.TypeTags()
	_ = tags
	nargs := msg.CountArguments()
	if nargs < 1 {
		log.Printf("Router.handlePatchXREvent: too few arguments\n")
		return
	}
	var err error
	// Add the required nuid argument, which OSC input doesn't provide
	if nargs < 3 {
		log.Printf("handlePatchXREvent: not enough arguments\n")
		return
	}
	action, _ := argAsString(msg, 0)
	region, _ := argAsString(msg, 1)
	switch action {
	case "cursordown":
		log.Printf("handlePatchXREvent: cursordown ignored\n")
		return
	case "cursorup":
		log.Printf("handlePatchXREvent: cursorup ignored\n")
		return
	}
	// drag
	if nargs < 5 {
		log.Printf("Not enough arguments for drag\n")
		return
	}
	x, _ := argAsFloat32(msg, 2)
	y, _ := argAsFloat32(msg, 3)
	z, _ := argAsFloat32(msg, 4)
	s := fmt.Sprintf("{ \"nuid\": \"%s\", \"event\": \"cursor_drag\", \"region\": \"%s\", \"x\": \"%f\", \"y\": \"%f\", \"z\": \"%f\" }", MyNUID(), region, x, y, z)
	var args map[string]string
	args, err = StringMap(s)
	if err != nil {
		return
	}

	err = r.HandleEvent(args)
	if err != nil {
		log.Printf("Router.handlePatchXREvent: err=%s\n", err)
		return
	}
}

func (r *Router) handleOSCSpriteEvent(msg *osc.Message) {

	tags, _ := msg.TypeTags()
	_ = tags
	nargs := msg.CountArguments()
	if nargs < 1 {
		log.Printf("Router.handleOSCSpriteEvent: too few arguments\n")
		return
	}
	rawargs, err := argAsString(msg, 0)
	if err != nil {
		log.Printf("Router.handleOSCSpriteEvent: err=%s\n", err)
		return
	}
	if len(rawargs) == 0 || rawargs[0] != '{' {
		log.Printf("Router.handleOSCSpriteEvent: first char of args must be curly brace\n")
		return
	}

	// Add the required nuid argument, which OSC input doesn't provide
	newrawargs := "{ \"nuid\": \"" + MyNUID() + "\", " + rawargs[1:]
	args, err := StringMap(newrawargs)
	if err != nil {
		log.Printf("Router.handleOSCSpriteEvent: Unable to process args=%s\n", newrawargs)
		return
	}

	err = r.HandleEvent(args)
	if err != nil {
		log.Printf("Router.handleOSCSpriteEvent: err=%s\n", err)
		return
	}
}

// No error return because it's OSC
func (r *Router) handleOSCAPI(msg *osc.Message) {
	tags, _ := msg.TypeTags()
	_ = tags
	nargs := msg.CountArguments()
	if nargs < 1 {
		log.Printf("Router.handleOSCAPI: too few arguments\n")
		return
	}
	api, err := argAsString(msg, 0)
	if err != nil {
		log.Printf("Router.handleOSCAPI: err=%s\n", err)
		return
	}
	rawargs, err := argAsString(msg, 1)
	if err != nil {
		log.Printf("Router.handleOSCAPI: err=%s\n", err)
		return
	}
	if len(rawargs) == 0 || rawargs[0] != '{' {
		log.Printf("Router.handleOSCAPI: first char of args must be curly brace\n")
		return
	}
	_, err = r.ExecuteAPI(api, MyNUID(), rawargs)
	if err != nil {
		log.Printf("Router.handleOSCAPI: err=%s", err)
	}
}

// availableRegion - return the name of a region that hasn't been assigned to a remote yet
// It is assumed that we have a mutex on r.regionAssignedToSource
func (r *Router) availableRegion(source string) string {

	nregions := len(r.regionLetters)
	alreadyAssigned := make([]bool, nregions)
	for _, v := range r.regionAssignedToNUID {
		i := strings.Index(r.regionLetters, v)
		if i >= 0 {
			alreadyAssigned[i] = true
		}
	}
	for i, used := range alreadyAssigned {
		if !used {
			avail := r.regionLetters[i : i+1]
			if Debug.Cursor {
				log.Printf("Router.assignRegion: %s is assigned to source=%s\n", avail, source)
			}
			return avail
		}
	}
	if Debug.Cursor {
		log.Printf("Router.assignRegion: No regions available\n")
	}
	return ""
}

type PluginRef struct {
	pluginid        string
	Events          uint // see Event* bits
	Active          bool
	forwardToPlugin chan interface{}
	killme          bool
}

func (r *Router) NewPluginRef(pluginid string, events uint) *PluginRef {
	ref := &PluginRef{
		pluginid:        pluginid,
		Events:          events,
		Active:          false,
		forwardToPlugin: make(chan interface{}),
		killme:          false,
	}
	return ref
}

/*
func (r *Router) pluginCallback(msg *nats.Msg) {
	data := string(msg.Data)
	log.Printf("Router.pluginCallback: data=%s\n", data)
	args, err := StringMap(data)
	if err != nil {
		log.Printf("natsCallback: err=%s\n", err)
		return
	}
	notestr, err := needStringArg("note", "pluginCallback", args)
	if err != nil {
		log.Printf("natsCallback: err=%s\n", err)
		return
	}
	log.Printf("Router.pluginCallback: note=%s\n", notestr)
}
*/

func (r *Router) registerPlugin(pluginid string, events string) error {
	_, pok := r.plugin[pluginid]
	if pok {
		return fmt.Errorf("registerPlugin: there is already a plugin %s", pluginid)
	}
	bits, err := events2bits(events)
	if err != nil {
		return err
	}
	log.Printf("Router.registerPlugin: pluginid=%s events=%s\n", pluginid, events)
	p := r.NewPluginRef(pluginid, bits)
	go r.PluginForwarder(p)
	r.plugin[pluginid] = p

	// Listen for NATS events from it
	// SubscribeNATS(PluginInputSubject(pluginid), r.pluginCallback)
	return nil
}

func (r *Router) PluginForwarder(ref *PluginRef) {
	if Debug.Plugin {
		log.Printf("PluginForwarder: plugin=%s started\n", ref.pluginid)
	}
	for !ref.killme {
		// Read from forwardToPlugin and send to the plugin via NATS
		msg := <-ref.forwardToPlugin
		switch note := msg.(type) {
		case *Note:
			if Debug.Plugin {
				log.Printf("PluginForwarder: plugin=%s note=%v\n", ref.pluginid, *note)
			}
			PublishNote(PluginInputSubject(ref.pluginid), note, "output.engine")
		default:
			log.Printf("PluginForwarder: unknown type received on forwardToPlugin\n")
		}
	}
	log.Printf("PluginForwarder: plugin=%s ended\n", ref.pluginid)
}

func events2bits(events string) (bits uint, err error) {
	words := strings.Split(events, ",")
	for _, w := range words {
		switch w {
		case "midiinput":
			bits |= EventMidiInput
		case "midioutput":
			bits |= EventNoteOutput
		case "cursor":
			bits |= EventCursor
		case "all":
			bits |= (EventMidiInput | EventNoteOutput | EventCursor)
		default:
			return 0, fmt.Errorf("unknown event name: %s", w)
		}
	}
	return bits, nil
}

func (r *Router) getRegionForNUID(nuid string) string {

	r.regionAssignedMutex.Lock()
	defer r.regionAssignedMutex.Unlock()

	// See if we've already assigned a region to this source
	region, ok := r.regionAssignedToNUID[nuid]
	if ok {
		return region
	}

	// Hasn't been seen yet, let's get an available region
	region = r.availableRegion(nuid)
	// log.Printf("getRegionForNUID: Assigning region=%s to nuid=%s\n", region, nuid)
	r.regionAssignedToNUID[nuid] = region

	return region
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
var rr *Router
var _ = rr.recordPadAPI
var _ = rr.handleOSCSpriteEvent
var _ = rr.handleOSCAPI
var _ = argAsInt
var _ = argAsFloat32
