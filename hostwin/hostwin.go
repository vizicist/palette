package hostwin

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"

	"gitlab.com/gomidi/midi/v2/drivers"
	"github.com/hypebeast/go-osc/osc"
	"github.com/vizicist/palette/kit"
)

type HostWin struct {
	logName          string
	done             chan bool
	oscoutput        bool
	oscClient        *osc.Client
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

var TheWinHost *HostWin
var TheRand *rand.Rand

func NewHost(logname string) *HostWin {

	h := &HostWin{
		logName:          logname,
		resolumeClient:   osc.NewClient(LocalAddress, ResolumePort),
		freeframeClients: map[string]*osc.Client{},
		biduleClient:     osc.NewClient(LocalAddress, BidulePort),
		bidulePort:       BidulePort,
	}
	TheWinHost = h
	InitLog(h.logName)
	TheProcessManager = NewProcessManager()
	return h
}

func (h HostWin) Init() error {

	TheProcessManager.AddBuiltins()

	err := h.loadResolumeJSON()
	if err != nil {
		LogIfError(err)
		return err
	}
	// Fixed rand sequence, better for testing
	TheRand = rand.New(rand.NewSource(1))

	h.biduleClient = osc.NewClient(LocalAddress, BidulePort)
	h.bidulePort = BidulePort

	TheRouter = NewRouter()
	TheMidiIO = NewMidiIO()

	return err
}

func (h HostWin) Start() {

	// This need to be done after engine parameters are loaded

	h.done = make(chan bool)
	LogInfo("Engine.Start")

	go h.StartOscListener(OscPort)
	go h.StartHttp(EngineHttpPort)

	go TheRouter.Start()
	go TheMidiIO.Start()

	go h.ResetAudio()

	// if ParamBool("mmtt.depth") {
	// 	go DepthRunForever()
	// }
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

func (h HostWin) SaveDataInFile(data []byte, category string, filename string) error {
	path, err := WritableSavedFilePath(category, filename)
	if err != nil {
		LogIfError(err)
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (h HostWin) GetSavedData(category string, filename string) (bytes []byte, err error) {
	path, err := ReadableSavedFilePath(category, filename)
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

func (h HostWin) HandleIncomingMidiEvent(me kit.MidiEvent) {
	err := fmt.Errorf("HandleIncomingMidiEvent needs work")
	kit.LogIfError(err)
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

// func (h HostWin) LogOfType(logtypes string, msg string, keysAndValues ...any) {
// 	LogOfType(logtypes, msg, keysAndValues...)
// }

func (h HostWin) SendMIDI(output any, bytes []byte) error {
	driverout, ok := output.(drivers.Out)
	if !ok {
		LogWarn("unable to convert output to drivers.Out")
		return fmt.Errorf("unable to convert output to drivers.Out")
	}
	return driverout.Send(bytes)
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
				response = kit.ErrorResponse(err)
			} else {
				bstr := string(body)
				_ = bstr
				resp, err := kit.ExecuteApiFromJson(bstr)
				if err != nil {
					response = kit.ErrorResponse(err)
				} else {
					response = kit.ResultResponse(resp)
				}
			}
		default:
			response = kit.ErrorResponse(fmt.Errorf("HTTP server unable to handle method=%s", req.Method))
		}
	})

	source := fmt.Sprintf("127.0.0.1:%d", port)
	err := http.ListenAndServe(source, nil)
	if err != nil {
		LogIfError(err)
	}
}

func (h HostWin) StartRunning(process string) (err error) {
	return TheProcessManager.StartRunning(process)
}

func (h HostWin) KillProcess(process string) (err error) {
	return TheProcessManager.KillProcess(process)
}

func (h HostWin) SetMidiInput(midiInputName string) error {
	return TheMidiIO.SetMidiInput(midiInputName)
}