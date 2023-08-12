package hostwin

import (
	"fmt"
	"io"
	"os"
	"math/rand"
	"net/http"
	"sync"

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

}

var TheHost *HostWin
var engineSysex sync.Mutex
var TheRand *rand.Rand
var TheKit *kit.Kit

func NewHost() *HostWin {
	h := &HostWin{
		resolumeClient:   osc.NewClient(LocalAddress, ResolumePort),
		freeframeClients: map[string]*osc.Client{},
	}
	TheHost = h
	return h
}

func (host HostWin) Init() {
	err := host.loadResolumeJSON()
	if err != nil {
		LogIfError(err)
	}
	InitParams()
	TheProcessManager = NewProcessManager()
	TheKit = kit.NewKit()
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
		done: make(chan bool),
	}

	e.params = kit.NewParamValues()

	kit.RegisterHost(e)

	TheRouter = NewRouter()
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

func (h HostWin) GenerateVisualsFromCursr(ce kit.CursorEvent, patchName string) {
	// send an OSC message to Resolume
	msg := kit.CursorToOscMsg(ce)
	TheResolume().ToFreeFramePlugin(patchName, msg)

}

func (h HostWin) GetParam(name string) (string, error) {
	return h.params.Get(name)
}

// func (h HostWin) Get(name string) string {
// 	return e.params.Get(name)
// }

func (h HostWin) SaveCurrent() (err error) {
	return e.params.Save("engine", "_Current")
}

func (h HostWin) LoadCurrent() (err error) {
	return e.LoadEngineParams("_Current")
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
	paramsmap, err := kit.LoadParamsMap(path)
	if err != nil {
		return err
	}
	e.params.ApplyValuesFromMap("engine", paramsmap, e.Set)
	return nil
}

func (e *Engine) ToFreeFramePlugin(patchName string, msg *osc.Message) {
	// send an OSC message to Resolume
	e.SendToOscCLients(msg)
}

func (e *Engine) Start() {

	e.done = make(chan bool)
	LogInfo("Engine.Start")

	InitMidiIO()

	go e.StartOscListener(OscPort)
	go e.StartHttp(EngineHttpPort)

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
