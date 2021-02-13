package isf

import (
	"fmt"
	"log"
	"time"

	"github.com/hypebeast/go-osc/osc"
	"github.com/nats-io/nats.go"
	"github.com/vizicist/palette/engine"
)

// DebugVenue controls debugging output
var DebugVenue = engine.DebugFlags{
	// MIDI:   true,
	// Cursor: true,
}

// OSCEvent is an OSC message
type OSCEvent struct {
	Msg    *osc.Message
	Source string
}

// Command is sent on the control channel of the Venue
type Command struct {
	Action string // e.g. "addmidi"
	Arg    interface{}
}

// Venue takes events and routes them
type Venue struct {
	config map[string]string
	// hubRemote *engine.Remote

	inputs          []*osc.Client
	MIDINumDown     int
	MIDIOctaveShift int
	killme          bool // set to true if Venue should be stopped
	lastClick       engine.Clicks

	defaultClicksPerSecond engine.Clicks
	clicksPerSecond        engine.Clicks
	oneBeat                engine.Clicks

	nowMilli       int // the time from the start, in milliseconds
	nowMilliOffset int
	nowClickOffset engine.Clicks
	nowClick       engine.Clicks

	control      chan Command
	time         time.Time
	time0        time.Time
	killPlayback bool
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

// NewVenue returns a pointer to the one-and-only Venue
func NewVenue(venueName string) (*Venue, error) {

	config, err := engine.ReadConfigFile(engine.ConfigFilePath("venue.json"))
	if err != nil {
		return nil, fmt.Errorf("NewVenue: err=%s", err)
	}

	engine.LoadParamEnums()
	engine.LoadParamDefs()
	// engine.LoadEffectsJSON()

	e := &Venue{
		config:                 config,
		defaultClicksPerSecond: engine.Clicks(192),
		globalParams:           engine.NewParamValues(),
	}

	// engine.SubscribeToMidi(e.midiHandler)
	// engine.SubscribeToCursor(e.cursorHandler)

	return e, nil
}

// ConfigValue xxx
func (e *Venue) ConfigValue(name string) (value string, err error) {
	value, ok := e.config[name]
	if !ok {
		return "", fmt.Errorf("Venue.ConfigValue: no value for %s", name)
	}
	return value, err
}

// This is a callback from NATS
func (e *Venue) midiHandler(msg *nats.Msg) {
	s := string(msg.Data)
	reply := msg.Reply
	subj := msg.Subject
	sub := msg.Sub.Subject
	if DebugVenue.MIDI {
		log.Printf("Venue.midiHandler: data=%s reply=%s subj=%s sub=%s\n",
			s, reply, subj, sub)
	}
}

/*
// This is a callback from NATS
func (e *Venue) cursorHandler(msg *nats.Msg) {
	reply := msg.Reply
	subj := msg.Subject
	sub := msg.Sub.Subject
	log.Printf("Venue.cursorHandler: reply=%s subj=%s sub=%s\n", reply, subj, sub)
	datastr := string(msg.Data)
	pmap, err := engine.StringMap(datastr)
	if err != nil {
		log.Printf("Venue.cursorHandler, bad datastr=%s\n", datastr)
	}
	id := pmap["id"]
	downdragup := pmap["downdragup"]
	var x, y, z float32
	n, err := fmt.Sscanf(pmap["x"], "%f", &x)
	if n < 1 || err != nil {
		log.Printf("cursorHandler: bad x value, pmap[x]=%s", pmap["x"])
		return
	}
	n, err = fmt.Sscanf(pmap["y"], "%f", &y)
	if n < 1 || err != nil {
		log.Printf("cursorHandler: bad y value, pmap[y]=%s", pmap["y"])
		return
	}
	n, err = fmt.Sscanf(pmap["z"], "%f", &z)
	if n < 1 || err != nil {
		log.Printf("cursorHandler: bad z value, pmap[z]=%s", pmap["z"])
		return
	}
	ce := engine.CursorStepEvent{
		ID:         id,
		X:          x,
		Y:          y,
		Z:          z,
		Downdragup: downdragup,
		Fresh:      true, // IS THIS RIGHT?
		Quantized:  false,
	}
	e.routeCursorStepEventToReactors(ce)
}
*/

// seconds2Clicks converts a Time value (elapsed seconds) to Clicks
func (e *Venue) seconds2Clicks(tm float64) engine.Clicks {
	var c engine.Clicks
	c = e.nowClickOffset + engine.Clicks(0.5+float64(tm*1000-float64(e.nowMilliOffset))*(float64(e.clicksPerSecond)/1000.0))
	return c
}

// TimeString returns time and clicks
func (e *Venue) TimeString() string {
	sofar := e.time.Sub(e.time0)
	click := e.seconds2Clicks(sofar.Seconds())
	return fmt.Sprintf("sofar=%f click=%d", sofar.Seconds(), click)

}

// InitializeClicksPerSecond initializes
func (e *Venue) InitializeClicksPerSecond(clkpersec engine.Clicks) {
	e.clicksPerSecond = clkpersec
	e.nowMilliOffset = 0
	e.nowClickOffset = 0
	e.oneBeat = engine.Clicks(e.clicksPerSecond / 2)
}

// ChangeClicksPerSecond is what you use to change the tempo
func (e *Venue) ChangeClicksPerSecond(clkpersec engine.Clicks) {
	minClicksPerSecond := e.defaultClicksPerSecond / 16
	if clkpersec < minClicksPerSecond {
		clkpersec = minClicksPerSecond
	}
	maxClicksPerSecond := e.defaultClicksPerSecond * 16
	if clkpersec > maxClicksPerSecond {
		clkpersec = maxClicksPerSecond
	}
	e.nowMilliOffset = e.nowMilli
	e.nowClickOffset = e.nowClick
	e.clicksPerSecond = clkpersec
	e.oneBeat = engine.Clicks(e.clicksPerSecond / 2)
}

/*
func (e *Venue) callbackOutput(n *engine.Note) {
	log.Printf("Venue callbackOutput n=%s\n", n)
	reactor := e.Reactors["B"]
	if reactor == nil {
		log.Printf("Venue.callbackOutput - no Reactor for B\n")
		return
	}

	_, frac := math.Modf(float64(n.Clicks) / (8 * float64(e.oneBeat)))
	x := float32(frac)
	y := float32(n.Pitch) / 128.0
	z := rand.Float32() / 10.0 // make smaller
	params := reactor.ParamsOfCategory["visual"]
	ss := NewSpriteSquare(x, y, z, params)
	reactor.AddSprite(ss)
}
*/

// Start starts any extra executable/cmdline that's needed, and then runs the looper and never returns
func (e *Venue) Start() {

	/*
		var lastPrintedClick engine.Clicks

		tick := time.NewTicker(2 * time.Millisecond)
		e.time0 = <-tick.C

		log.Printf("Running.\n")

		// By reading from tick.C, we wake up every so many milliseconds
		for now := range tick.C {

			e.time = now
			sofar := now.Sub(e.time0)
			secs := sofar.Seconds()
			newclick := e.seconds2Clicks(secs)
			e.nowMilli = int(secs * 1000.0)

			if newclick > e.nowClick {
				e.advanceClickTo(e.nowClick)
				e.nowClick = newclick
			}

			var everySoOften = e.oneBeat * 32
			if (e.nowClick%everySoOften) == 0 && e.nowClick != lastPrintedClick {
				if DebugVenue.Realtime {
					log.Printf("e.nowClick=%d  nowSeconds=%d\n", e.nowClick, e.nowMilli/1000)
				}
				lastPrintedClick = e.nowClick
			}

			select {
			case cmd := <-e.control:
				_ = cmd
				fmt.Println("got command on control channel: ", cmd)
			default:
			}
		}
	*/
	log.Printf("Venue.Start: ends unexpectedly\n")
}

/*
func (e *Venue) advanceClickTo(click engine.Clicks) {
	var nclicks = click - e.lastClick
	var clk engine.Clicks
	for ; clk < nclicks; clk++ {
		e.activePhrasesManager.AdvanceActivePhrasesByOneStep()
		for _, reactor := range e.Reactors {
			// Sprites get aged and potentially killed here
			if reactor.targetVizlet != nil {
				reactor.targetVizlet.AgeSprites()
			}
			// Parameters gets updated here
			reactor.advanceStepLooperByOneStep()
			reactor.activePhrasesManager.AdvanceActivePhrasesByOneStep()
		}
	}
	e.lastClick = click
}
*/

// APIHandler is an API execution func
type APIHandler func(method string, pmap map[string]string) (string, error)

func (e *Venue) handlePerPadAPI(apitype string, apifunc APIHandler, pad string, method string, pmap map[string]string, paramsString string) (string, error) {

	_, err := apifunc(method, pmap)
	if err != nil {
		return "", fmt.Errorf("Error in handleAPI for apitype=%s method=%s: %s", apitype, method, err)
	}
	// XXX - for the moment, PerPad APIs don't return any values
	return "", err
}

func (e *Venue) parameterCallback(name, value string) (err error) {
	/*
		switch name {

		case "play:midifile":
			venue := "PhotonSalon1"
			e.playMIDIFile, err = engine.NewMIDIFile(engine.MidifilePath(venue, value))
			if err != nil {
				return err
			}
			p := e.playMIDIFile.Phrase()
			if DebugVenue.MIDI {
				log.Printf("Venue.playMIDIFile: %s has %d notes\n", value, p.NumNotes())
			}
			cid := "fakecid"
			// XXX - does this work?  Might not
			e.activePhrasesManager.StopPhrase(cid, nil, true)
			e.activePhrasesManager.StartPhrase(p, cid)
		}

	*/
	return nil
}

func (e *Venue) oldexecuteGlobalAPI(method string, pmap map[string]string) (result string, err error) {

	globalParams := e.globalParams

	result = "0" // most APIs just return 0, so pre-populate it

	switch method {

	case "midievent":
		log.Printf("global.midievent: Should eventually call handleMidi\n")
		// e.handleMIDI(m engine.MidiDeviceEvent)

	case "set_param":
		nm := pmap["param"]
		val := pmap["value"]
		err := globalParams.SetParamValueWithString(nm, val, e.parameterCallback)
		if err != nil {
			return "", err
		}

	case "set_params":

		log.Printf("set_params API in executeGlobalAPI - needed?\n")
		for name, value := range pmap {
			err := globalParams.SetParamValueWithString(name, value, e.parameterCallback)
			if err != nil {
				return "", fmt.Errorf("error in SoundAPI: method=sound.set_params name=%s value=%s err=%s", name, value, err)
			}
		}

		/*
			case "playmidifile":
				// If  you pass in a midifile argument, it is used
				// to set the value of the global "play:midifile" value.
				// the globalParams "play:midifile" value is used.
				mf, ok := pmap["midifile"]
				if !ok {
					return "", fmt.Errorf("playmidifile: missing midifile value in arguments")
				}
				e.playMIDIFile, err = engine.NewMIDIFile(engine.MidiFilePath(mf))
				if err != nil {
					return "", fmt.Errorf("playmidifile: error reading midifile - %s", err)
				}
				p := e.playMIDIFile.Phrase()
				cid := "fakecid"
				e.activePhrasesManager.StopPhrase(cid, nil, true)
				e.activePhrasesManager.StartPhrase(p, cid)

			case "sprite":
				log.Printf("Adding sprite\n")
				reactor := e.Reactors["B"]

				x := rand.Float32()
				y := rand.Float32()
				z := rand.Float32()
				params := reactor.ParamsOfCategory["visual"]
				ss := NewSpriteSquare(x, y, z, params)
				reactor.AddSprite(ss)

			case "stopmidifile":
				cid := "fakecid"
				e.activePhrasesManager.StopPhrase(cid, nil, true)
		*/

	default:
		err = fmt.Errorf("executeGlobalAPI unrecognized meth=%v", method)
		result = ""
	}

	return result, err
}

// Time returns the current time
func (e *Venue) Time() time.Time {
	return time.Now()
}
