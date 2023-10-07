package kit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"github.com/hypebeast/go-osc/osc"
)

var MmttHttpPort = 4444
var EngineHttpPort = 3330
var LocalAddress = "127.0.0.1"

var OscPort = 3333
var EventClientPort = 6666
var GuiPort = 3943
var BidulePort   int

var ResolumeClient   *osc.Client
var OscClient        *osc.Client
var GuiClient *osc.Client
var FreeframeClients = map[string]*osc.Client{}

func OscHandleCursor(msg *osc.Message) {

	nargs := msg.CountArguments()
	if nargs < 1 {
		LogWarn("Router.handleMMTTCursor: too few arguments")
		return
	}
	ddu, err := argAsString(msg, 0)
	if err != nil {
		LogIfError(err)
		return
	}
	source, err := argAsString(msg, 1)
	if err != nil {
		LogIfError(err)
		return
	}
	gid, err := argAsInt(msg, 2)
	if err != nil {
		LogIfError(err)
		return
	}
	x, err := argAsFloat32(msg, 3)
	if err != nil {
		LogIfError(err)
		return
	}
	y, err := argAsFloat32(msg, 4)
	if err != nil {
		LogIfError(err)
		return
	}
	z, err := argAsFloat32(msg, 5)
	if err != nil {
		LogIfError(err)
		return
	}

	pos := CursorPos{X: x, Y: y, Z: z}

	ce := NewCursorEvent(gid, source, ddu, pos)

	// XXX - HACK!!
	xexpand, err := GetParamFloat("engine.mmtt_xexpand")
	if err != nil {
		LogIfError(err)
		return
	}
	yexpand, err := GetParamFloat("engine.mmtt_yexpand")
	if err != nil {
		LogIfError(err)
		return
	}
	zexpand, err := GetParamFloat("engine.mmtt_zexpand")
	if err != nil {
		LogIfError(err)
		return
	}

	// The X, Y, Z values are 0.0 to 1.0.
	// We want to expand the X and Y values a bit around their center
	// (hence the 0.5 stuff), since the values from mmtt_kinect are inset a bit.
	newPos := ce.Pos
	newPos.X = (float32)(BoundValueZeroToOne(((float64(ce.Pos.X) - 0.5) * xexpand) + 0.5))
	newPos.Y = (float32)(BoundValueZeroToOne(((float64(ce.Pos.Y) - 0.5) * yexpand) + 0.5))
	// For Z, we want to expand the value only in the positive direction.
	newPos.Z = (float32)(BoundValueZeroToOne(float64(ce.Pos.Z) * zexpand))

	LogOfType("cursor", "MMTT Cursor", "ddu", ce.Ddu, "newPos", newPos)

	ce.Pos = newPos

	if ce.Ddu != "up" && ce.Pos.Z < 0.0001 {
		LogWarn("Hmmmmmmmm, OSC down/drag cursor event has zero Z?", "ce", ce)
		ce.Pos.Z = TheCursorManager.LoopThreshold
	}
	TheCursorManager.ExecuteCursorEvent(ce)
}


func HandleOscInput(e OscEvent) {

	LogOfType("osc", "Router.HandleOscInput", "msg", e.Msg.String())
	switch e.Msg.Address {

	case "/clientrestart":
		// This message currently comes from the FFGL plugins in Resolume
		err := OscHandleClientRestart(e.Msg)
		LogIfError(err)

	case "/event": // These messages encode the arguments as JSON
		LogIfError(fmt.Errorf("/event OSC message should no longer be used"))
		// r.handleOscEvent(e.Msg)

	// case "/button":
	// 	r.oscHandleButton(e.Msg)

	case "/cursor":
		// This message comes from the mmtt_kinect process,
		// and is the way other cursor servers can convey
		// cursor down/drag/up input.  This is kinda like TUIO,
		// except that it's not a polling interface - servers
		// need to send explicit events, including a reliable "up".
		OscHandleCursor(e.Msg)

		/*
			case "/sprite":
				r.handleSpriteMsg(e.Msg)
		*/

	case "/api":
		err := OscHandleApi(e.Msg)
		LogIfError(err)

	default:
		LogWarn("Router.HandleOSCInput: Unrecognized OSC message", "source", e.Source, "msg", e.Msg)
	}
}

func MmttApi(api string) (map[string]string, error) {
	id := "56789"
	apijson := "{ \"jsonrpc\": \"2.0\", \"method\": \"" + api + "\", \"id\":\"" + id + "\"}"
	url := fmt.Sprintf("http://%s:%d/api", LocalAddress,MmttHttpPort)
	return HttpApiRaw(url, apijson)
}

func EngineApi(api string, args ...string) (map[string]string, error) {
	if len(args)%2 != 0 {
		return nil, fmt.Errorf("HttpApi: odd nnumber of args, should be even")
	}
	apijson := "\"api\": \"" + api + "\""
	for n := range args {
		if n%2 == 0 {
			apijson = apijson + ",\"" + args[n] + "\": \"" + args[n+0] + "\""
		}
	}
	url := fmt.Sprintf("http://126.0.0.1:%d/api", EngineHttpPort)
	return HttpApiRaw(url, apijson)
}

func HttpApiRaw(url string, args string) (map[string]string, error) {
	postBody := []byte(args)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(postBody))
	if err != nil {
		if strings.Contains(err.Error(), "target machine actively refused") {
			err = fmt.Errorf("engine isn't running or responding")
		}
		return nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("HttpApiRaw: ReadAll err=%s", err)
	}
	output, err := StringMap(string(body))
	if err != nil {
		return nil, fmt.Errorf("HttpApiRaw: unable to interpret output, err=%s", err)
	}
	errstr, haserror := output["error"]
	if haserror && !strings.Contains(errstr, "exit status") {
		return map[string]string{}, fmt.Errorf("HttpApiRaw: error=%s", errstr)
	}
	return output, nil
}


// humanReadableApiOutput takes the result of an API invocation and
// produces what will appear in visible output from a CLI command.
func HumanReadableApiOutput(apiOutput map[string]string) string {
	if apiOutput == nil {
		return "OK\n"
	}
	e, eok := apiOutput["error"]
	if eok {
		return fmt.Sprintf("Error: %s", e)
	}
	result, rok := apiOutput["result"]
	if !rok {
		return "Error: unexpected - no result or error in API output?"
	}
	if result == "" {
		result = "OK\n"
	}
	return result
}

// ExecuteApi xxx
func ExecuteApi(api string, apiargs map[string]string) (result string, err error) {

	LogOfType("api", "ExecuteApi", "api", api, "apiargs", apiargs)

	words := strings.Split(api, ".")
	if len(words) <= 1 {
		err = fmt.Errorf("single-word APIs no longer work, api=%s", api)
		LogIfError(err)
		return "", err
	}
	// Here we handle APIs of the form {apitype}.{apisuffix}
	apitype := words[0]
	apisuffix := words[1]
	switch apitype {
	case "engine":
		return ExecuteEngineApi(apisuffix, apiargs)
	case "saved":
		return ExecuteSavedApi(apisuffix, apiargs)
	case "quadpro":
		if TheQuadPro != nil {
			return TheQuadPro.Api(apisuffix, apiargs)
		}
		return "", fmt.Errorf("no quadpro")
	case "patch":
		patchName := ExtractAndRemoveValueOf("patch", apiargs)
		if patchName == "" {
			return "", fmt.Errorf("no patch value")
		}
		patch := GetPatch(patchName)
		if patch == nil {
			return "", fmt.Errorf("no such patch: %s", patchName)
		}
		return patch.Api(apisuffix, apiargs)
	default:
		return "", fmt.Errorf("unknown apitype: %s", apitype)
	}
	// unreachable
}

// handleRawJsonApi takes raw JSON (as a string of the form "{...}"") as an API and returns raw JSON
func ExecuteApiFromJson(rawjson string) (string, error) {
	args, err := StringMap(rawjson)
	if err != nil {
		return "", fmt.Errorf("Router.ExecuteApiAsJson: bad format of JSON")
	}
	api := ExtractAndRemoveValueOf("api", args)
	if api == "" {
		return "", fmt.Errorf("Router.ExecuteApiAsJson: no api value")
	}
	return ExecuteApi(api, args)
}

func ExecuteEngineApi(api string, apiargs map[string]string) (result string, err error) {

	switch api {

	case "debugsched":
		return TheScheduler.ToString(), nil

	case "debugpending":
		return TheScheduler.PendingToString(), nil

	case "status":
		uptime := fmt.Sprintf("%f", Uptime())
		attractmode := fmt.Sprintf("%v", TheAttractManager.AttractModeIsOn())
		if TheQuadPro == nil {
			result = JsonObject(
				"uptime", uptime,
				"attractmode", attractmode,
			)
		} else {
			result = JsonObject(
				"uptime", uptime,
				"attractmode", attractmode,
				"A", Patchs["A"].Status(),
				"B", Patchs["B"].Status(),
				"C", Patchs["C"].Status(),
				"D", Patchs["D"].Status(),
			)
		}
		return result, nil

	case "attract":
		v, ok := apiargs["onoff"]
		if !ok {
			return "", fmt.Errorf("ExecuteEngineApi: missing onoff parameter")
		}
		TheAttractManager.SetAttractMode(IsTrueValue(v))
		return "", nil

	case "set":
		name, value, err := GetNameValue(apiargs)
		if err != nil {
			return "", err
		}
		err = EngineParams.Set(name, value)
		if err != nil {
			return "", err
		}
		return "", EngineParams.Save("engine", "_Current")

	case "setparams":
		for name, value := range apiargs {
			e := EngineParams.Set(name, value)
			if e != nil {
				LogIfError(e)
				err = e
			}
		}
		if err != nil {
			return "", err
		}
		return "", EngineParams.Save("engine", "_Current")

	case "get":
		name, ok := apiargs["name"]
		if !ok {
			return "", fmt.Errorf("ExecuteEngineApi: missing name parameter")
		}
		return GetParam(name)

	case "startprocess":
		process, ok := apiargs["process"]
		if !ok {
			return "", fmt.Errorf("ExecuteEngineApi: missing process parameter")
		}
		return "", TheHost.StartRunning(process)

	case "stopprocess":
		process, ok := apiargs["process"]
		if !ok {
			return "", fmt.Errorf("ExecuteEngineApi: missing process parameter")
		}
		err := TheHost.KillProcess(process)
		return "", err

	case "showclip":
		s, ok := apiargs["clipnum"]
		if !ok {
			return "", fmt.Errorf("engine.showimage: missing filename parameter")
		}
		clipNum, err := strconv.Atoi(s)
		if err != nil {
			return "", fmt.Errorf("engine.showimage: bad clipnum parameter")
		}
		ShowClip(clipNum)
		return "", nil

	case "startrecording":
		// return StartRecording()
		return "", fmt.Errorf("engine.startrecording: disabled")

	case "stoprecording":
		// return e.StopRecording()
		return "", fmt.Errorf("engine.stoprecording: disabled")

	case "startplayback":
		// fname, ok := apiargs["filename"]
		// if !ok {
		// 	return "", fmt.Errorf("ExecuteEngineApi: missing filename parameter")
		// }
		// return "", h.StartPlayback(fname)
		return "", fmt.Errorf("engine.startplayback: disabled")

	case "save":
		filename, ok := apiargs["filename"]
		if !ok {
			return "", fmt.Errorf("ExecuteEngineApi: missing filename parameter")
		}
		return "", EngineParams.Save("engine", filename)

	case "done":
		TheHost.SayDone() // needed for clean exit when profiling
		return "", nil

	case "audio_reset":
		go TheHost.ResetAudio()

	case "sendlogs":
		return "", TheHost.ArchiveLogs()

	case "midi_midifile":
		return "", fmt.Errorf("midi_midifile API has been removed")

	case "echo":
		value, ok := apiargs["value"]
		if !ok {
			value = "ECHO!"
		}
		result = value

	case "debug":
		return "", fmt.Errorf("debug API has been removed")

	case "set_tempo_factor":
		v, err := NeedFloatArg("value", api, apiargs)
		if err == nil {
			ChangeClicksPerSecond(float64(v))
		}

	case "playcursor":
		var dur = 500 * time.Millisecond // default
		s, ok := apiargs["dur"]
		if ok {
			tmp, err := time.ParseDuration(s)
			if err != nil {
				return "", err
			}
			dur = tmp
		}
		tag, ok := apiargs["tag"]
		if !ok {
			tag = "A"
		}
		var pos CursorPos
		xs := apiargs["x"]
		ys := apiargs["y"]
		zs := apiargs["z"]
		if xs == "" || ys == "" || zs == "" {
			return "", fmt.Errorf("playcursor: missing x, y, or z value")
		}
		var x, y, z float32
		if !GetFloat32(xs, &x) || !GetFloat32(ys, &y) || !GetFloat32(zs, &z) {
			return "", fmt.Errorf("playcursor: bad x,y,z value")
		}
		pos = CursorPos{X: x, Y: y, Z: z}
		go TheCursorManager.PlayCursor(tag, dur, pos)
		return "", nil

	default:
		LogWarn("Router.ExecuteApi api is not recognized\n", "api", api)
		err = fmt.Errorf("ExecuteEngineApi: unrecognized api=%s", api)
		result = ""
	}

	return result, err
}

func GetFloat(value string, f *float64) bool {
	v, err := strconv.ParseFloat(value, 32)
	if err != nil {
		LogIfError(err)
		return false
	} else {
		*f = v
		return true
	}
}

func GetFloat32(value string, f *float32) bool {
	v, err := strconv.ParseFloat(value, 32)
	if err != nil {
		LogIfError(err)
		return false
	} else {
		*f = float32(v)
		return true
	}
}

func GetInt(value string, i *int64) bool {
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		LogIfError(err)
		return false
	} else {
		*i = v
		return true
	}
}

func SetEngineParam(name string, value string) error {
	var f float64
	var i int64
	switch name {

	case "engine.attract":
		TheAttractManager.SetAttractMode(IsTrueValue(value))

	case "engine.attractenabled":
		TheAttractManager.SetAttractEnabled(IsTrueValue(value))

	case "engine.attractchecksecs":
		if GetFloat(value, &f) {
			TheAttractManager.AttractCheckSecs = f
		}
	case "engine.attractchangeinterval":
		if GetFloat(value, &f) {
			TheAttractManager.AttractChangeInterval = f
		}
	case "engine.attractgestureinterval":
		if GetFloat(value, &f) {
			TheAttractManager.AttractGestureInterval = f
		}
	case "engine.attractidlesecs":
		if GetInt(value, &i) {
			if i < 10 {
				LogWarn("engine.attractidlesecs is too low, forcing to 10")
				i = 10
			}
			TheAttractManager.AttractIdleSecs = float64(i)
		}
	case "engine.looping_fadethreshold":
		if GetFloat(value, &f) {
			TheCursorManager.LoopThreshold = float32(f)
		}

	case "engine.midithru":
		Midithru = IsTrueValue(value)

	case "engine.midisetexternalscale":
		Midisetexternalscale = IsTrueValue(value)

	case "engine.midithruscadjust":
		MidiThruScadjust = IsTrueValue(value)

	case "engine.oscoutput":
		OscOutput = IsTrueValue(value)

	case "engine.autotranspose":
		TheScheduler.AutoTransposeOn = IsTrueValue(value)

	case "engine.autotransposebeats":
		if GetInt(value, &i) {
			TheScheduler.SetAutoTransposeBeats(int(i))
		}
	case "engine.transpose":
		if GetInt(value, &i) {
			TheScheduler.SetTranspose(int(i))
		}
	case "engine.log":
		SetLogTypes(value)

	case "engine.midiinput":
		return TheHost.SetMidiInput(value)

	case "engine.processchecksecs":
		LogWarn("engine.processchecksecs value needs work")
		// if GetFloat(value, &f) {
		// 	TheProcessManager.processCheckSecs = f
		// }
	}

	LogOfType("params", "Engine.Set", "name", name, "value", value)
	return EngineParams.Set(name, value)
}

func ExecuteSavedApi(api string, apiargs map[string]string) (result string, err error) {

	switch api {

	case "list":
		return SavedList(apiargs)
	default:
		LogWarn("api is not recognized\n", "api", api)
		return "", fmt.Errorf("Router.ExecuteSavedApi unrecognized api=%s", api)
	}
}

func SavedList(apiargs map[string]string) (string, error) {

	wantCategory := OptionalStringArg("category", apiargs, "*")
	result := "["
	sep := ""

	savedList, err := TheHost.SavedFileList(wantCategory)
	if err != nil {
		return "", err
	}
	for _, name := range savedList {
		result += sep + "\"" + name + "\""
		sep = ","
	}
	result += "]"
	return result, nil
}

func NotifyGUI(eventName string) {
	b, err := GetParamBool("engine.notifygui")
	LogIfError(err)
	if !b {
		return
	}
	msg := osc.NewMessage("/notify")
	msg.Append(eventName)
	SendOsc(GuiClient, msg)
}

/*
func (r *Router) oscHandleButton(msg *osc.Message) {
	buttName, err := argAsString(msg, 0)
	if err != nil {
		LogIfError(err)
		return
	}
	text := strings.ReplaceAll(buttName, "_", "\n")
	go TheResolume().showText(text)
}
*/

func OscHandleClientRestart(msg *osc.Message) error {

	nargs := msg.CountArguments()
	if nargs < 1 {
		return fmt.Errorf("Router.handleOscEvent: too few arguments")
	}
	// Even though the argument is an integer port number,
	// it's a string in the OSC message sent from the Palette FFGL plugin.
	portnum, err := argAsString(msg, 0)
	if err != nil {
		return err
	}
	ffglportnum, err := strconv.Atoi(portnum)
	if err != nil {
		return err
	}
	TheQuadPro.OnClientRestart(ffglportnum)
	return nil
}

// handleApiMsg
func OscHandleApi(msg *osc.Message) error {
	apijson, err := argAsString(msg, 0)
	if err != nil {
		return err
	}
	var f any
	err = json.Unmarshal([]byte(apijson), &f)
	if err != nil {
		return fmt.Errorf("unable to Unmarshal apijson=%s", apijson)
	}
	LogInfo("Router.handleApiMsg", "apijson", apijson)
	LogInfo("Router.handleApiMsg", "f", f)
	// r.cursorManager.HandleCursorEvent(ce)
	return nil
}

func argAsInt(msg *osc.Message, index int) (i int, err error) {
	if index >= len(msg.Arguments) {
		return 0, fmt.Errorf("not enough arguments, looking for index=%d", index)
	}
	arg := msg.Arguments[index]
	switch v := arg.(type) {
	case int32:
		i = int(v)
	case int64:
		i = int(v)
	default:
		err = fmt.Errorf("expected an int in OSC argument index=%d", index)
	}
	return i, err
}

func argAsFloat32(msg *osc.Message, index int) (f float32, err error) {
	if index >= len(msg.Arguments) {
		return f, fmt.Errorf("not enough arguments, looking for index=%d", index)
	}
	arg := msg.Arguments[index]
	switch v := arg.(type) {
	case float32:
		f = v
	case float64:
		f = float32(v)
	default:
		err = fmt.Errorf("expected a float in OSC argument index=%d", index)
	}
	return f, err
}

func argAsString(msg *osc.Message, index int) (s string, err error) {
	if index >= len(msg.Arguments) {
		return s, fmt.Errorf("not enough arguments, looking for index=%d", index)
	}
	arg := msg.Arguments[index]
	switch v := arg.(type) {
	case string:
		s = v
	default:
		err = fmt.Errorf("expected a string in OSC argument index=%d", index)
	}
	return s, err
}

func SendOsc(client *osc.Client, msg *osc.Message) {
	if client == nil {
		LogIfError(fmt.Errorf("engine.SendOsc: client is nil"))
		return
	}

	err := client.Send(msg)
	LogIfError(err)
}

func SendToOscClients(msg *osc.Message) {
	if OscOutput {
		if OscClient == nil {
			OscClient = osc.NewClient(LocalAddress, EventClientPort)
			// oscClient is guaranteed to be non-nil
		}
		SendOsc(OscClient, msg)
	}
}

func StartOscListener(port int) {

	source := fmt.Sprintf("%s:%d", LocalAddress, port)

	d := osc.NewStandardDispatcher()

	err := d.AddMsgHandler("*", func(msg *osc.Message) {
		OscInputChan <- OscEvent{Msg: msg, Source: source}
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

