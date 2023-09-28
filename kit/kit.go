package kit

import (
	"bytes"
	"io"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	midi "gitlab.com/gomidi/midi/v2"
)

var MidiOctaveShift int
var Midisetexternalscale bool
var Midithru bool
var MidiThruScadjust bool
var MidiNumDown int

var TheHost Host

func RegisterAndInit(h Host) error {
	TheHost = h
	return Init()
}

func RegisterHost(h Host) {
	TheHost = h
}

func Init() error {

	InitParamDefs()
	if TheHost == nil {
		return fmt.Errorf("kit.Init called without registering a Host")
	}

	LoadEngineParams("_Current.json")

	logtypes, err := GetParam("engine.log")
	if err == nil {
		SetLogTypes(logtypes)
	}

	TheCursorManager = NewCursorManager()
	TheScheduler = NewScheduler()
	TheAttractManager = NewAttractManager()
	TheQuadPro = NewQuadPro()

	err = TheHost.Init()
	if err != nil {
		return err
	}

	enabled, err := GetParamBool("engine.attractenabled")
	LogIfError(err)
	TheAttractManager.SetAttractEnabled(enabled)

	natsEnabled, err := GetParamBool("engine.natsenabled")
	LogIfError(err)

	InitSynths()

	if natsEnabled {
		TheNats = NewVizNats()
		natsUser := os.Getenv("NATS_USER")
		natsPassword := os.Getenv("NATS_PASSWORD")
		natsUrl := os.Getenv("NATS_URL")
		err = TheNats.Connect(natsUser, natsPassword, natsUrl)
		LogIfError(err)
	}

	return err
}

func StartEngine() {
	LogInfo("Engine.Start")
	go StartHttp(EngineHttpPort)
	TheHost.Start()
	TheQuadPro.Start()
	go TheScheduler.Start()
	TheNats.Subscribe("toengine.>", NatsHandler)
}

// StartHttp xxx
func StartHttp(port int) {

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
				resp, err := ExecuteApiFromJson(bstr)
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


func RemoteEngineApi(api string, data string) (string, error) {
	LogInfo("RemoteEngineApi before Request", "api", api, "data", data)
	result, err := TheNats.Request("engine.api."+api, data, time.Second)
	if err == nats.ErrNoResponders {
		return "", err
	}
	LogIfError(err)
	LogInfo("RemoteEngineApi after Request", "result", result)
	return result, nil
}

func NatsHandler(msg *nats.Msg) {
	data := string(msg.Data)
	LogInfo("NatsHandler", "subject", msg.Subject, "data", data)
}

func LoadEngineParams(fname string) {
	// First set the defaults
	for nm, d := range ParamDefs {
		if strings.HasPrefix(nm, "engine.") {
			err := EngineParams.Set(nm, d.Init)
			LogIfError(err)
		}
	}
	bytes, err := TheHost.GetSavedData("engine", fname)
	if err != nil {
		TheHost.LogError(err)
		return
	}
	paramsmap, err := LoadParamsMap(bytes)
	if err != nil {
		TheHost.LogError(err)
		return
	}
	EngineParams.ApplyValuesFromMap("engine", paramsmap, EngineParams.Set)
}

func ScheduleCursorEvent(ce CursorEvent) {

	// schedule CursorEvents rather than handle them right away.
	// This makes it easier to do looping, among other benefits.
	LogOfType("scheduler", "ScheduleCursorEvent", "ce", ce)

	ScheduleAt(CurrentClick(), ce.Tag, ce)
}

func HandleMidiEvent(me MidiEvent) {
	TheHost.SendToOscClients(MidiToOscMsg(me))

	if Midithru {
		LogOfType("midi", "PassThruMIDI", "msg", me.Msg)
		if MidiThruScadjust {
			LogWarn("PassThruMIDI, midiThruScadjust needs work", "msg", me.Msg)
		}
		ScheduleAt(CurrentClick(), "midi", me.Msg)
	}
	if Midisetexternalscale {
		handleMIDISetScaleNote(me)
	}

	err := TheQuadPro.onMidiEvent(me)
	LogIfError(err)

	if TheErae.enabled {
		TheErae.onMidiEvent(me)
	}
}

func handleMIDISetScaleNote(me MidiEvent) {
	if !me.HasPitch() {
		return
	}
	pitch := me.Pitch()
	if me.Msg.Is(midi.NoteOnMsg) {

		if MidiNumDown < 0 {
			// may happen when there's a Read error that misses a noteon
			MidiNumDown = 0
		}

		// If there are no notes held down (i.e. this is the first), clear the scale
		if MidiNumDown == 0 {
			LogOfType("scale", "Clearing external scale")
			ClearExternalScale()
		}
		SetExternalScale(pitch, true)
		MidiNumDown++

		// adjust the octave shift based on incoming MIDI
		newshift := 0
		if pitch < 60 {
			newshift = -1
		} else if pitch > 72 {
			newshift = 1
		}
		if newshift != MidiOctaveShift {
			MidiOctaveShift = newshift
			LogOfType("midi", "MidiOctaveShift changed", "octaveshift", MidiOctaveShift)
		}
	} else if me.Msg.Is(midi.NoteOffMsg) {
		MidiNumDown--
	}
}

func GoroutineID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

func JsonObject(args ...string) string {
	s := JsonString(args...)
	return "{ " + s + " }"
}

func JsonString(args ...string) string {
	if len(args)%2 != 0 {
		LogWarn("ApiParams: odd number of arguments", "args", args)
		return ""
	}
	params := ""
	sep := ""
	for n := range args {
		if n%2 == 0 {
			params = params + sep + "\"" + args[n] + "\": \"" + args[n+1] + "\""
		}
		sep = ", "
	}
	return params
}

func BoundValueZeroToOne(v float64) float64 {
	if v < 0.0 {
		return 0.0
	}
	if v > 1.0 {
		return 1.0
	}
	return v
}

func GetNameValue(apiargs map[string]string) (name string, value string, err error) {
	name, ok := apiargs["name"]
	if !ok {
		err = fmt.Errorf("missing name argument")
		return
	}
	value, ok = apiargs["value"]
	if !ok {
		err = fmt.Errorf("missing value argument")
		return
	}
	return
}

var FirstTime = true
var Time0 = time.Time{}

// Uptime returns the number of seconds since the program started.
func Uptime() float64 {
	now := time.Now()
	if FirstTime {
		FirstTime = false
		Time0 = now
	}
	return now.Sub(Time0).Seconds()
}
