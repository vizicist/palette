package engine

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hypebeast/go-osc/osc"
)

type RunningCmd struct {
	cmd *exec.Cmd
}

var RunningCmds = map[string]RunningCmd{}

func activateLater(dur time.Duration, ntimes int) {
	for i := 0; i < ntimes; i++ {
		time.Sleep(dur)
		_, err := executeAPIActivate()
		if err != nil {
			log.Printf("activateLater: err=%s\n", err)
		}
	}
}

func StartRunning(process string) error {

	log.Printf("StartRunning: process %s\n", process)

	_, found := RunningCmds[process]
	if found {
		return fmt.Errorf("executeAPIStart: process %s already running", process)
	}
	switch process {

	case "all":
		err := StartRunning("resolume")
		if err != nil {
			return fmt.Errorf("start: resolume err=%s", err)
		}
		err = StartRunning("gui")
		if err != nil {
			return fmt.Errorf("start: gui err=%s", err)
		}
		err = StartRunning("bidule")
		if err != nil {
			return fmt.Errorf("start: bidule err=%s", err)
		}
		// Always send logs after starting all
		SendLogs()

	case "gui":
		fullexe := filepath.Join(PaletteDir(), "bin", "pyinstalled", "palette_gui.exe")
		cmd, err := StartExecutableRedirectOutput(process, fullexe, true, "")
		if err != nil {
			return fmt.Errorf("start: err=%s", err)
		}
		RunningCmds[process] = RunningCmd{cmd: cmd}

	case "bidule":
		fullexe := ConfigValue("bidule")
		if fullexe == "" {
			fullexe = "C:\\Program Files\\Plogue\\Bidule\\PlogueBidule_X64.exe"
		}
		exearg := ConfigFilePath("palette.bidule")
		if !FileExists(fullexe) {
			return fmt.Errorf("no Bidule found, looking for %s", fullexe)
		}
		cmd, err := StartExecutableRedirectOutput("bidule", fullexe, true, exearg)
		if err != nil {
			return fmt.Errorf("start: bidule err=%s", err)
		}
		RunningCmds[process] = RunningCmd{cmd: cmd}
		// bidule can take forever to load, especially on non-SSD
		go activateLater(10*time.Second, 20)

	case "resolume":
		fullexe := ConfigValue("resolume")
		if fullexe != "" && !FileExists(fullexe) {
			return fmt.Errorf("no Resolume found, looking for %s", fullexe)
		}
		if fullexe == "" {
			fullexe = "C:\\Program Files\\Resolume Avenue\\Avenue.exe"
			if !FileExists(fullexe) {
				fullexe = "C:\\Program Files\\Resolume Arena\\Arena.exe"
				if !FileExists(fullexe) {
					return fmt.Errorf("no Resolume found in default locations")
				}
			}
		}
		cmd, err := StartExecutableRedirectOutput("resolume", fullexe, true, "")
		if err != nil {
			return fmt.Errorf("start: resolume err=%s", err)
		}
		RunningCmds[process] = RunningCmd{cmd: cmd}
		go activateLater(8*time.Second, 2)

	default:
		return fmt.Errorf("start: unrecognized process %s", process)
	}

	return nil
}

func stopRunning(process string) (string, error) {
	cmd, found := RunningCmds[process]
	if !found {
		return "", fmt.Errorf("stop: process %s not running", process)
	}
	p := cmd.cmd.Process
	defer delete(RunningCmds, process)
	if p == nil {
		return "", fmt.Errorf("stop: process %s not Running", process)
	}
	err := p.Kill()
	if err != nil {
		return "", fmt.Errorf("stop: Kill of %s err=%s", process, err)
	}
	return "0", nil
}

func executeAPIActivate() (string, error) {
	// handle_activate sends OSC messages to start the layers in Resolume,
	// and make sure the audio is on in Bidule.
	addr := "127.0.0.1"
	resolumePort := 7000
	bidulePort := 3210

	resolumeClient := osc.NewClient(addr, resolumePort)
	start_layer(resolumeClient, 1)
	start_layer(resolumeClient, 2)
	start_layer(resolumeClient, 3)
	start_layer(resolumeClient, 4)

	biduleClient := osc.NewClient(addr, bidulePort)
	msg := osc.NewMessage("/play")
	msg.Append(int32(1)) // turn it on
	// log.Printf("Sending %s\n", msg.String())
	err := biduleClient.Send(msg)
	if err != nil {
		return "", fmt.Errorf("handle_activate: osc to Bidule, err=%s", err)
	}
	return "0", nil
}

func start_layer(resolumeClient *osc.Client, layer int) {
	addr := fmt.Sprintf("/composition/layers/%d/clips/1/connect", layer)
	msg := osc.NewMessage(addr)
	msg.Append(int32(1))
	// log.Printf("Sending %s\n", msg.String())
	err := resolumeClient.Send(msg)
	if err != nil {
		log.Printf("start_layer: osc to Resolume err=%s\n", err)
	}
}

// ExecuteAPI xxx
func (r *Router) ExecuteAPI(api string, nuid string, rawargs string) (result interface{}, err error) {

	apiargs, err := StringMap(rawargs)
	if err != nil {
		response := ErrorResponse(fmt.Errorf("Router.ExecuteAPI: Unable to interpret value - %s", rawargs))
		log.Printf("Router.ExecuteAPI: bad rawargs value = %s\n", rawargs)
		return response, nil
	}

	result = "0" // most APIs just return 0, so pre-populate it

	words := strings.SplitN(api, ".", 2)
	if len(words) != 2 {
		return nil, fmt.Errorf("Router.ExecuteAPI: api=%s is badly formatted, needs a dot", api)
	}
	apiprefix := words[0]
	apisuffix := words[1]

	if apiprefix == "region" {

		region := optionalStringArg("region", apiargs, "")
		if region == "" {
			region = r.getRegionForNUID(nuid)
		} else {
			// Remove it from the args given to ExecuteAPI
			delete(apiargs, "region")
		}
		motor, ok := r.motors[region]
		if !ok {
			return nil, fmt.Errorf("api/event=%s there is no region named %s", api, region)
		}
		return motor.ExecuteAPI(apisuffix, apiargs, rawargs)
	}

	if apiprefix == "process" {
		switch apisuffix {

		case "start":
			process, ok := apiargs["process"]
			if !ok {
				process = "all"
			}
			return nil, StartRunning(process)

		case "stop":
			process, ok := apiargs["process"]
			if !ok {
				process = "all"
			}
			return stopRunning(process)

		case "activate":
			return executeAPIActivate()

		default:
			return nil, fmt.Errorf("ExecuteAPI: unknown api %s", api)
		}
	}

	// Everything else should be "global", eventually I'll factor this
	if apiprefix != "global" {
		return nil, fmt.Errorf("ExecuteAPI: api=%s unknown apiprefix=%s", api, apiprefix)
	}

	switch apisuffix {

	case "sendlogs":
		SendLogs()

	case "midi_midifile":
		return nil, fmt.Errorf("midi_midifile API has been removed")

	case "echo":
		value, ok := apiargs["value"]
		if !ok {
			value = "ECHO!"
		}
		result = value

	case "fakemidi":
		// publish fake event for testing
		me := MIDIDeviceEvent{
			Timestamp: int64(0),
			Status:    0x90,
			Data1:     0x10,
			Data2:     0x10,
		}
		eee := PublishMIDIDeviceEvent(me)
		if eee != nil {
			log.Printf("InputListener: me=%+v err=%s\n", me, eee)
		}
		/*
			d := time.Duration(secs) * time.Second
			log.Printf("hang: d=%+v\n", d)
			time.Sleep(d)
			log.Printf("hang is done\n")
		*/

	case "debug":
		s, err := needStringArg("debug", api, apiargs)
		if err == nil {
			b, err := needBoolArg("onoff", api, apiargs)
			if err == nil {
				setDebug(s, b)
			}
		}

	case "set_tempo_factor":
		v, err := needFloatArg("value", api, apiargs)
		if err == nil {
			ChangeClicksPerSecond(float64(v))
		}

	case "set_transpose":
		v, err := needFloatArg("value", api, apiargs)
		if err == nil {
			for _, motor := range r.motors {
				motor.TransposePitch = int(v)
			}
		}

	case "set_transposeauto":
		b, err := needBoolArg("onoff", api, apiargs)
		if err == nil {
			r.transposeAuto = b
			// Quantizing CurrentClick() to a beat or measure might be nice
			r.transposeNext = CurrentClick() + r.transposeBeats*oneBeat
			for _, motor := range r.motors {
				motor.TransposePitch = 0
			}
		}

	case "set_scale":
		v, err := needStringArg("value", api, apiargs)
		if err == nil {
			for _, motor := range r.motors {
				motor.setOneParamValue("misc.", "scale", v)
			}
		}

	case "audio_reset":
		go r.audioReset()

	case "recordingStart":
		r.recordingOn = true
		if r.recordingFile != nil {
			log.Printf("Hey, recordingFile wasn't nil?\n")
			r.recordingFile.Close()
		}
		r.recordingFile, err = os.Create(recordingsFile("LastRecording.json"))
		if err != nil {
			return nil, err
		}
		r.recordingBegun = time.Now()
		if r.recordingOn {
			r.recordEvent("global", "*", "start", "{}")
		}
	case "recordingSave":
		var name string
		name, err = needStringArg("name", api, apiargs)
		if err == nil {
			err = r.recordingSave(name)
		}

	case "recordingStop":
		if r.recordingOn {
			r.recordEvent("global", "*", "stop", "{}")
		}
		if r.recordingFile != nil {
			r.recordingFile.Close()
			r.recordingFile = nil
		}
		r.recordingOn = false

	case "recordingPlay":
		name, err := needStringArg("name", api, apiargs)
		if err == nil {
			events, err := r.recordingLoad(name)
			if err == nil {
				r.sendANO()
				go r.recordingPlayback(events)
			}
		}

	case "recordingPlaybackStop":
		r.recordingPlaybackStop()

	default:
		log.Printf("Router.ExecuteAPI api=%s is not recognized\n", api)
		err = fmt.Errorf("Router.ExecuteAPI unrecognized api=%s", api)
		result = ""
	}

	return result, err
}
