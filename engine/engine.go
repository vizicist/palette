package engine

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hypebeast/go-osc/osc"
)

type Engine struct {
	// ProcessManager *ProcessManager
	Router        *Router
	Scheduler     *Scheduler
	PluginManager *PluginManager
	// CursorManager *CursorManager
	done          chan bool
}

var theEngine *Engine

func TheEngine() *Engine {
	if theEngine == nil {
		InitLog("engine")
		theEngine = newEngine()
	}
	return theEngine
}

func TheRouter() *Router {
	return TheEngine().Router
}

func newEngine() *Engine {
	e := &Engine{}
	e.initDebug()
	// e.ProcessManager = NewProcessManager()
	e.Router = NewRouter()
	e.Scheduler = NewScheduler()
	e.PluginManager = NewPluginManager()
	return e
}

func RegisterPlugin(name string, plugin PluginFunc) {
	ThePluginManager().RegisterPlugin(name, plugin)
}

// func (e *Engine) HandleCursorEvent(ce CursorEvent) {
// 	TheEngine().cursorManager.HandleCursorEvent(ce)
// 	TheEngine().agentManager.HandleCursorEvent(ce)
// }

func (e *Engine) StartPlugin(name string) {
	err := e.PluginManager.StartPlugin(name)
	if err != nil {
		LogError(err)
	}
}

func (e *Engine) Start() {

	e.done = make(chan bool)
	LogInfo("Engine.Start")

	InitMIDI()
	InitSynths()

	// Eventually the defult should be "" rather than "ppro"
	plugins := ConfigStringWithDefault("plugins", "ppro")

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

func (e *Engine) WaitTillDone() {
	<-e.done
}

func (e *Engine) StopMe() {
	e.done <- true
}

func (e *Engine) initDebug() {
	logtypes := ConfigValueWithDefault("debug", "")
	// Migrate to using log rather than debug
	if logtypes == "" {
		logtypes = ConfigValueWithDefault("log", "")
	}
	darr := strings.Split(logtypes, ",")
	for _, d := range darr {
		if d != "" {
			LogInfo("Turning logging ON for", "logtype", d)
			SetLogTypeEnabled(d, true)
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
		LogWarn("StartOSC", "err", err)
	}

	server := &osc.Server{
		Addr:       source,
		Dispatcher: d,
	}
	server.ListenAndServe()
}

// StartHTTP xxx
func (e *Engine) StartHTTP(port int) {

	http.HandleFunc("/api", func(responseWriter http.ResponseWriter, req *http.Request) {

		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusOK)

		response := ""
		defer func() {
			responseWriter.Write([]byte(response))
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
	http.ListenAndServe(source, nil)
}
