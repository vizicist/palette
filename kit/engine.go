package kit

import (
	"bufio"
	json "github.com/goccy/go-json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hypebeast/go-osc/osc"
)

type Engine struct {
	done           chan bool
	oscoutput      bool
	oscClient      *osc.Client
	recordingIndex int
	recordingFile  *os.File
	recordingPath  string
	recordingMutex sync.RWMutex
	// params         *ParamValues

	currentPitchOffset  *atomic.Int32
	autoTransposeOn     bool
	autoTransposeNext   Clicks
	autoTransposeClicks Clicks // time between auto transpose changes
	autoTransposeIndex  int    // current place in transposeValues
	autoTransposeValues []int
}

var TheEngine *Engine
var engineSysex sync.Mutex

func InitEngine() {

	InitializeClicks()

	if TheEngine != nil {
		LogIfError(fmt.Errorf("TheEngine already exists, there can be only one"))
		return
	}

	engineSysex.Lock()
	defer engineSysex.Unlock()

	e := &Engine{
		done:                make(chan bool),
		currentPitchOffset:  &atomic.Int32{},
		autoTransposeOn:     false,
		autoTransposeNext:   0,
		autoTransposeClicks: 8 * OneBeat,
		autoTransposeIndex:  0,
		autoTransposeValues: []int{0, -2, 3, -5},
	}
	e.currentPitchOffset.Store(0) // probably not needed

	// e.params = NewParamValues()

	err := NatsStartLeafServer()
	LogIfError(err)

	NatsConnectLocalAndSubscribe()

	TheCursorManager = NewCursorManager()
	TheRouter = NewRouter()
	TheScheduler = NewScheduler()
	TheAttractManager = NewAttractManager()
	TheQuad = NewQuad()
	TheMidiIO = NewMidiIO()
	TheErae = NewErae()

	InitLogTypes()

	TheEngine = e

	for name := range ParamDefs {
		if strings.HasPrefix(name, "global.") {
			// LogInfo("global name","name",name)
			ActivateGlobalParam(name)
		}
	}

	LogInfo("PaletteDataPath", "path", PaletteDataPath())

	/*
	// The _Boot.json file is now used to set the initial state of the system,
	// so if you want to force looping off at startup, you can do it there.
	LogInfo("Looping is being forced off")
	err = GlobalParams.SetParamWithString("global.looping_override", "true")
	LogIfError(err)
	err = GlobalParams.SetParamWithString("global.looping_fade", "0.00")
	LogIfError(err)
	err = SaveCurrentGlobalParams()
	LogIfError(err)
	*/
}

func LocalEngineApi(api string, args ...string) (map[string]string, error) {
	return EngineHttpApi(LocalAddress, api, args...)
}

func EngineHttpApi(host string, api string, args ...string) (map[string]string, error) {

	if len(args)%2 != 0 {
		return nil, fmt.Errorf("RemoteApi: odd nnumber of args, should be even")
	}
	apijson := "\"api\": \"" + api + "\""
	for n := range args {
		if n%2 == 0 {
			apijson = apijson + ",\"" + args[n] + "\": \"" + args[n+1] + "\""
		}
	}
	url := fmt.Sprintf("http://%s:%d/api", host, EngineHttpPort)
	return HttpApiRaw(url, apijson)
}

func EngineNatsApi(host string, cmd string) (result string, err error) {
	if !natsIsConnected {
		return "", fmt.Errorf("EngineNatsAPI: NATS is not connected")
	}
	if natsConn == nil {
		return "", fmt.Errorf("NatsAPI: NatsConn is nil?")
	}
	timeout := 3 * time.Second
	subject := fmt.Sprintf("to_palette.%s.api", host)
	retdata, err := NatsRequest(subject, cmd, timeout)
	LogIfError(err)
	return retdata, err
}

func EngineCloseNats() {
	NatsDisconnect()
}

func StartEngine() {
	TheEngine.Start()
}

func WaitTillDone() {
	TheEngine.WaitTillDone()
}

func GetParam(name string) (string, error) {
	return GlobalParams.Get(name)
}

func GetParamWithDefault(name string, dflt string) string {
	val, err := GlobalParams.Get(name)
	if err != nil {
		val = dflt
	}
	return val
}

// func (e *Engine) Get(name string) string {
// 	return e.params.Get(name)
// }

func GetParamBool(nm string) (bool, error) {
	v, err := GlobalParams.Get(nm)
	if err != nil {
		return false, err
	} else {
		return IsTrueValue(v), nil
	}
}

func GetParamInt(nm string) (int, error) {
	s, err := GlobalParams.Get(nm)
	if err != nil {
		return 0, err
	}
	var val int
	nfound, err := fmt.Sscanf(s, "%d", &val)
	if err != nil || nfound == 0 {
		return 0, fmt.Errorf("bad format of integer parameter name=%s", nm)
	}
	return val, nil
}

func GetParamFloat(nm string) (float64, error) {
	s, err := GlobalParams.Get(nm)
	if err != nil {
		return 0.0, err
	}
	f, err := strconv.ParseFloat(s, 32)
	if err != nil {
		// XXX - should be setting f to "init" value
		return 0.0, fmt.Errorf("bad format of float parameter name=%s", nm)
	}
	return f, nil
}

func SaveCurrentGlobalParams() (err error) {
	return SaveGlobalParamsIn("_Current")
}

func SaveGlobalParamsIn(filename string) (err error) {
	return GlobalParams.Save("global", filename)
}

func LoadGlobalParamsFrom(filename string, activate bool) (err error) {
	paramsMap, err := LoadParamsMapOfCategory("global", filename)
	if err != nil {
		LogIfError(err)
		return err
	}
	GlobalParams.ApplyValuesFromMap("global", paramsMap, GlobalParams.SetParamWithString)
	if activate {
		for nm := range paramsMap {
			ActivateGlobalParam(nm)
		}
	}
	SaveCurrentGlobalParams()
	return nil
}

func (e *Engine) Start() {

	e.done = make(chan bool)

	LogInfo("Engine.Start")

	InitMidiIO()
	InitSynths()

	TheQuad.Start()

	go e.StartOscListener(OscPort)
	go e.StartHttp(EngineHttpPort)

	go TheScheduler.Start()
	go TheRouter.Start()
	go TheMidiIO.Start()

	go TheBidule().Reset()

	// TheAttractManager.setAttractMode(true)

	// if ParamBool("mmtt.depth") {
	// 	go DepthRunForever()
	// }
}

func (e *Engine) WaitTillDone() {
	<-e.done
}

func (e *Engine) SayDone() {
	e.done <- true
}

func (e *Engine) SendOsc(client *osc.Client, msg *osc.Message) {
	if client == nil {
		LogIfError(fmt.Errorf("global.SendOsc: client is nil"))
		return
	}

	LogOfType("osc", "Sending to OSC client", "client", client.IP(), "port", client.Port(), "msg", msg)
	err := client.Send(msg)
	LogIfError(err)
}

func (e *Engine) sendToOscClients(msg *osc.Message) {
	if e.oscoutput {
		if e.oscClient == nil {
			e.oscClient = osc.NewClient(LocalAddress, EventClientPort)
			// oscClient is guaranteed to be non-nil
		}
		e.SendOsc(e.oscClient, msg)
	}
}

func (e *Engine) StartOscListener(port int) {

	defer func() {
		if r := recover(); r != nil {
			// Print stack trace in the error messages
			stacktrace := string(debug.Stack())
			// First to stdout, then to log file
			fmt.Printf("PANIC: recover in StartOscListener called, r=%+v stack=%v", r, stacktrace)
			err := fmt.Errorf("PANIC: recover in StartOscListener has been called")
			LogError(err, "r", r, "stack", stacktrace)
		}
	}()

	source := fmt.Sprintf("%s:%d", LocalAddress, port)

	d := osc.NewStandardDispatcher()

	err := d.AddMsgHandler("*", func(msg *osc.Message) {
		TheRouter.oscInputChan <- OscEvent{Msg: msg, Source: source}
	})
	if err != nil {
		LogIfError(err)
	}

	server := &osc.Server{
		Addr:       source,
		Dispatcher: d,
	}
	err = server.ListenAndServe()
	if err != nil {
		LogIfError(err)
	}
}

// StartHttp xxx
func (e *Engine) StartHttp(port int) {

	defer func() {
		if r := recover(); r != nil {
			// Print stack trace in the error messages
			stacktrace := string(debug.Stack())
			// First to stdout, then to log file
			fmt.Printf("PANIC: recover in StartHttp called, r=%+v stack=%v", r, stacktrace)
			err := fmt.Errorf("PANIC: recover in StartHttp has been called")
			LogError(err, "r", r, "stack", stacktrace)
		}
	}()

	http.HandleFunc("/api", func(responseWriter http.ResponseWriter, req *http.Request) {

		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusOK)

		response := ""
		defer func() {
			_, err := responseWriter.Write([]byte(response))
			if err != nil {
				LogIfError(err)
			}
		}()

		switch req.Method {
		case "POST":
			body, err := io.ReadAll(req.Body)
			if err != nil {
				response = ErrorResponse(err)
			} else {
				bstr := string(body)
				_ = bstr
				resp, err := ExecuteApiFromJson(bstr)
				if err != nil {
					response = ErrorResponse(err)
				} else {
					response = ResultResponse(resp)
				}
			}
		default:
			response = ErrorResponse(fmt.Errorf("HTTP server unable to handle method=%s", req.Method))
		}
	})

	source := fmt.Sprintf("%s:%d", LocalAddress, port)
	err := http.ListenAndServe(source, nil)
	if err != nil {
		LogIfError(err)
	}
}

func (e *Engine) StartRecording() (string, error) {
	fpath, err := e.NewRecordingPath()
	if err != nil {
		return "", err
	}
	f, err := os.Create(fpath)
	if err != nil {
		return "", err
	}
	e.recordingFile = f
	e.recordingPath = fpath
	e.RecordStartEvent()
	LogInfo("startrecording", "fpath", fpath)
	return fpath, nil
}

func (e *Engine) StopRecording() (string, error) {
	if e.recordingFile == nil {
		return "", fmt.Errorf("Engine.StopRecording: not recording")
	}
	e.RecordStopEvent()
	LogInfo("stoprecording", "recordingPath", e.recordingPath)
	e.recordingFile.Close()
	e.recordingFile = nil
	e.recordingPath = ""
	return "", nil
}

func (e *Engine) StartPlayback(fname string) error {
	fpath := e.RecordingPath(fname)
	f, err := os.Open(fpath)
	if err != nil {
		return err
	}
	go e.doPlayback(f)
	return nil
}

func (e *Engine) doPlayback(f *os.File) {
	fileScanner := bufio.NewScanner(f)
	fileScanner.Split(bufio.ScanLines)
	LogInfo("doPlayback start")
	for fileScanner.Scan() {
		var rec RecordingEvent
		err := json.Unmarshal(fileScanner.Bytes(), &rec)
		if err != nil {
			LogIfError(err)
			continue
		}
		LogInfo("Playback", "rec", rec)
		switch rec.Event {
		case "cursor":
			ce := rec.Value.(CursorEvent)
			LogInfo("Playback", "cursor", ce)
			ScheduleAt(CurrentClick(), ce.Tag, ce)
		}
	}
	err := fileScanner.Err()
	if err != nil {
		LogIfError(err)
	}
	LogInfo("doPlayback ran out of input")
	f.Close()
}

func (e *Engine) RecordingPath(fname string) string {
	return filepath.Join(PaletteDataPath(), "recordings", fname)
}

func (e *Engine) NewRecordingPath() (string, error) {
	recdir := filepath.Join(PaletteDataPath(), "recordings")
	_, err := os.Stat(recdir)
	if err != nil {
		if os.IsNotExist(err) {
			// Try to create it
			LogInfo("NewRecordingPath: Creating", "recdir", recdir)
			err = os.MkdirAll(recdir, os.FileMode(0777))
		}
		if err != nil {
			return "", err
		}
	}
	for {
		fname := fmt.Sprintf("%03d.json", e.recordingIndex)
		fpath := filepath.Join(recdir, fname)
		if !FileExists(fpath) {
			return fpath, nil
		}
		e.recordingIndex++
	}
}

func (e *Engine) RecordStartEvent() {
	pe := PlaybackEvent{
		Click: CurrentClick(),
	}
	e.RecordPlaybackEvent(pe)
}

func (e *Engine) RecordStopEvent() {
	pe := PlaybackEvent{
		Click:     CurrentClick(),
		IsRunning: false,
	}
	e.RecordPlaybackEvent(pe)
}

// The following routines can make use of generics, I suspect

func (e *Engine) RecordPlaybackEvent(event PlaybackEvent) {
	if e.recordingFile == nil {
		return
	}
	bytes := []byte("{\"PlaybackEvent\":")
	morebytes, err := json.Marshal(event)
	if err != nil {
		LogIfError(err)
		return
	}
	bytes = append(bytes, morebytes...)
	bytes = append(bytes, '}', '\n')
	_, err = e.recordingFile.Write(bytes)
	LogIfError(err)
}

func (e *Engine) RecordMidiEvent(event *MidiEvent) {
	if e.recordingFile == nil {
		return
	}
	bytes := []byte("{\"MidiEvent\":")
	morebytes, err := json.Marshal(event)
	if err != nil {
		LogIfError(err)
		return
	}
	bytes = append(bytes, morebytes...)
	bytes = append(bytes, '}', '\n')
	_, err = e.recordingFile.Write(bytes)
	LogIfError(err)
}

func (e *Engine) RecordOscEvent(event *OscEvent) {
	if e.recordingFile == nil {
		return
	}
	re := RecordingEvent{
		Event: "osc",
		Value: event,
	}
	bytes, err := json.Marshal(re)
	if err != nil {
		LogIfError(err)
		return
	}
	bytes = append(bytes, '}', '\n')
	_, err = e.recordingFile.Write(bytes)
	LogIfError(err)
}

func (e *Engine) RecordCursorEvent(event CursorEvent) {
	if e.recordingFile == nil {
		return
	}

	re := RecordingEvent{
		Event: "cursor",
		Value: event,
	}
	bytes, err := json.Marshal(re)
	if err != nil {
		LogIfError(err)
		return
	}
	bytes = append(bytes, '\n')

	e.recordingMutex.Lock()

	_, err = e.recordingFile.Write(bytes)

	e.recordingMutex.Unlock()

	LogIfError(err)
}

/*
func (e *Engine) SaveRecordingEvent(re RecordingEvent) {
	bytes, err := json.Marshal(re)
	if err != nil {
		LogIfError(err)
		return
	}
	bytes = append(bytes, '\n')
	_, err = e.recordingFile.Write(bytes)
	LogIfError(err)
}
*/

func (e *Engine) SetTranspose(i int) {
	e.currentPitchOffset.Store(int32(i))
	LogOfType("transpose", "SetTranspose", "i", i)
	// LogInfo("Engine.SetTranspose", "schedule", TheScheduler.ToString())
	// LogInfo("Engine.SetTranspose", "pending", TheScheduler.PendingToString())
}

func (e *Engine) SetAutoTransposeBeats(beats int) {
	e.autoTransposeNext = Clicks(beats) * OneBeat
	e.autoTransposeClicks = Clicks(beats) * OneBeat
	LogOfType("transpose", "SetTransposeBeats", "beats", beats)
}

func (e *Engine) advanceTransposeTo(newclick Clicks) {

	if !e.autoTransposeOn {
		return
	}
	if newclick < e.autoTransposeNext {
		return
	}
	e.autoTransposeNext = newclick + e.autoTransposeClicks
	e.autoTransposeIndex = (e.autoTransposeIndex + 1) % len(e.autoTransposeValues)
	transpose := TheEngine.autoTransposeValues[TheEngine.autoTransposeIndex]
	e.SetTranspose(transpose)
	LogOfType("transpose", "TransposeTo", "index", e.autoTransposeIndex, "transpose", transpose, "newclick", newclick)
	// TheScheduler.SendAllPendingNoteoffs()
}

/*
	XXX - pitchOffset should be done elsewhere
	pitchOffset := TheMidiIO.engineTranspose

	// Hardcoded, channel 10 is usually drums, doesn't get transposed
	// Should probably be an attribute of the Synth.
	const drumChannel = 10
	if TheMidiIO.autoTransposeOn && synth.portchannel.channel != drumChannel {
		pitchOffset += TheMidiIO.autoTransposeValues[TheMidiIO.autoTransposeIndex]
	}
	newpitch := int(pitch) + pitchOffset
	if newpitch < 0 {
		newpitch = 0
	} else if newpitch > 127 {
		newpitch = 127
	}
	if newpitch != int(pitch) {
		LogOfType("midi", "SendNoteToMidiOutput adjusting pitch", "newpitch", newpitch, "oldpitch", pitch)
	}
	pitch = uint8(newpitch)
*/
