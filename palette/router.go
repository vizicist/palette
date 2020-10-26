package palette

import (
	"bufio"
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
	"github.com/vizicist/portmidi"
)

const debug bool = false

// Router takes events and routes them
type Router struct {
	reactors        map[string]*Reactor
	inputs          []*osc.Client
	OSCInput        chan OSCEvent
	MIDIInput       chan portmidi.Event
	MIDINumDown     int
	MIDIOctaveShift int
	// MIDISetScale      bool
	MIDIThru               string // "", "A", "B", "C", "D"
	MIDIThruScadjust       bool
	UseExternalScale       bool // if true, scadjust uses "external" Scale
	MIDIQuantized          bool
	killme                 bool // set to true if Router should be stopped
	lastClick              Clicks
	control                chan Command
	time                   time.Time
	time0                  time.Time
	killPlayback           bool
	recordingOn            bool
	recordingFile          *os.File
	recordingBegun         time.Time
	resolumeClient         *osc.Client
	guiClient              *osc.Client
	plogueClient           *osc.Client
	publishCursor          bool // e.g. "nats"
	paletteHost            string
	isPaletteHost          bool
	myHostname             string
	generateVisuals        bool
	generateSound          bool
	regionForSource        map[string]string // for all known Morph serial#'s
	regionAssignedToSource map[string]string
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

var onceRouter sync.Once
var oneRouter Router

// Executor xxx
type Executor func(api string, rawargs string) (result interface{}, err error)

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

		LoadParamEnums()
		LoadParamDefs()
		LoadEffectsJSON()

		ClearExternalScale()
		SetExternalScale(60%12, true) // Middle C

		oneRouter.reactors = make(map[string]*Reactor)
		oneRouter.regionForSource = make(map[string]string)
		oneRouter.regionAssignedToSource = make(map[string]string)

		freeframeClientA := osc.NewClient("127.0.0.1", 3334)
		freeframeClientB := osc.NewClient("127.0.0.1", 3335)
		freeframeClientC := osc.NewClient("127.0.0.1", 3336)
		freeframeClientD := osc.NewClient("127.0.0.1", 3337)

		oneRouter.resolumeClient = osc.NewClient("127.0.0.1", 7000)
		oneRouter.guiClient = osc.NewClient("127.0.0.1", 3943)
		oneRouter.plogueClient = osc.NewClient("127.0.0.1", 3210)

		oneRouter.reactors["A"] = NewReactor("A", 1, freeframeClientA, oneRouter.resolumeClient, oneRouter.guiClient)
		oneRouter.reactors["B"] = NewReactor("B", 2, freeframeClientB, oneRouter.resolumeClient, oneRouter.guiClient)
		oneRouter.reactors["C"] = NewReactor("C", 3, freeframeClientC, oneRouter.resolumeClient, oneRouter.guiClient)
		oneRouter.reactors["D"] = NewReactor("D", 4, freeframeClientD, oneRouter.resolumeClient, oneRouter.guiClient)
		oneRouter.OSCInput = make(chan OSCEvent)
		oneRouter.MIDIInput = make(chan portmidi.Event)
		oneRouter.recordingOn = false
		oneRouter.MIDIOctaveShift = 0

		oneRouter.paletteHost = ConfigValue("palettehost")

		oneRouter.myHostname = ConfigValue("hostname")
		if oneRouter.myHostname == "" {
			hostname, err := os.Hostname()
			if err != nil {
				log.Printf("os.Hostname: err=%s\n", err)
				hostname = "unknown"
			}
			oneRouter.myHostname = hostname
		}
		if oneRouter.paletteHost == oneRouter.myHostname {
			oneRouter.isPaletteHost = true
		} else {
			oneRouter.isPaletteHost = false
		}

		oneRouter.publishCursor = ConfigBool("publishcursor")
		oneRouter.generateVisuals = ConfigBool("generatevisuals")
		oneRouter.generateSound = ConfigBool("generatesound")

		// oneRouter.MIDISetScale = false
		oneRouter.MIDIThru = ""
		oneRouter.MIDIThruScadjust = false
		oneRouter.UseExternalScale = false
		oneRouter.MIDIQuantized = false

		go oneRouter.notifyGUI("restart")
	})
	return &oneRouter
}

// StartOSC xxx
func StartOSC(source string) {

	r := TheRouter()
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
func StartNATSClient() {

	StartVizNats()

	// Hand all NATS messages to HandleAPI
	router := TheRouter()

	localapi := fmt.Sprintf("palette.%s.api", router.myHostname)

	log.Printf("StartNATS: subscribing to %s\n", localapi)

	TheVizNats.Subscribe(localapi, func(msg *nats.Msg) {
		data := string(msg.Data)
		response := router.HandleAPI(router.ExecuteAPILocal, data)
		msg.Respond([]byte(response))
	})

	if router.isPaletteHost {
		remoteapi := fmt.Sprintf("palette.%s.api", router.paletteHost)
		log.Printf("StartNATS: subscribing to %s\n", remoteapi)
		TheVizNats.Subscribe(remoteapi, func(msg *nats.Msg) {
			data := string(msg.Data)
			response := router.HandleAPI(router.ExecuteAPIHost, data)
			msg.Respond([]byte(response))
		})
	}

	if ConfigBoolWithDefault("subscribecursor", false) {
		log.Printf("StartNATS: subscribing to %s\n", SubscribeCursorSubject)
		TheVizNats.Subscribe(SubscribeCursorSubject, func(msg *nats.Msg) {
			data := string(msg.Data)
			router.handleSubscribedCursorInput(data)
		})
	}

	if ConfigBoolWithDefault("subscribemidi", false) {
		if ConfigBool("subscribemidi") {
			log.Printf("StartNATS: subscribing to %s\n", SubscribeMIDISubject)
			TheVizNats.Subscribe(SubscribeMIDISubject, func(msg *nats.Msg) {
				data := string(msg.Data)
				router.HandleMIDIEventMsg(data)
			})
		}
	}
}

// TimeString returns time and clicks
func TimeString() string {
	r := TheRouter()
	sofar := r.time.Sub(r.time0)
	click := Seconds2Clicks(sofar.Seconds())
	return fmt.Sprintf("sofar=%f click=%d", sofar.Seconds(), click)

}

// StartRealtime runs the looper and never returns
func StartRealtime() {

	log.Println("StartRealtime begins")

	r := TheRouter()

	// Wake up every 2 milliseconds and check looper events
	tick := time.NewTicker(2 * time.Millisecond)
	r.time0 = <-tick.C

	var lastPrintedClick Clicks

	// By reading from tick.C, we wake up every 2 milliseconds
	for now := range tick.C {
		// log.Printf("Realtime loop now=%v\n", time.Now())
		r.time = now
		sofar := now.Sub(r.time0)
		secs := sofar.Seconds()
		newclick := Seconds2Clicks(secs)
		CurrentMilli = int(secs * 1000.0)

		if newclick > currentClick {
			// log.Printf("ADVANCING CLICK now=%v click=%d\n", time.Now(), newclick)
			r.advanceClickTo(currentClick)
			currentClick = newclick
		}

		var everySoOften = oneBeat * 4
		if (currentClick%everySoOften) == 0 && currentClick != lastPrintedClick {
			if debug {
				log.Printf("currentClick=%d  unix=%d:%d\n", currentClick, time.Now().Unix(), time.Now().UnixNano())
			}
			lastPrintedClick = currentClick
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

// StartMIDI listens for MIDI events and sends them to the MIDIInput chan
func StartMIDI() {
	r := TheRouter()
	inputName := ConfigValue("midiinput")
	if inputName == "" {
		return
	}
	input := MIDI.getInput(inputName)
	if input == nil {
		log.Printf("There is no MIDI input named %s\n", inputName)
		return
	}
	r.MIDINumDown = 0
	log.Printf("Successfully opened MIDI input device %s\n", inputName)
	for {
		hasinput, err := input.Poll()
		if err != nil {
			log.Printf("ERROR in Poll?  err=%v\n", err)
		} else if hasinput {
			event, err := input.ReadEvent()
			if err != nil {
				log.Printf("ERROR in Read?  err = %v\n", err)
				// we may have lost some NOTEOFF's so reset our count
				r.MIDINumDown = 0
			} else {
				// log.Printf("MIDI input ReadEvent = %+v\n", event)
				r.MIDIInput <- event
			}
		} else {
			// log.Printf("No input time=%v\n", time.Now())
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// IngestRealtimeCommand sends a RealtimeCommand to the Realtime
func IngestRealtimeCommand(cmd Command) {
	TheRouter().control <- cmd

}

func (r *Router) handleDeviceCursorInput(e CursorDeviceEvent) {
	r.routeCursorDeviceEvent(e)
}

func (r *Router) handleSubscribedCursorInput(data string) {

	args, err := StringMap(data)
	if err != nil {
		log.Printf("HandleSubscribedCursor: err=%s\n", err)
		return
	}

	log.Printf("HandleSubscribedCursor: data=%s\n", data)

	api := SubscribeCursorSubject

	cid, err := needStringArg("cid", api, args)
	if err != nil {
		log.Printf("HandleSubscribedCursor: err=%s\n", err)
		return
	}

	words := strings.SplitN(cid, ".", 2)
	if len(words) != 2 {
		log.Printf("HandleSubscribedCursor: wrong format cid=%s\n", cid)
		return
	}
	source := words[0]
	if source == MyNUID() {
		log.Printf("Ignoring cursorevent from myself, source=%s\n", source)
		return
	}

	if DebugUtil.Cursor {
		log.Printf("Router.HandleSubscribedCursor: data=%s\n", data)
	}

	eventType, err := needStringArg("event", api, args)
	if err != nil {
		log.Printf("HandleSubscribedCursor: err=%s\n", err)
		return
	}
	switch eventType {
	case "down":
	case "drag":
	case "up":
	default:
		log.Printf("handleSubscribedCursorInput: Unexpected cursorevent type: %s\n", eventType)
		return
	}

	// region := optionalStringArg("region", args, "")

	x, err := needFloatArg("x", api, args)
	if err != nil {
		log.Printf("HandleSubscribedCursor: err=%s\n", err)
		return
	}

	y, err := needFloatArg("y", api, args)
	if err != nil {
		log.Printf("HandleSubscribedCursor: err=%s\n", err)
		return
	}

	z, err := needFloatArg("z", api, args)
	if err != nil {
		log.Printf("HandleSubscribedCursor: err=%s\n", err)
		return
	}

	ce := CursorDeviceEvent{
		Cid:        cid,
		Timestamp:  int64(CurrentMilli),
		DownDragUp: eventType,
		X:          x,
		Y:          y,
		Z:          z,
		Area:       0.0,
	}
	r.routeCursorDeviceEvent(ce)
	return
}

// HandleMIDIEventMsg xxx
func (r *Router) HandleMIDIEventMsg(data string) {

	args, err := StringMap(data)

	if err != nil {
		log.Printf("HandleMIDIEventMsg: err=%s\n", err)
		return
	}

	source := optionalStringArg("source", args, "unknown")
	if source == MyNUID() {
		// if DebugUtil.Cursor {
		// 	log.Printf("Ignoring event from myself, source=%s\n", source)
		// }
		return
	}

	if DebugUtil.MIDI {
		log.Printf("Router.HandleMIDIEventMsg: data=%s\n", data)
	}

	log.Printf("HandleMIDIEventMsg: NOT IMPLEMENTED YET\n")
	return
}

// HandleAPI xxx
func (r *Router) HandleAPI(executor Executor, data string) (response string) {

	smap, err := StringMap(data)

	defer func() {
		if DebugUtil.API {
			log.Printf("Router.HandleAPI: response=%s\n", response)
		}
	}()

	if err != nil {
		response = ErrorResponse(err)
		return
	}
	api, ok := smap["api"]
	if !ok {
		response = ErrorResponse(fmt.Errorf("Missing api parameter"))
		return
	}
	rawargs, ok := smap["params"]
	if !ok {
		response = ErrorResponse(fmt.Errorf("Missing params parameter"))
		return
	}
	if DebugUtil.API {
		log.Printf("Router.HandleAPI: api=%s args=%s\n", api, rawargs)
	}
	// result, err := r.ExecuteAPI(api, rawargs)
	result, err := executor(api, rawargs)
	if err != nil {
		response = ErrorResponse(err)
	} else {
		response = ResultResponse(result)
	}
	return
}

// ExecuteAPILocal xxx
func (r *Router) ExecuteAPILocal(api string, rawargs string) (result interface{}, err error) {

	args, err := StringMap(rawargs)
	if err != nil {
		response := ErrorResponse(fmt.Errorf("Router.ExecuteAPI: Unable to interpret value - %s", rawargs))
		log.Printf("Router.ExecuteAPI: bad rawargs value = %s\n", rawargs)
		return response, nil
	}

	result = "0" // most APIs just return 0, so pre-populate it

	words := strings.SplitN(api, ".", 2)
	if len(words) != 2 {
		return nil, fmt.Errorf("Router.ExecuteAPI: api=%s is badly formatted, needs a dot", api)
	}
	apiprefix := words[0]
	apisuffix := words[1]

	if apiprefix == "region" {
		reactor, err := needRegionArg(api, args)
		if err != nil {
			return nil, err
		}
		return reactor.ExecuteAPI(apisuffix, args, rawargs)
	}

	// Everything else should be "global", eventually I'll factor this
	if apiprefix != "global" {
		return nil, fmt.Errorf("ExecuteAPI: api=%s unknown apiprefix=%s", api, apiprefix)
	}

	switch apisuffix {

	case "echo":
		value, ok := args["value"]
		if !ok {
			value = "ECHO!"
		}
		result = value

	case "debug":
		s, err := needStringArg("debug", api, args)
		if err == nil {
			b, err := needBoolArg("onoff", api, args)
			if err == nil {
				setDebug(s, b)
			}
		}

	case "set_transpose":
		v, err := needIntArg("value", api, args)
		if err == nil {
			TransposePitch = v
		}

	case "midi_thru":
		v, err := needStringArg("thru", api, args)
		if err == nil {
			r.MIDIThru = v
		}

	case "midi_thruscadjust":
		v, err := needBoolArg("onoff", api, args)
		if err == nil {
			r.MIDIThruScadjust = v
		}

	case "useexternalscale":
		v, err := needBoolArg("onoff", api, args)
		if err == nil {
			r.UseExternalScale = v
		}

	case "clearexternalscale":
		// log.Printf("router is clearing external scale\n")
		ClearExternalScale()
		r.MIDINumDown = 0

	case "midi_quantized":
		v, err := needBoolArg("quantized", api, args)
		if err == nil {
			r.MIDIQuantized = v
		}

	case "set_tempo_factor":
		v, err := needFloatArg("value", api, args)
		if err == nil {
			ChangeClicksPerSecond(float64(v))
		}

	case "audioOn":
		msg := osc.NewMessage("/play")
		msg.Append(int32(1))
		r.plogueClient.Send(msg)
		if DebugUtil.OSC {
			log.Printf("Router.ExecuteAPI: audioOn sending msg=%v\n", msg)
		}

	case "audioOff":
		log.Printf("Router.ExecuteAPI: audioOff is now ignored\n")

	case "audioStop":
		// In case we really need it
		msg := osc.NewMessage("/play")
		msg.Append(int32(0))
		r.plogueClient.Send(msg)
		if DebugUtil.OSC {
			log.Printf("Router.ExecuteAPI: audioStop sending msg=%v\n", msg)
		}

	case "recordingStart":
		r.recordingOn = true
		if r.recordingFile != nil {
			log.Printf("Hey, recordingFile wasn't nil?\n")
			r.recordingFile.Close()
		}
		r.recordingFile, err = os.Create(recordingsFile("LastRecording.json"))
		if err != nil {
			return nil, err
		}
		r.recordingBegun = time.Now()
		if r.recordingOn {
			r.recordEvent("global", "*", "start", "{}")
		}
	case "recordingSave":
		name, err := needStringArg("name", api, args)
		if err == nil {
			err = r.recordingSave(name)
		}

	case "recordingStop":
		if r.recordingOn {
			r.recordEvent("global", "*", "stop", "{}")
		}
		if r.recordingFile != nil {
			r.recordingFile.Close()
			r.recordingFile = nil
		}
		r.recordingOn = false

	case "recordingPlay":
		name, err := needStringArg("name", api, args)
		if err == nil {
			events, err := r.recordingLoad(name)
			if err == nil {
				r.sendANO()
				go r.recordingPlayback(events)
			}
		}

	case "recordingPlaybackStop":
		r.recordingPlaybackStop()

	default:
		log.Printf("Router.ExecuteAPI api=%s is not recognized\n", api)
		err = fmt.Errorf("Router.ExecuteAPI unrecognized api=%s", api)
		result = ""
	}

	return result, err
}

// ExecuteAPIHost xxx
func (r *Router) ExecuteAPIHost(api string, rawargs string) (result interface{}, err error) {

	args, err := StringMap(rawargs)
	if err != nil {
		response := ErrorResponse(fmt.Errorf("Router.ExecuteAPI: Unable to interpret value - %s", rawargs))
		log.Printf("Router.ExecuteAPI: bad rawargs value = %s\n", rawargs)
		return response, nil
	}

	result = "0" // most APIs just return 0, so pre-populate it

	words := strings.SplitN(api, ".", 2)
	if len(words) != 2 {
		return nil, fmt.Errorf("Router.ExecuteAPI: api=%s is badly formatted, needs a dot", api)
	}
	apiprefix := words[0]
	apisuffix := words[1]

	if apiprefix == "region" {
		reactor, err := needRegionArg(api, args)
		if err != nil {
			return nil, err
		}
		return reactor.ExecuteAPI(apisuffix, args, rawargs)
	}

	// Everything else should be "global", eventually I'll factor this
	if apiprefix != "global" {
		return nil, fmt.Errorf("ExecuteAPI: api=%s unknown apiprefix=%s", api, apiprefix)
	}

	switch apisuffix {

	case "echo":
		value, ok := args["value"]
		if !ok {
			value = "ECHO!"
		}
		result = value

	case "debug":
		s, err := needStringArg("debug", api, args)
		if err == nil {
			b, err := needBoolArg("onoff", api, args)
			if err == nil {
				setDebug(s, b)
			}
		}

	case "set_transpose":
		v, err := needIntArg("value", api, args)
		if err == nil {
			TransposePitch = v
		}

	case "midi_thru":
		v, err := needStringArg("thru", api, args)
		if err == nil {
			r.MIDIThru = v
		}

	case "midi_thruscadjust":
		v, err := needBoolArg("onoff", api, args)
		if err == nil {
			r.MIDIThruScadjust = v
		}

	case "useexternalscale":
		v, err := needBoolArg("onoff", api, args)
		if err == nil {
			r.UseExternalScale = v
		}

	case "clearexternalscale":
		// log.Printf("router is clearing external scale\n")
		ClearExternalScale()
		r.MIDINumDown = 0

	case "midi_quantized":
		v, err := needBoolArg("quantized", api, args)
		if err == nil {
			r.MIDIQuantized = v
		}

	case "set_tempo_factor":
		v, err := needFloatArg("value", api, args)
		if err == nil {
			ChangeClicksPerSecond(float64(v))
		}

	case "audioOn":
		msg := osc.NewMessage("/play")
		msg.Append(int32(1))
		r.plogueClient.Send(msg)
		if DebugUtil.OSC {
			log.Printf("plogClient %s msg=%v\n", TimeString(), msg)
		}

	case "audioOff":
		msg := osc.NewMessage("/play")
		msg.Append(int32(0))
		r.plogueClient.Send(msg)
		if DebugUtil.OSC {
			log.Printf("plogClient %s msg=%v\n", TimeString(), msg)
		}

	case "recordingStart":
		r.recordingOn = true
		if r.recordingFile != nil {
			log.Printf("Hey, recordingFile wasn't nil?\n")
			r.recordingFile.Close()
		}
		r.recordingFile, err = os.Create(recordingsFile("LastRecording.json"))
		if err != nil {
			return nil, err
		}
		r.recordingBegun = time.Now()
		if r.recordingOn {
			r.recordEvent("global", "*", "start", "{}")
		}
	case "recordingSave":
		name, err := needStringArg("name", api, args)
		if err == nil {
			err = r.recordingSave(name)
		}

	case "recordingStop":
		if r.recordingOn {
			r.recordEvent("global", "*", "stop", "{}")
		}
		if r.recordingFile != nil {
			r.recordingFile.Close()
			r.recordingFile = nil
		}
		r.recordingOn = false

	case "recordingPlay":
		name, err := needStringArg("name", api, args)
		if err == nil {
			events, err := r.recordingLoad(name)
			if err == nil {
				r.sendANO()
				go r.recordingPlayback(events)
			}
		}

	case "recordingPlaybackStop":
		r.recordingPlaybackStop()

	default:
		log.Printf("Router.ExecuteAPI api=%s is not recognized\n", api)
		err = fmt.Errorf("Router.ExecuteAPI unrecognized api=%s", api)
		result = ""
	}

	return result, err
}

func (r *Router) advanceClickTo(toClick Clicks) {
	// log.Printf("advanceClickTo r.lastClick=%d toClick=%d\n", r.lastClick, toClick)
	for clk := r.lastClick; clk < toClick; clk++ {
		for _, reactor := range r.reactors {
			if (clk % oneBeat) == 0 {
				reactor.checkCursorUp()
			}
			reactor.advanceClickBy1()
		}
	}
	r.lastClick = toClick
}

func (r *Router) recordEvent(eventType string, pad string, method string, args string) {
	if r.recordingOn == false {
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
	for _, reactor := range r.reactors {
		reactor.sendANO()
	}
}

func (r *Router) notifyGUI(eventName string) {
	msg := osc.NewMessage("/notify")
	msg.Append(eventName)
	r.guiClient.Send(msg)
	if DebugUtil.OSC {
		log.Printf("toGUI %s msg=%v\n", TimeString(), msg)
	}
}

func (r *Router) recordingPlayback(events []*PlaybackEvent) error {

	log.Printf("recordingPlay, #events = %d\n", len(events))
	r.killPlayback = false

	r.notifyGUI("start")

	playbackBegun := r.Time()
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
			var reactor *Reactor
			if pad != "*" {
				reactor = r.reactors[pad]
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
				reactor.executeIncomingCursor(CursorStepEvent{
					ID:         id,
					X:          xf,
					Y:          yf,
					Z:          zf,
					Downdragup: ddu,
				})
			case "api":
				// since we already have args
				reactor.ExecuteAPI(method, args, rawargs)
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

// ListenForever listens for things on its input channels
func ListenForever() {
	r := TheRouter()
	for r.killme == false {
		select {
		case msg := <-r.OSCInput:
			// log.Printf("received OSCInput - now=%v\n", time.Now())
			r.handleOSC(msg)
		case event := <-r.MIDIInput:
			// log.Printf("received MIDIInput - now=%v\n", time.Now())
			r.handleMIDI(event)
		default:
			// log.Printf("Sleeping 1 ms - now=%v\n", time.Now())
			time.Sleep(time.Millisecond)
		}
	}
	log.Printf("Router is being killed\n")
}

//		case bool, int32, int64, float32, float64, string:
func (r *Router) handleOSC(e OSCEvent) {
	// log.Printf("handleOSC time=%v\n", r.time)
	if e.Msg.Address == "/api" {
		r.handleOSCAPI(e.Msg, e.Source)
	} else if e.Msg.Address == "/quit" {
		log.Printf("Router received QUIT message!\n")
		r.killme = true
	} else {
		log.Printf("Unrecognized OSC message %v from %v\n", e.Msg, e.Source)
	}
}

func (r *Router) handleMIDISetScale(e portmidi.Event) {
	status := e.Status & 0xf0
	pitch := int(e.Data1)
	// log.Printf("handleMIDIScale numdown=%d e=%s\n", r.MIDINumDown, e)
	if status == 0x90 {
		// If there are no notes held down (i.e. this is the first), clear the scale
		if r.MIDINumDown < 0 {
			// this can happen when there's a Read error that misses a NOTEON
			r.MIDINumDown = 0
		}
		if r.MIDINumDown == 0 {
			ClearExternalScale()
		}
		SetExternalScale(pitch%12, true)
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
	// log.Printf("handleMIDIScale end numdown=%d\n", r.MIDINumDown)
}

func (r *Router) handleMIDI(e portmidi.Event) {

	// log.Printf("handleMIDI e=%s\n", e)
	switch r.MIDIThru {
	case "":
		// do nothing
	case "setscale":
		r.handleMIDISetScale(e)
	case "A", "B", "C", "D":
		reactor := r.reactors[r.MIDIThru]
		reactor.PassThruMIDI(e, r.MIDIThruScadjust, r.UseExternalScale)
	}
}

// No error return because it's OSC
func (r *Router) handleOSCAPI(msg *osc.Message, source string) {
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
	if DebugUtil.API {
		log.Printf("Router.handleOSCAPI api=%s rawargs=%s\n", api, rawargs)
	}
	_, err = r.ExecuteAPILocal(api, rawargs)
	if err != nil {
		log.Printf("Router.handleOSCAPI: err=%s", err)
	}
	return
}

// availableRegion - return the name of a region that hasn't been assigned to a remote yet
func (r *Router) assignRegion(source string) string {

	regionLetters := "ABCD"

	nregions := len(regionLetters)
	regionAssigned := make([]bool, nregions)
	for _, v := range r.regionAssignedToSource {
		i := strings.Index(regionLetters, v)
		if i >= 0 {
			regionAssigned[i] = true
		}
	}
	for i, used := range regionAssigned {
		if !used {
			avail := regionLetters[i : i+1]
			log.Printf("Router.availableRegion: %s\n", avail)
			r.regionAssignedToSource[source] = avail
			return avail
		}
	}
	log.Printf("Router.availableRegion: No regions available\n")
	return ""
}

func (r *Router) regionForCursor(e CursorDeviceEvent) string {
	words := strings.SplitN(e.Cid, ".", 2)
	if len(words) != 2 {
		log.Printf("Router.regionForCursore: invalid e.Cid=%s\n", e.Cid)
		return "A"
	}
	source := words[0]

	// See if we've already assigned a region to this source
	reg, ok := r.regionForSource[source]
	if ok {
		return reg
	}

	// Hasn't been seen yet, let's get an available region
	reg = r.assignRegion(source)
	r.setRegionForSource(source, reg)

	return reg
}

func (r *Router) setRegionForSource(source string, region string) {
	r.regionForSource[source] = region
}

func (r *Router) routeCursorDeviceEvent(e CursorDeviceEvent) {
	region := r.regionForCursor(e)

	reactor, ok := r.reactors[region]
	if !ok {
		log.Printf("routeCursorDeviceEvent: no region named %s, unable to process ce=%+v\n", region, e)
		return
	}
	reactor.handleCursorDeviceEvent(e)
	if r.publishCursor {
		err := PublishCursorDeviceEvent(e)
		if err != nil {
			log.Printf("Router.routeCursorDeviceEvent: NATS publishing err=%s\n", err)
		}
	}
}

// Time returns the current time
func (r *Router) Time() time.Time {
	return time.Now()
}

func argAsInt(msg *osc.Message, index int) (i int, err error) {
	arg := msg.Arguments[index]
	switch arg.(type) {
	case int32:
		i = int(arg.(int32))
	case int64:
		i = int(arg.(int64))
	default:
		err = fmt.Errorf("Expected an int in OSC argument index=%d", index)
	}
	return i, err
}

func argAsFloat32(msg *osc.Message, index int) (f float32, err error) {
	arg := msg.Arguments[index]
	switch arg.(type) {
	case float32:
		f = arg.(float32)
	case float64:
		f = float32(arg.(float64))
	default:
		err = fmt.Errorf("Expected a float in OSC argument index=%d", index)
	}
	return f, err
}

func argAsString(msg *osc.Message, index int) (s string, err error) {
	arg := msg.Arguments[index]
	switch arg.(type) {
	case string:
		s = arg.(string)
	default:
		err = fmt.Errorf("Expected a string in OSC argument index=%d", index)
	}
	return s, err
}

func cursorid(sid int, source string) string {
	return fmt.Sprintf("%d@%s", sid, source)
}
