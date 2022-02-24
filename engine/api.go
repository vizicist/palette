package engine

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// ExecuteAPI xxx
func (r *Router) ExecuteAPI(api string, nuid string, rawargs string) (result interface{}, err error) {

	apiargs, e := StringMap(rawargs)
	if e != nil {
		result = ErrorResponse(fmt.Errorf("Router.ExecuteAPI: Unable to interpret value - %s", rawargs))
		log.Printf("Router.ExecuteAPI: bad rawargs value = %s\n", rawargs)
		return result, e
	}

	result = "" // pre-populate most common result

	words := strings.SplitN(api, ".", 2)
	if len(words) != 2 {
		return nil, fmt.Errorf("Router.ExecuteAPI: api=%s is badly formatted, needs a dot", api)
	}
	apiprefix := words[0]
	apisuffix := words[1]

	/////////////////////// region.* APIs

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

	/////////////////////// process.* APIs

	if apiprefix == "process" {

		switch apisuffix {

		case "start":
			process, ok := apiargs["process"]
			if !ok {
				err = fmt.Errorf("ExecuteAPI: missing process argument")
			} else {
				err = StartRunning(process)
			}

		case "stop":
			process, ok := apiargs["process"]
			if !ok {
				err = fmt.Errorf("ExecuteAPI: missing process argument")
			} else {
				err = StopRunning(process)
			}

		case "activate":
			result, err = executeAPIActivate()

		default:
			err = fmt.Errorf("ExecuteAPI: unknown api %s", api)
		}

		if err != nil {
			return nil, err
		} else {
			return result, nil
		}
	}

	// Everything else should be "global", eventually I'll factor this
	if apiprefix != "global" {
		return nil, fmt.Errorf("ExecuteAPI: api=%s unknown apiprefix=%s", api, apiprefix)
	}

	/////////////////////// global.* APIs

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
				motor.setOneParamValue("misc.scale", v)
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
