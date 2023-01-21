package engine

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"strings"

	"github.com/hypebeast/go-osc/osc"
)

type Engine struct {
	// ProcessManager *ProcessManager
	params         *ParamValues
	Router         *Router
	Scheduler      *Scheduler
	ProcessManager *ProcessManager
	// CursorManager *CursorManager
	done chan bool
}

var TheEngine *Engine
var engineSysex sync.Mutex

func NewEngine() *Engine {

	if TheEngine != nil {
		LogError(fmt.Errorf("TheEngine already exists, there can be only one"))
		return TheEngine
	}

	engineSysex.Lock()
	defer engineSysex.Unlock()

	InitLog("engine")
	TheEngine = &Engine{
		params:         NewParamValues(),
		Router:         NewRouter(),
		Scheduler:      NewScheduler(),
		ProcessManager: NewProcessManager(),
		done:           make(chan bool),
	}
	TheEngine.SetLogTypes()
	LogInfo("Engine InitLog ==============================================")
	TheEngine.SetDefaultValues()
	err := TheEngine.LoadCurrent()
	if err != nil {
		LogError(err)
		// but keep going
	}
	return TheEngine
}

func TheRouter() *Router {
	return TheEngine.Router
}

func TheScheduler() *Scheduler {
	return TheEngine.Scheduler
}

// func (e *Engine) HandleCursorEvent(ce CursorEvent) {
// 	TheEngine.cursorManager.HandleCursorEvent(ce)
// 	TheEngine.agentManager.HandleCursorEvent(ce)
// }

func (e *Engine) StartPlugin(name string) {
	err := PluginsStartPlugin(name)
	if err != nil {
		LogError(err)
	}
}

func (e *Engine) Start() {

	e.done = make(chan bool)
	LogInfo("Engine.Start")

	InitMIDI()
	InitSynths()

	// Eventually the defult should be "" rather than "quadpro"
	plugins := e.GetWithDefault("plugins", "quadpro")

	// Plugins should be started before anything,
	// to guarante that the "start" event will be the first

	for _, nm := range strings.Split(plugins, ",") {
		e.StartPlugin(nm)
	}

	go e.StartOSC(OSCPort)
	go e.StartHTTP(HTTPPort)
	// go r.StartNATSClient()
	go e.StartMIDI()
	go e.Scheduler.Start()
	go e.Router.Start()

	if ConfigBoolWithDefault("depth", false) {
		go DepthRunForever()
	}
}

func (e *Engine) SetDefaultValues() {
	LogInfo("SetDefaultValues start")
	for nm, pd := range ParamDefs {
		if pd.Category == "engine" {
			err := e.params.SetParamValueWithString(nm, pd.Init)
			if err != nil {
				LogError(err)
			}
			LogInfo("SetDefaultValues", "name", nm, "value", pd.Init)
		}
	}
}

func (e *Engine) WaitTillDone() {
	<-e.done
}

func (e *Engine) StopMe() {
	e.done <- true
}

func (e *Engine) SetLogTypes() {
	logtypes := os.Getenv("PALETTE_LOG")
	if logtypes != "" {
		darr := strings.Split(logtypes, ",")
		for _, d := range darr {
			if d != "" {
				LogInfo("Turning logging ON for", "logtype", d)
				SetLogTypeEnabled(d, true)
			}
		}
	}
}

// StartMIDI listens for MIDI events and sends their bytes to the MIDIInput chan
func (e *Engine) StartMIDI() {
	MIDI.Start(e.Router.midiInputChan)
}

// StartOSC xxx
func (e *Engine) StartOSC(port int) {

	source := fmt.Sprintf("%s:%d", LocalAddress, port)

	d := osc.NewStandardDispatcher()

	err := d.AddMsgHandler("*", func(msg *osc.Message) {
		e.Router.OSCInput <- OSCEvent{Msg: msg, Source: source}
	})
	if err != nil {
		LogError(err)
	}

	server := &osc.Server{
		Addr:       source,
		Dispatcher: d,
	}
	err = server.ListenAndServe()
	if err != nil {
		LogError(err)
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
				LogError(err)
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
		LogError(err)
	}
}
