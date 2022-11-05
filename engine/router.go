package engine

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hypebeast/go-osc/osc"
)

// Router takes events and routes them
type Router struct {
	playerLetters     string
	players           map[string]*Player
	OSCInput          chan OSCEvent
	MIDIInput         chan MidiEvent
	NoteInput         chan *Note
	CursorDeviceInput chan CursorDeviceEvent

	AliveWaiters map[string]chan string

	layerMap map[string]int // map of pads to resolume layer numbers

	midiEventHandler MIDIEventHandler
	// lastClick        Clicks
	// control chan Command
	// time             time.Time
	// time0            time.Time
	// killPlayback bool
	// recordingOn    bool
	// recordingFile  *os.File
	// recordingBegun time.Time

	resolumeClient *osc.Client
	guiClient      *osc.Client
	plogueClient   *osc.Client

	myHostname           string
	generateVisuals      bool
	generateSound        bool
	playerAssignedToNUID map[string]string
	eventMutex           sync.RWMutex

	// responders map[string]Responder
}

// OSCEvent is an OSC message
type OSCEvent struct {
	Msg    *osc.Message
	Source string
}

type HeartbeatEvent struct {
	UpTime      time.Duration
	CursorCount int
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

// APIExecutorFunc xxx
type APIExecutorFunc func(api string, nuid string, rawargs string) (result interface{}, err error)

// CallAPI calls an API in the Palette Freeframe plugin running in Resolume
func CallAPI(method string, args []string) {
	log.Printf("CallAPI method=%s", method)
}

func NewRouter() *Router {

	r := Router{}

	// r.deviceCursors = make(map[string]*DeviceCursor)
	// r.activeCursors = make(map[string]*ActiveStepCursor)
	// r.responders = make(map[string]Responder)

	r.playerLetters = ConfigValue("pads")
	if r.playerLetters == "" {
		if Debug.Morph {
			log.Printf("No value for pads, assuming ABCD")
		}
		r.playerLetters = "ABCD"
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

	r.players = make(map[string]*Player)
	r.playerAssignedToNUID = make(map[string]string)

	resolumePort := 7000
	guiPort := 3943
	ploguePort := 3210

	r.resolumeClient = osc.NewClient("127.0.0.1", resolumePort)
	r.guiClient = osc.NewClient("127.0.0.1", guiPort)
	r.plogueClient = osc.NewClient("127.0.0.1", ploguePort)

	log.Printf("OSC client ports: resolume=%d gui=%d plogue=%d\n", resolumePort, guiPort, ploguePort)

	for i, c := range r.playerLetters {
		resolumeLayer := r.ResolumeLayerForPad(string(c))
		ffglPort := ResolumePort + i
		ch := string(c)
		freeframeClient := osc.NewClient("127.0.0.1", ffglPort)
		log.Printf("OSC freeframeClient: port=%d layer=%d\n", ffglPort, resolumeLayer)
		r.players[ch] = NewPlayer(ch, resolumeLayer, freeframeClient, r.resolumeClient, r.guiClient)
		// log.Printf("Pad %s created, resolumeLayer=%d resolumePort=%d\n", ch, resolumeLayer, resolumePort)
	}

	r.OSCInput = make(chan OSCEvent)
	r.MIDIInput = make(chan MidiEvent)
	r.NoteInput = make(chan *Note)
	// r.recordingOn = false

	r.myHostname = ConfigValue("hostname")
	if r.myHostname == "" {
		hostname, err := os.Hostname()
		if err != nil {
			log.Printf("os.Hostname: err=%s\n", err)
			hostname = "unknown"
		}
		r.myHostname = hostname
	}

	// By default, the engine handles Cursor events internally.
	// However, if publishcursor is set, it ONLY publishes them,
	// so the expectation is that a plugin will handle it.
	r.generateVisuals = ConfigBoolWithDefault("generatevisuals", true)
	r.generateSound = ConfigBoolWithDefault("generatesound", true)

	for _, player := range r.players {
		player.restoreCurrentSnap()
	}

	go r.notifyGUI("restart")
	return &r
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
		playerLayers := "1,2,3,4"
		s := ConfigStringWithDefault("playerLayers", playerLayers)
		layers := strings.Split(s, ",")
		if len(layers) != 4 {
			log.Printf("ResolumeLayerForPad: playerLayers value needs 4 values\n")
			layers = strings.Split(playerLayers, ",")
		}
		r.layerMap = make(map[string]int)
		r.layerMap["A"], _ = strconv.Atoi(layers[0])
		r.layerMap["B"], _ = strconv.Atoi(layers[1])
		r.layerMap["C"], _ = strconv.Atoi(layers[2])
		r.layerMap["D"], _ = strconv.Atoi(layers[3])

		// log.Printf("Router: Resolume layerMap = %+v\n", r.layerMap)
	}
	return r.layerMap[pad]
}

type MIDIEventHandler func(MidiEvent)

func (r *Router) SetMIDIEventHandler(handler MIDIEventHandler) {
	r.midiEventHandler = handler
}

func (r *Router) handleMidiInput(event MidiEvent) {
	log.Printf("Router.handleMidiInput is disabled\n")
	/*
		for _, player := range r.players {
			player.HandleMidiInput(event)
		}
	*/
}

func ArgToFloat(nm string, args map[string]string) float32 {
	f, err := strconv.ParseFloat(args[nm], 32)
	if err != nil {
		log.Printf("Unable to parse %s value (%s), assuming 0.0\n", nm, args[nm])
		f = 0.0
	}
	return float32(f)
}

func ArgsToCursorDeviceEvent(args map[string]string) CursorDeviceEvent {
	id := args["id"]
	source := args["source"]
	event := strings.TrimPrefix(args["event"], "cursor_")
	x := ArgToFloat("x", args)
	y := ArgToFloat("y", args)
	z := ArgToFloat("z", args)
	ce := CursorDeviceEvent{
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
func (r *Router) HandleInputEvent(args map[string]string) error {

	r.eventMutex.Lock()
	defer r.eventMutex.Unlock()

	event, err := needStringArg("event", "HandleEvent", args)
	if err != nil {
		return err
	}

	// If no "player" argument, assign one, though it should eventually be required
	playerName, playerok := args["player"]
	if !playerok {
		return fmt.Errorf("HandleInputEvent: No player value")
	}

	if Debug.Router {
		log.Printf("Router.HandleEvent: player=%s event=%s\n", playerName, event)
	}

	// XXX - player value should allow "*" and other multi-player values
	player, ok := r.players[playerName]
	if !ok {
		return fmt.Errorf("there is no player named %s", playerName)
	}

	switch event {

	case "engine":
		log.Printf("Router: ignoring engine event\n")
		return nil

	case "cursor_down", "cursor_drag", "cursor_up":
		ce := ArgsToCursorDeviceEvent(args)
		TheEngine.handleCursorDeviceEvent(ce)

	case "sprite":

		x, y, z, err := GetArgsXYZ(args)
		if err != nil {
			return nil
		}
		player.generateSprite("dummy", x, y, z)

	case "midi_reset":
		log.Printf("HandleEvent: midi_reset, sending ANO\n")
		player.HandleMIDITimeReset()
		player.sendANO()

	case "audio_reset":
		log.Printf("HandleEvent: audio_reset!!\n")
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
		log.Printf("HandleInputEvent: notestr=%s clickstr=%s\n", notestr, clickstr)
		note, err := NoteFromString(notestr)
		if err != nil {
			return err
		}
		if note.Synth == "" {
			note.Synth = "default"
		}
		SendNoteToSynth(note)

	default:
		log.Printf("HandleInputEvent: event not handled: %s\n", event)
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

	// r.eventMutex.Lock()
	// defer r.eventMutex.Unlock()

	if Debug.OSC {
		log.Printf("Router.HandleOSCInput: msg=%s\n", e.Msg.String())
	}
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

/*
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
			var player *Player
			if pad != "*" {
				player = r.players[pad]
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
				player.executeIncomingCursor(CursorStepEvent{
					ID:  id,
					X:   xf,
					Y:   yf,
					Z:   zf,
					Ddu: ddu,
				})
			case "api":
				// since we already have args
				player.ExecuteAPI(method, args, rawargs)
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

func (r *Router) sendANO() {
	for _, player := range r.players {
		player.sendANO()
	}
}
*/

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

func (r *Router) handleMMTTButton(butt string) {
	presetName := ConfigStringWithDefault(butt, "")
	if presetName == "" {
		log.Printf("No Preset assigned to BUTTON %s, using Perky_Shapes\n", butt)
		presetName = "Perky_Shapes"
		return
	}
	preset := GetPreset("quad." + presetName)
	log.Printf("Router.handleMMTTButton: butt=%s preset=%s\n", butt, presetName)
	err := preset.loadQuadPreset("*")
	if err != nil {
		log.Printf("handleMMTTButton: preset=%s err=%s\n", presetName, err)
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
	var found *Player
	for _, player := range r.players {
		if player.freeframeClient.Port() == portnum {
			found = player
			break
		}
	}
	if found == nil {
		log.Printf("handleClientRestart unable to find Player with portnum=%d\n", portnum)
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
	player := "A"
	words := strings.Split(cid, ".")
	if len(words) > 1 {
		player = words[0]
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

	_, mok := r.players[player]
	if !mok {
		// If it's not a player, it's a button.
		buttonDepth := ConfigFloatWithDefault("mmttbuttondepth", 0.002)
		if z > buttonDepth {
			log.Printf("NOT triggering button too deep z=%f buttonDepth=%f\n", z, buttonDepth)
			return
		}
		if ddu == "down" {
			if Debug.MMTT {
				log.Printf("MMT BUTTON TRIGGERED buttonDepth=%f  z=%f\n", buttonDepth, z)
			}
			r.handleMMTTButton(player)
		}
		return
	}

	ce := CursorDeviceEvent{
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

	if Debug.MMTT && Debug.Cursor {
		log.Printf("MMTT Cursor %s %s xyz= %f %f %f\n", ce.Source, ce.Ddu, ce.X, ce.Y, ce.Z)
	}

	TheEngine.CursorManager.handleCursorDeviceEvent(ce, true)
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
	r.handleInputEventRaw(rawargs)
}

func (r *Router) handleInputEventRaw(rawargs string) {

	args, err := StringMap(rawargs)
	if err != nil {
		return
	}
	err = r.HandleInputEvent(args)
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
	if nargs < 3 {
		log.Printf("handlePatchXREvent: not enough arguments\n")
		return
	}
	action, _ := argAsString(msg, 0)
	playerName, _ := argAsString(msg, 1)
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
	s := fmt.Sprintf("\"event\": \"cursor_drag\", \"player\": \"%s\", \"x\": \"%f\", \"y\": \"%f\", \"z\": \"%f\"", playerName, x, y, z)
	var args map[string]string
	args, err = StringMap(s)
	if err != nil {
		return
	}

	err = r.HandleInputEvent(args)
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

	args, err := StringMap(rawargs)
	if err != nil {
		log.Printf("Router.handleOSCSpriteEvent: Unable to process args=%s\n", rawargs)
		return
	}

	err = r.HandleInputEvent(args)
	if err != nil {
		log.Printf("Router.handleOSCSpriteEvent: err=%s\n", err)
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

// This silliness is to avoid unused function errors from go-staticcheck
var rr *Router

// var _ = rr.recordPadAPI
var _ = rr.handleOSCSpriteEvent
var _ = argAsInt
var _ = argAsFloat32
