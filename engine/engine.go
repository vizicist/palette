package engine

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hypebeast/go-osc/osc"
)

type Engine struct {
	ProcessManager *ProcessManager
	Router         *Router
	Scheduler      *Scheduler
	done           chan bool
}

type TaskContext struct {
	context.Context
	*Engine
	scheduler *Scheduler
	taskFunc  TaskFunc
	taskData  TaskData
	// taskContext *context.Context
	// agentParams     *ParamValues
	// layerParams     map[string]*ParamValues
	// scale           *Scale
	sources map[string]bool
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
	e.ProcessManager = NewProcessManager()
	e.Router = NewRouter()
	e.Scheduler = NewScheduler()
	return e
}

func ProcessStatus() string {
	return TheEngine().ProcessManager.ProcessStatus()
}

func StopRunning(what string) {
	TheEngine().ProcessManager.StopRunning(what)
}

func RegisterTask(name string, f TaskFunc, taskContext context.Context) {
	TheRouter().taskManager.RegisterTask(name, f, taskContext)
}

/*
func ActivateAgent(name string) {
	TheRouter().taskManager.ActivateAgent(name)
}

func DeactivateAgent(name string) {
	TheRouter().taskManager.DeactivateAgent(name)
}
*/

// func (e *Engine) handleCursorEvent(ce CursorEvent) {
// 	TheEngine().cursorManager.handleCursorEvent(ce)
// 	TheEngine().taskManager.handleCursorEvent(ce)
// }

func (e *Engine) StartTask(name string) {
	Info("Engine.StartTask needs work", "task", name)
	// e.Router.taskManager.StartTask(name)
}

func (e *Engine) Start(done chan bool) {

	e.done = done
	Info("Engine.Start")

	// Normally, the engine should never die, but if it does,
	// other processes (e.g. resolume, bidule) may be left around.
	// So, unless told otherwise, we kill everything to get a clean start.
	if ConfigBoolWithDefault("killonstartup", true) {
		e.ProcessManager.killAll()
	}

	InitMIDI()
	InitSynths()

	go e.StartOSC(OSCPort)
	go e.StartHTTP(HTTPPort)
	// go r.StartNATSClient()
	go e.StartMIDI()
	go e.Scheduler.Start(e.Scheduler.DefaultOnClick)
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
			Info("Turning logging ON for", "logtype", d)
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
		Warn("StartOSC", "err", err)
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
					response = resp
				}
			}
		default:
			response = ErrorResponse(fmt.Errorf("HTTP server unable to handle method=%s", req.Method))
		}
	})

	source := fmt.Sprintf("127.0.0.1:%d", port)
	http.ListenAndServe(source, nil)
}
