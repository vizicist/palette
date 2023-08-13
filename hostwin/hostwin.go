package hostwin

import (
	"fmt"
	"io"
	"os"
	"math/rand"
	"net/http"

	"github.com/hypebeast/go-osc/osc"
	"github.com/vizicist/palette/kit"
)

type HostWin struct {
	done      chan bool
	params    *kit.ParamValues
	oscoutput bool
	oscClient *osc.Client
	resolumeClient   *osc.Client
	freeframeClients map[string]*osc.Client

	// recordingIndex int
	// recordingFile  *os.File
	// recordingPath  string
	// recordingMutex sync.RWMutex
	// currentPitchOffset  *atomic.Int32

	biduleClient *osc.Client
	bidulePort   int
}

// var TheHost *HostWin
var TheRand *rand.Rand

func NewHost(logname string) *HostWin {

	h := &HostWin{
		resolumeClient:   osc.NewClient(LocalAddress, ResolumePort),
		freeframeClients: map[string]*osc.Client{},
	}
	TheHost = h

	InitLog(logname)
	// InitMisc()

	err := h.loadResolumeJSON()
	if err != nil {
		LogIfError(err)
	}
	kit.InitParams()
	TheProcessManager = NewProcessManager()
	// Fixed rand sequence, better for testing
	TheRand = rand.New(rand.NewSource(1))

	h.params = kit.NewParamValues()

	h.biduleClient = osc.NewClient(LocalAddress, BidulePort)
	h.bidulePort=   BidulePort

	TheRouter = NewRouter()
	TheMidiIO = NewMidiIO()

	InitLogTypes()

	// Set all the default engine.* values
	for nm, pd := range kit.ParamDefs {
		if pd.Category == "engine" {
			err := h.params.SetParamValueWithString(nm, pd.Init)
			if err != nil {
				LogIfError(err)
			}
		}
	}
	err = h.LoadCurrent()
	if err != nil {
		LogIfError(err)
	}

	// TheEngine = e

	// This need to be done after engine parameters are loaded
	TheProcessManager.AddBuiltins()

	kit.RegisterHost(h)

	return h
}

func (h HostWin) EveryTick() {
	TheProcessManager.checkProcess()
}

func (h HostWin) InputEventLock() {
	TheRouter.inputEventMutex.Lock()
}

func (h HostWin) InputEventUnlock() {
	TheRouter.inputEventMutex.Unlock()
}

func Start() {
	TheHost.Start()
}

func WaitTillDone() {
	TheHost.WaitTillDone()
}

func (h HostWin) SaveDataInFile(data []byte, category string, filename string) error {
	path, err := WritableSavedFilePath(category, filename, ".json")
	if err != nil {
		LogIfError(err)
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (h HostWin) GetDataInFile(category string, filename string) (bytes []byte, err error) {
	path, err := ReadableSavedFilePath(category, filename, ".json")
	if err != nil {
		LogIfError(err)
		return nil, err
	}
	return os.ReadFile(path)
}

func (h HostWin) GetConfigFileData(filename string) ([]byte, error) {
	path := ConfigFilePath(filename)
	return os.ReadFile(path)
}

func (h HostWin) GenerateVisualsFromCursor(ce kit.CursorEvent, patchName string) {
	// send an OSC message to Resolume
	msg := kit.CursorToOscMsg(ce)
	h.ToFreeFramePlugin(patchName, msg)

}

func (h HostWin) GetParam(name string) (string, error) {
	return h.params.Get(name)
}

func (h HostWin) GetParamBool(name string) (bool, error) {
	return kit.GetParamBool(name)
}

// func (h HostWin) Get(name string) string {
// 	return e.params.Get(name)
// }

func (h HostWin) GetSavedData(category string, filename string) ([]byte, error) {
	err := fmt.Errorf("GetSavedData needs work")
	kit.LogError(err)
	return []byte{}, err
}

func (h HostWin) SaveCurrent() (err error) {
	data, err := h.GetSavedData("engine","_Current")
	if err != nil {
		return err
	}
	return h.SaveDataInFile(data,"engine","_Current")
}

func (h HostWin) LoadCurrent() (err error) {
	return h.LoadEngineParams("_Current")
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

// func (e *Engine) SendToOscCLients(oscMessage *osc.Message) {
// 	e.sendToOscClients(oscMessage)
// }

func (h HostWin) LoadEngineParams(fname string) (err error) {
	path, err := ReadableSavedFilePath("engine", fname, ".json")
	if err != nil {
		return err
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	paramsmap, err := kit.LoadParamsMap(bytes)
	if err != nil {
		return err
	}
	h.params.ApplyValuesFromMap("engine", paramsmap, h.Set)
	return nil
}

func (h HostWin) HandleIncomingMidiEvent(me kit.MidiEvent) {
	err := fmt.Errorf("HandleIncomingMidiEvent needs work!")
	kit.LogIfError(err)
}

func (h HostWin) IsLogging(logtype string) bool {
	return IsLogging(logtype)
}

func (h HostWin) LogError(err error, keysAndValues ...any) {
	LogError(err, keysAndValues...)
}

func (h HostWin) LogIfError(err error, keysAndValues ...any) {
	LogIfError(err, keysAndValues...)
}

func (h HostWin) LogInfo(msg string, keysAndValues ...any) {
	LogInfo(msg, keysAndValues...)
}

func (h HostWin) LogWarn(msg string, keysAndValues ...any) {
	LogWarn(msg, keysAndValues...)
}

func (h HostWin) LogOfType(logtypes string, msg string, keysAndValues ...any) {
	LogOfType(logtypes, msg, keysAndValues...)
}

func (h HostWin) ResetAudio() {
	LogWarn("ResetAudio needs work")
}

func (h HostWin) SendMIDI(bytes []byte) {
	LogWarn("SendMIDI needs work")
}

// func (h HostWin) ToFreeFramePlugin(patchName string, msg *osc.Message) {
// 	// send an OSC message to Resolume
// 	h.SendToOscClients(msg)
// }

func (h HostWin) Start() {

	h.done = make(chan bool)
	LogInfo("Engine.Start")

	InitMidiIO()

	go h.StartOscListener(OscPort)
	go h.StartHttp(EngineHttpPort)

	go TheRouter.Start()
	go TheMidiIO.Start()

	go TheBidule().Reset()

	// TheAttractManager.setAttractMode(true)

	// if ParamBool("mmtt.depth") {
	// 	go DepthRunForever()
	// }
}

func (h HostWin) WaitTillDone() {
	<-h.done
}

func (h HostWin) SayDone() {
	h.done <- true
}

func (h HostWin) SendOsc(client *osc.Client, msg *osc.Message) {
	if client == nil {
		LogIfError(fmt.Errorf("engine.SendOsc: client is nil"))
		return
	}

	LogOfType("osc", "Sending to OSC client", "client", client.IP(), "port", client.Port(), "msg", msg)
	err := client.Send(msg)
	LogIfError(err)
}

func (h HostWin) SendToOscClients(msg *osc.Message) {
	if h.oscoutput {
		if h.oscClient == nil {
			h.oscClient = osc.NewClient(LocalAddress, EventClientPort)
			// oscClient is guaranteed to be non-nil
		}
		h.SendOsc(h.oscClient, msg)
	}
}

func (h HostWin) StartOscListener(port int) {

	source := fmt.Sprintf("%s:%d", LocalAddress, port)

	d := osc.NewStandardDispatcher()

	err := d.AddMsgHandler("*", func(msg *osc.Message) {
		TheRouter.oscInputChan <- kit.OscEvent{Msg: msg, Source: source}
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
func (h HostWin) StartHttp(port int) {

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
				resp, err := TheHost.ExecuteApiFromJson(bstr)
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
