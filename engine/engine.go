package engine

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"

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
	params         *ParamValues
	loading bool

	currentPitchOffset  *atomic.Int32
	autoTransposeOn     bool
	autoTransposeNext   Clicks
	autoTransposeClicks Clicks // time between auto transpose changes
	autoTransposeIndex  int    // current place in transposeValues
	autoTransposeValues []int
}

var TheEngine *Engine
var engineSysex sync.Mutex
var TheRand *rand.Rand

func InitMisc() {
	InitParams()
	TheProcessManager = NewProcessManager()
	// Fixed rand sequence, better for testing
	TheRand = rand.New(rand.NewSource(1))
}

func InitEngine() {

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

	e.params = NewParamValues()

	TheCursorManager = NewCursorManager()
	TheRouter = NewRouter()
	TheScheduler = NewScheduler()
	TheAttractManager = NewAttractManager()
	TheQuadPro = NewQuadPro()
	TheMidiIO = NewMidiIO()
	TheErae = NewErae()

	InitLogTypes()

	// Set all the default engine.* values
	for nm, pd := range ParamDefs {
		if pd.Category == "engine" {
			err := e.params.setParamValueWithString(nm, pd.Init)
			if err != nil {
				LogIfError(err)
			}
		}
	}
	e.loading = true
	err := e.LoadCurrent()
	if err != nil {
		LogIfError(err)
	}
	e.loading = false

	TheEngine = e

	enabled, err := GetParamBool("engine.attractenabled")
	LogIfError(err)
	TheAttractManager.SetAttractEnabled(enabled)

	// This need to be done after engine parameters are loaded
	TheProcessManager.AddBuiltins()
}

func Start() {
	TheEngine.Start()
}

func WaitTillDone() {
	TheEngine.WaitTillDone()
}

func GetParam(name string) (string, error) {
	if TheEngine == nil {
		return "", fmt.Errorf("GetParam called before NewEngine, name=%s)", name)
	}
	return TheEngine.params.Get(name)
}

func GetParamWithDefault(name string, dflt string) string {
	if TheEngine == nil {
		return ""
	}
	val, err := TheEngine.params.Get(name)
	if err != nil {
		val = dflt
	}
	return val
}

// func (e *Engine) Get(name string) string {
// 	return e.params.Get(name)
// }

func GetParamBool(nm string) (bool, error) {
	v, err := TheEngine.params.Get(nm)
	if err != nil {
		return false, err
	} else {
		return IsTrueValue(v), nil
	}
}

func GetParamInt(nm string) (int, error) {
	s, err := TheEngine.params.Get(nm)
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
	s, err := TheEngine.params.Get(nm)
	if err != nil {
		return 0.0, err
	}
	f, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0.0, fmt.Errorf("bad format of float parameter name=%s", nm)
	}
	return f, nil
}

func (e *Engine) SaveCurrent() (err error) {
	return e.params.Save("engine", "_Current")
}

func (e *Engine) LoadCurrent() (err error) {
	paramsMap, err := LoadParamsMapOfCategory("engine","_Current")
	if err != nil {
		LogIfError(err)
		return err
	}
	return e.LoadEngineParams(paramsMap)
	/*
		path, err := ReadableSavedFilePath("engine", "_Current", ".json")
		if err != nil {
			return err
		}
		paramsmap, err := LoadParamsMap(path)
		if err != nil {
			return err
		}
		e.params.ApplyValuesFromMap("engine", paramsmap, e.Set)
		return nil
	*/
}

func (e *Engine) LoadEngineParams(paramsmap ParamsMap) (err error) {
	e.params.ApplyValuesFromMap("engine", paramsmap, e.Set)
	return nil
}

func (e *Engine) Start() {

	e.done = make(chan bool)
	LogInfo("Engine.Start")

	InitMidiIO()
	InitSynths()

	TheQuadPro.Start()

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
		LogIfError(fmt.Errorf("engine.SendOsc: client is nil"))
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
				resp, err := e.ExecuteApiFromJson(bstr)
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

	source := fmt.Sprintf("127.0.0.1:%d", port)
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
		return "", fmt.Errorf("executeengineapi: not recording")
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
			LogInfo("NewRecordingPath: Creating %s", recdir)
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
	e.SetTranspose(TheEngine.autoTransposeValues[TheEngine.autoTransposeIndex])
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
