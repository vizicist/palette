package kit

import (
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	json "github.com/goccy/go-json"

	"github.com/hypebeast/go-osc/osc"
)

type Engine struct {
	done      chan bool
	oscoutput bool
	oscClient *osc.Client
	// params         *ParamValues

	currentPitchOffset  *atomic.Int32
	autoTransposeOn     bool
	autoTransposeNext   Clicks
	autoTransposeClicks Clicks // time between auto transpose changes
	autoTransposeIndex  int    // current place in transposeValues
	autoTransposeValues []int
}

var theEngine *Engine
var engineSysex sync.Mutex

func InitEngine() {

	InitializeClicks()

	if theEngine != nil {
		LogIfError(fmt.Errorf("theEngine already exists, there can be only one"))
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

	StartEmbeddedNATSAndConnectEngine()

	theCursorManager = NewCursorManager()
	theRouter = NewRouter()
	theScheduler = NewScheduler()
	theAttractManager = NewAttractManager()
	theStepper = NewStepper()
	theQuad = NewQuad()
	theMidiIO = NewMidiIO()
	theErae = NewErae()

	InitLogTypes()

	theEngine = e

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

func LocalEngineAPI(api string, args ...string) (map[string]string, error) {
	return EngineHTTPAPI(LocalAddress, api, args...)
}

func EngineHTTPAPI(host string, api string, args ...string) (map[string]string, error) {

	if len(args)%2 != 0 {
		return nil, fmt.Errorf("RemoteAPI: odd number of args, should be even")
	}

	// Build request map
	request := map[string]string{"api": api}
	for i := 0; i < len(args); i += 2 {
		request[args[i]] = args[i+1]
	}

	jsonBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("EngineHTTPAPI: failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("http://%s:%d/api", host, engineHTTPPort)
	return HTTPAPIRaw(url, string(jsonBytes))
}

func EngineNatsAPI(host string, cmd string, timeout time.Duration) (result string, err error) {
	if !natsIsConnected {
		return "", fmt.Errorf("EngineNatsAPI: NATS is not connected")
	}
	subject := fmt.Sprintf("to_palette.%s.api", host)
	retdata, err := NatsRequest(subject, cmd, timeout)
	LogIfError(err)
	return retdata, err
}

func EngineCloseNats() {
	NatsDisconnect()
	StopEmbeddedNATSServer()
}

func StartEngine() {
	theEngine.Start()
}

func WaitTillDone() {
	theEngine.WaitTillDone()
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
	return SaveCurrentGlobalParams()
}

func (e *Engine) Start() {

	e.done = make(chan bool)

	LogInfo("Engine.Start")

	InitMIDIIO()
	InitPitchSets()
	InitSynths()

	theQuad.Start()

	go e.StartOscListener(oscPort)
	go e.StartHTTP(engineHTTPPort)

	go theScheduler.Start()
	go theRouter.Start()
	go theMidiIO.Start()
	StartUIStateProjector()

	go TheBidule().Reset()

	// theAttractManager.setAttractMode(true)

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
			e.oscClient = osc.NewClient(LocalAddress, eventClientPort)
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
		theRouter.oscInputChan <- OscEvent{Msg: msg, Source: source}
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

// jsonAPIHandler wraps an API handler with the common JSON-response boilerplate:
// it sets the content-type, runs handler, and writes the body with a 200 status
// on success or a 400 status (with an error body) on failure.
func jsonAPIHandler(handler func(req *http.Request) (string, error)) http.HandlerFunc {
	return func(responseWriter http.ResponseWriter, req *http.Request) {
		responseWriter.Header().Set("Content-Type", "application/json")
		response, err := handler(req)
		if err != nil {
			responseWriter.WriteHeader(http.StatusBadRequest)
			response = ErrorResponse(err)
		} else {
			responseWriter.WriteHeader(http.StatusOK)
		}
		if _, werr := responseWriter.Write([]byte(response)); werr != nil {
			LogIfError(werr)
		}
	}
}

// StartHTTP xxx
func (e *Engine) StartHTTP(port int) {

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

	// Serve web UI at root
	http.Handle("/", WebUIHandler())

	http.HandleFunc("/api", jsonAPIHandler(func(req *http.Request) (string, error) {
		if req.Method != "POST" {
			return "", fmt.Errorf("HTTP server unable to handle method=%s", req.Method)
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return "", err
		}
		resp, err := ExecuteAPIFromJSON(string(body))
		if err != nil {
			return "", err
		}
		return ResultResponse(resp), nil
	}))

	http.HandleFunc("/nats/api", jsonAPIHandler(func(req *http.Request) (string, error) {
		if req.Method != "POST" {
			return "", fmt.Errorf("NATS API proxy unable to handle method=%s", req.Method)
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return "", err
		}
		args, err := StringMap(string(body))
		if err != nil {
			return "", fmt.Errorf("NATS API proxy: bad JSON format")
		}
		host := ExtractAndRemoveValueOf("host", args)
		if host == "" {
			host = Hostname()
		}
		bytes, err := json.Marshal(args)
		if err != nil {
			return "", err
		}
		return EngineNatsAPI(host, string(bytes), time.Second)
	}))

	source := fmt.Sprintf("%s:%d", engineHTTPBindAddress, port)
	err := http.ListenAndServe(source, nil)
	if err != nil {
		LogIfError(err)
	}
}

func (e *Engine) SetTranspose(i int) {
	e.currentPitchOffset.Store(int32(i))
	LogOfType("transpose", "SetTranspose", "i", i)
	// LogInfo("Engine.SetTranspose", "schedule", theScheduler.ToString())
	// LogInfo("Engine.SetTranspose", "pending", theScheduler.PendingToString())
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
	transpose := theEngine.autoTransposeValues[theEngine.autoTransposeIndex]
	e.SetTranspose(transpose)
	LogOfType("transpose", "TransposeTo", "index", e.autoTransposeIndex, "transpose", transpose, "newclick", newclick)
	// theScheduler.SendAllPendingNoteoffs()
}
