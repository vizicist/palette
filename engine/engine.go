package engine

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/hypebeast/go-osc/osc"
)

type Engine struct {
	params          *ParamValues
	done            chan bool
	oscoutput       bool
	oscClient       *osc.Client
	recording       bool
	recordingIndex int
	recordingFile *os.File
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
	LogInfo("Engine InitLog ==============================================")

	TheEngine = &Engine{
		params: NewParamValues(),
		done:   make(chan bool),
	}

	TheCursorManager = NewCursorManager()
	TheRouter = NewRouter()
	TheProcessManager = NewProcessManager()
	TheScheduler = NewScheduler()
	TheAttractManager = NewAttractManager()
	TheQuadPro = NewQuadPro()
	TheMidiIO = NewMidiIO()
	TheErae = NewErae()

	// The Managers above should be created first so that
	// loading the Current settings can change values in them.
	TheEngine.SetDefaultValues()
	err := TheEngine.LoadCurrent()
	if err != nil {
		LogError(err)
		// but keep going
	}

	AddProcessBuiltIn("keykit")
	AddProcessBuiltIn("bidule")
	AddProcessBuiltIn("resolume")
	AddProcessBuiltIn("gui")
	AddProcessBuiltIn("mmtt")

	TheEngine.ResetLogTypes(os.Getenv("PALETTE_LOG"))
	return TheEngine
}

func (e *Engine) Start() {

	e.done = make(chan bool)
	LogInfo("Engine.Start")

	InitMidiIO()
	InitSynths()

	TheQuadPro.Start()

	go e.StartOSCListener(OSCPort)
	go e.StartHTTP(HTTPPort)

	// StartOSC xxx

	go TheScheduler.Start()
	go TheRouter.Start()
	go TheMidiIO.Start()

	if e.ParamBool("mmtt.depth") {
		go DepthRunForever()
	}
}

func (e *Engine) SetDefaultValues() {
	for nm, pd := range ParamDefs {
		if pd.Category == "engine" {
			err := e.params.SetParamValueWithString(nm, pd.Init)
			if err != nil {
				LogError(err)
			}
		}
	}
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
					LogError(fmt.Errorf("ResetLogTypes: logtype not recognized"), "logtype", d)
				} else {
					LogEnabled[d] = true
				}
				return
			}
		}
	}
}

func (e *Engine) sendToOscClients(msg *osc.Message) {
	if e.oscoutput {
		if e.oscClient == nil {
			e.oscClient = osc.NewClient(LocalAddress, EventClientPort)
			// oscClient is guaranteed to be non-nil
		}
		err := e.oscClient.Send(msg)
		LogIfError(err)
		LogOfType("cursor", "Router sending to OSC client", "msg", msg)
	}
}

func (e *Engine) StartOSCListener(port int) {

	source := fmt.Sprintf("%s:%d", LocalAddress, port)

	d := osc.NewStandardDispatcher()

	err := d.AddMsgHandler("*", func(msg *osc.Message) {
		TheRouter.oscInputChan <- OSCEvent{Msg: msg, Source: source}
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
