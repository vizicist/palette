package engine

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

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
}

var TheEngine *Engine
var engineSysex sync.Mutex

func InitMisc() {
	InitParams()
	TheProcessManager = NewProcessManager()
}

func InitEngine() {

	if TheEngine != nil {
		LogIfError(fmt.Errorf("TheEngine already exists, there can be only one"))
		return
	}

	engineSysex.Lock()
	defer engineSysex.Unlock()

	LogInfo("Engine InitLog ==============================================")

	e := &Engine{
		done: make(chan bool),
	}

	e.params = NewParamValues()

	TheCursorManager = NewCursorManager()
	TheRouter = NewRouter()
	TheScheduler = NewScheduler()
	TheAttractManager = NewAttractManager()
	TheQuadPro = NewQuadPro()
	TheMidiIO = NewMidiIO()
	TheErae = NewErae()

	e.ResetLogTypes(os.Getenv("PALETTE_LOG"))

	// Set all the default engine.* values
	for nm, pd := range ParamDefs {
		if pd.Category == "engine" {
			err := e.params.setParamValueWithString(nm, pd.Init)
			if err != nil {
				LogIfError(err)
			}
		}
	}
	err := e.LoadCurrent()
	if err != nil {
		LogIfError(err)
	}

	TheEngine = e

	// This need to be done after engine parameters are loaded
	TheProcessManager.AddBuiltins()
}

func Start() {
	TheEngine.Start()
}

func WaitTillDone() {
	TheEngine.WaitTillDone()
}

func GetParam(name string) string {
	if TheEngine == nil {
		LogIfError(fmt.Errorf("GetParam called before NewEngine!?)"))
		return ""
	}
	return TheEngine.Get(name)
}

func (e *Engine) Get(name string) string {
	return e.params.Get(name)
}

func (e *Engine) SaveCurrent() (err error) {
	return e.params.Save("engine", "_Current")
}

func (e *Engine) LoadCurrent() (err error) {
	path, err := SavedFilePath("engine", "_Current")
	if err != nil {
		return err
	}
	paramsmap, err := LoadParamsMap(path)
	if err != nil {
		return err
	}
	e.params.ApplyValuesFromMap("engine", paramsmap, e.Set)
	return nil
}

func (e *Engine) Start() {

	e.done = make(chan bool)
	LogInfo("Engine.Start")

	InitMidiIO()
	InitSynths()

	TheQuadPro.Start()

	go e.StartOSCListener(OSCPort)
	go e.StartHTTP(HTTPPort)

	go TheScheduler.Start()
	go TheRouter.Start()
	go TheMidiIO.Start()

	// if ParamBool("mmtt.depth") {
	// 	go DepthRunForever()
	// }
}

func (e *Engine) WaitTillDone() {
	<-e.done
}

func (e *Engine) StopMe() {
	e.done <- true
}

func (e *Engine) ResetLogTypes(logtypes string) {
	for logtype := range LogEnabled {
		LogEnabled[logtype] = false
	}
	if logtypes != "" {
		darr := strings.Split(logtypes, ",")
		for _, d := range darr {
			if d != "" {
				d := strings.ToLower(d)
				LogInfo("Turning logging ON for", "logtype", d)
				_, ok := LogEnabled[d]
				if !ok {
					LogIfError(fmt.Errorf("ResetLogTypes: logtype not recognized"), "logtype", d)
				} else {
					LogEnabled[d] = true
				}
			}
		}
	}
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

func (e *Engine) StartOSCListener(port int) {

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

// StartHTTP xxx
func (e *Engine) StartHTTP(port int) {

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
				resp, err := e.ExecuteAPIFromJson(bstr)
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
			ScheduleAt(CurrentClick(), ce)
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
