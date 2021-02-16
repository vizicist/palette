package isf

import (
	"fmt"
	"log"
	"time"

	"github.com/hypebeast/go-osc/osc"
	"github.com/vizicist/palette/engine"
)

// DebugISF xxx
var DebugISF = true

// DebugShader xxx
var DebugShader = false

type apiEvent struct {
	method string
	args   string
}

// Realtime takes events and routes them
type Realtime struct {
	apiInput chan apiEvent
	config   map[string]string

	nowMilli       int // the time from the start, in milliseconds
	nowMilliOffset int

	time         time.Time
	time0        time.Time
	globalParams *engine.ParamValues
}

// PlaybackEvent is a time-tagged cursor or API event
type PlaybackEvent struct {
	time      float32
	eventType string
	pad       string
	method    string
	args      map[string]string
	rawargs   string
}

// NewRealtime xxx
func NewRealtime() (*Realtime, error) {

	/*
		config, err := engine.ReadConfigFile(engine.ConfigFilePath("isf.json"))
		if err != nil {
			return nil, fmt.Errorf("NewRealtime: err=%s", err)
		}
	*/

	engine.LoadParamEnums()
	engine.LoadParamDefs()
	// engine.LoadEffectsJSON()

	e := &Realtime{
		globalParams: engine.NewParamValues(),
		apiInput:     make(chan apiEvent),
	}

	return e, nil
}

// Start starts any extra executable/cmdline that's needed, and then runs the looper and never returns
func (e *Realtime) Start() {

	go e.startOSCListener("127.0.0.1:3334")

	tick := time.NewTicker(2 * time.Millisecond)
	e.time0 = <-tick.C

	log.Printf("Realtime.Start: started\n")

	// By reading from tick.C, we wake up every so many milliseconds
	for now := range tick.C {

		e.time = now
		sofar := now.Sub(e.time0)
		secs := sofar.Seconds()
		e.nowMilli = int(secs * 1000.0)

		// Sprites should get aged and potentially killed here
		// if reactor.targetVizlet != nil {
		// 	reactor.targetVizlet.AgeSprites()
		// }
		// Parameters gets updated here
		// reactor.advanceStepLooperByOneStep()
		// reactor.activePhrasesManager.AdvanceActivePhrasesByOneStep()

		select {
		case api := <-e.apiInput:
			_, err := e.executeAPI(api)
			if err != nil {
				log.Printf("Realtime: api err=%s\n", err)
			}
		default:
			// do nothing, so that apiInput is non-blocking
		}
	}
	log.Printf("Realtime.Start: ends unexpectedly?\n")
}

// StartOSCListener xxx
func (e *Realtime) startOSCListener(source string) {

	d := osc.NewStandardDispatcher()

	err := d.AddMsgHandler("*", e.handleOSCMessage)
	if err != nil {
		log.Printf("Realtime.startOSCListener: err=%s\n", err)
	}

	server := &osc.Server{
		Addr:       source,
		Dispatcher: d,
	}
	server.ListenAndServe()
}

func (e *Realtime) handleOSCMessage(msg *osc.Message) {

	if msg.Address != "/api" {
		log.Printf("Realtime: Can't handle addr=%s\n", msg.Address)
		return
	}
	nargs := msg.CountArguments()
	if nargs < 1 {
		log.Printf("Realtime: not enough arguments in /api message\n")
		return
	}
	// The first argument is the method
	meth, err := engine.ArgAsString(msg, 0)
	if err != nil {
		log.Printf("Realtime.handleOSCMessage: err=%s\n", err)
		return
	}
	args := ""
	if nargs > 1 {
		args, err = engine.ArgAsString(msg, 1)
		if err != nil {
			log.Printf("Realtime.handleOSCMessage: err=%s\n", err)
			return
		}
	}
	e.apiInput <- apiEvent{method: meth, args: args}
}

func (e *Realtime) executeAPI(api apiEvent) (result string, err error) {

	method := api.method
	var args map[string]string
	if api.args == "" {
		args = map[string]string{}
	} else {
		args, err = engine.StringMap(api.args)
		if err != nil {
			return "", err
		}
	}

	globalParams := e.globalParams

	result = "0" // most APIs just return 0, so pre-populate it

	switch method {

	case "set_param":
		nm := args["param"]
		val := args["value"]
		err := globalParams.SetParamValueWithString(nm, val, nil)
		if err != nil {
			return "", err
		}

	case "set_params":

		log.Printf("set_params API in executeGlobalAPI - needed?\n")
		for name, value := range args {
			err := globalParams.SetParamValueWithString(name, value, nil)
			if err != nil {
				return "", fmt.Errorf("error in SoundAPI: method=sound.set_params name=%s value=%s err=%s", name, value, err)
			}
		}

	case "sprite":
		log.Printf("Adding sprite\n")

		x, y, z, err := engine.GetXYZ(method, args)
		if err != nil {
			return "", err
		}
		log.Printf("Realtime: Should be adding sprite at %f,%f,%f\n", x, y, z)
		ss := NewSpriteSquare(x, y, z, globalParams)
		_ = ss
		// reactor.AddSprite(ss)

	default:
		err = fmt.Errorf("Realtime.executeAPI: unrecognized meth=%v", method)
		result = ""
	}

	return result, err
}

// Time returns the current time
func (e *Realtime) Time() time.Time {
	return time.Now()
}
