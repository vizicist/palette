package engine

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hypebeast/go-osc/osc"
)

type Engine struct {
	ProcessManager   *ProcessManager
	Router           *Router
	Scheduler        *Scheduler
	CursorManager    *CursorManager
	responderManager *ResponderManager
	killme           bool // true if Engine should be stopped
}

var HTTPPort = 3330
var OSCPort = 3333
var AliveOutputPort = 3331
var ResolumePort = 3334

var theEngine *Engine

func TheEngine() *Engine {
	if theEngine == nil {
		InitLog("engine")
		theEngine = newEngine()
	}
	return theEngine
}

func newEngine() *Engine {
	e := &Engine{}
	e.initDebug()
	e.ProcessManager = NewProcessManager()
	e.Router = NewRouter()
	e.Scheduler = NewScheduler()
	e.CursorManager = NewCursorManager()
	e.responderManager = NewResponderManager()
	return e
}

func ProcessStatus() string {
	return TheEngine().ProcessManager.ProcessStatus()
}

func StopRunning(what string) {
	TheEngine().ProcessManager.StopRunning(what)
}

type CreateResponderFunc func(*ResponderContext) Responder

func AddResponder(name string, responder Responder) {
	TheEngine().responderManager.AddResponder(name, responder)
}

func ActivateResponder(name string) {
	TheEngine().responderManager.ActivateResponder(name)
}

func DeactivateResponder(name string) {
	TheEngine().responderManager.DeactivateResponder(name)
}

func (e *Engine) handleCursorEvent(ce CursorEvent) {
	TheEngine().CursorManager.handleCursorEvent(ce)
	TheEngine().responderManager.handleCursorEvent(ce)
}

func (e *Engine) Start() {

	Log.Debugf("====================== Palette Engine is starting\n")

	e.Router.Restart()

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
	go e.Scheduler.Start()
	go e.StartCursorInput()
	go e.InputListener()

	if ConfigBoolWithDefault("depth", false) {
		go DepthRunForever()
	}
}

func (e *Engine) StopMe() {
	e.killme = true
}

func (e *Engine) initDebug() {
	debug := ConfigValueWithDefault("debug", "")
	darr := strings.Split(debug, ",")
	for _, d := range darr {
		if d != "" {
			Log.Debugf("Turning Debug ON for %s\n", d)
			setDebug(d, true)
		}
	}
}

// InputListener listens for local device inputs (OSC, MIDI)
// We could have separate goroutines for the different inputs, but doing
// them in a single select eliminates some need for locking.
func (e *Engine) InputListener() {
	for !e.killme {
		select {
		case msg := <-e.Router.OSCInput:
			e.Router.handleOSCInput(msg)
		case event := <-e.Router.MIDIInput:
			e.Router.handleMidiInput(event)
		case event := <-e.Router.CursorInput:
			e.CursorManager.handleCursorEvent(event)
		default:
			// Log.Debugf("Sleeping 1 ms - now=%v\n", time.Now())
			time.Sleep(time.Millisecond)
		}
	}
	Log.Debugf("InputListener is being killed\n")
}

// StartCursorInput xxx
func (e *Engine) StartCursorInput() {
	err := LoadMorphs()
	if err != nil {
		Log.Debugf("StartCursorInput: LoadMorphs err=%s\n", err)
	}
	go StartMorph(e.CursorManager.handleCursorEvent, 1.0)
}

// StartMIDI listens for MIDI events and sends their bytes to the MIDIInput chan
func (e *Engine) StartMIDI() {

	if MIDI.Input == nil {
		Log.Debugf("StartMIDI: there is no MIDI input\n")
	}

	if EraeEnabled {
		e.Router.SetMIDIEventHandler(HandleEraeMIDI)
	}
	MIDI.Start()
}

// StartOSC xxx
func (e *Engine) StartOSC(port int) {

	handler := e.Router.OSCInput
	source := fmt.Sprintf("127.0.0.1:%d", port)

	d := osc.NewStandardDispatcher()

	err := d.AddMsgHandler("*", func(msg *osc.Message) {
		handler <- OSCEvent{Msg: msg, Source: source}
	})
	if err != nil {
		Log.Debugf("ERROR! %s\n", err.Error())
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

		// vals := req.URL.Query()
		// Log.Debugf("/api vals=%v\n", vals)

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
