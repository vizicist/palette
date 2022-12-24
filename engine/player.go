package engine

/*
// Layer is an entity that that reacts to things (cursor events, apis) and generates output (midi, graphics)
type Layer struct {
	layerName      string
	resolumeLayer   int // see ResolumeLayerForPad
	freeframeClient *osc.Client
	resolumeClient  *osc.Client

	params  *ParamValues // across all presets
	sources map[string]string

	agents        []Agent
	agentsContext []*EngineContext


*/

/*

func (p *Layer) SetResolumeLayer(layernum int, ffglport int) {
	p.resolumeLayer = layernum
	p.resolumeClient = osc.NewClient(LocalAddress, ResolumePort)
	p.freeframeClient = osc.NewClient(LocalAddress, ffglport)
}
*/

/*
// NewLayer makes a new Layer
func NewLayer(layerName string) *Layer {
	p := &Layer{
		layerName:      layerName,
		resolumeLayer:   0,
		freeframeClient: nil,
		resolumeClient:  nil,
		// tempoFactor:         1.0,
		// params:                         make(map[string]any),
		params: NewParamValues(),
		// activeNotes: make(map[string]*ActiveNote),
		// fadeLoop:    0.5,
		sources: map[string]string{},

		// MIDIThru:         true,
		// MIDISetScale:     false,
		// MIDIThruScadjust: false,
		// MIDIUseScale:     false,
		// MIDIQuantized:    false,
		// TransposePitch:   0,
	}
	p.params.SetDefaultValues()
	//p.clearExternalScale()
	//p.setExternalScale(60%12, true) // Middle C

	TheRouter().agentManager.AddAgent(p)

	return p
}
*/

/*
func (layer *Layer) HandleCursorEvent(ce CursorEvent) {
	DebugLogOfType("cursor", "Layer.HandleCursorEvent", "ce", ce)
	for n, agent := range layer.agents {
		ctx := layer.agentsContext[n]
		agent.OnCursorEvent(ctx, ce)
	}
}

// HandleMIDIInput xxx
func (layer *Layer) HandleMidiEvent(me MidiEvent) {
	Info("Layer.HandleMidiEvent", "me", me)
	for n, agent := range layer.agents {
		ctx := layer.agentsContext[n]
		agent.OnMidiEvent(ctx, me)
	}

		layer.midiInputMutex.Lock()
		defer layer.midiInputMutex.Unlock()

		DebugLogOfType("midi", "Router.HandleMIDIInput", "event", e)

		if layer.MIDIThru {
			layer.PassThruMIDI(e)
		}
		if layer.MIDISetScale {
			layer.handleMIDISetScaleNote(e)
		}
}

/*
func CallerPackageAndFunc() (string, string) {
	pc, _, _, _ := runtime.Caller(1)

	funcName := runtime.FuncForPC(pc).Name()
	lastDot := strings.LastIndexByte(funcName, '.')

	packagename := funcName[:lastDot]
	funcname := funcName[lastDot+1:]
	return packagename, funcname
}

// CallerFunc gives just the final func name
func CallerFunc() string {
	pc, _, _, _ := runtime.Caller(1)

	funcName := runtime.FuncForPC(pc).Name()
	lastSlash := strings.LastIndexByte(funcName, '/')

	funcname := funcName[lastSlash+1:]
	return funcname
}

func (layer *Layer) SetParam(fullname, value string) error {
	return layer.SetOneParamValue(fullname, value)
}

func (layer *Layer) SetOneParamValue(fullname, value string) error {

	DebugLogOfType("value", "SetOneParamValue", "layer", layer.layerName, "fullname", fullname, "value", value)
	err := layer.params.SetParamValueWithString(fullname, value, nil)
	if err != nil {
		return err
	}

	if strings.HasPrefix(fullname, "visual.") {
		name := strings.TrimPrefix(fullname, "visual.")
		msg := osc.NewMessage("/api")
		msg.Append("set_params")
		args := fmt.Sprintf("{\"%s\":\"%s\"}", name, value)
		msg.Append(args)
		layer.toFreeFramePluginForLayer(msg)
	}

	if strings.HasPrefix(fullname, "effect.") {
		name := strings.TrimPrefix(fullname, "effect.")
		// Effect parameters get sent to Resolume
		layer.sendEffectParam(name, value)
	}

	return nil
}

/*
// ClearExternalScale xxx
func (layer *Layer) clearExternalScale() {
	DebugLogOfType("scale", "clearExternalScale", "pad", layer.layerName)
	layer.externalScale = MakeScale()
}
*/

/*
var uniqueIndex = 0


/*
func (layer *Layer) generateVisualsFromCursor(ce CursorEvent) {
	if !TheRouter().generateVisuals {
		return
	}
	// send an OSC message to Resolume
	msg := osc.NewMessage("/cursor")
	msg.Append(ce.Ddu)
	msg.Append(ce.ID)
	msg.Append(float32(ce.X))
	msg.Append(float32(ce.Y))
	msg.Append(float32(ce.Z))
	layer.toFreeFramePluginForLayer(msg)
}
*/

/*

 */

/*
 */

/*
func (r *Layer) paramStringValue(paramname string, def string) string {
	r.paramsMutex.RLock()
	param, ok := r.params[paramname]
	r.paramsMutex.RUnlock()
	if !ok {
		return def
	}
	return r.params.param).(paramValString).value
}

func (r *Layer) paramIntValue(paramname string) int {
	r.paramsMutex.RLock()
	param, ok := r.params[paramname]
	r.paramsMutex.RUnlock()
	if !ok {
		Warn("No param named","paramname", paramname)
		return 0
	}
	return (param).(paramValInt).value
}
*/

/*
 */

/*
func (layer) *Layer) cursorToPitch(ce CursorStepEvent) uint8 {
	pitchmin := layer.params.ParamIntValue("sound.pitchmin")
	pitchmax := layer.params.ParamIntValue("sound.pitchmax")
	dp := pitchmax - pitchmin + 1
	p1 := int(ce.X * float32(dp))
	p := uint8(pitchmin + p1%dp)
	chromatic := layer.params.ParamBoolValue("sound.chromatic")
	if !chromatic {
		scale := layer.getScale()
		p = scale.ClosestTo(p)
		// MIDIOctaveShift might be negative
		i := int(p) + 12*layer.MIDIOctaveShift
		for i < 0 {
			i += 12
		}
		for i > 127 {
			i -= 12
		}
		p = uint8(i + layer.TransposePitch)
	}
	return p
}
*/

/*
 */

/*
func (layer) *Layer) cursorToDuration(ce CursorStepEvent) int {
	return 92
}

func (layer) *Layer) cursorToQuant(ce CursorStepEvent) Clicks {
	quant := layer.params.ParamStringValue("misc.quant", "fixed")

	q := Clicks(1)
	if quant == "none" || quant == "" {
		// q is 1
	} else if quant == "frets" {
		if ce.Y > 0.85 {
			q = oneBeat / 8
		} else if ce.Y > 0.55 {
			q = oneBeat / 4
		} else if ce.Y > 0.25 {
			q = oneBeat / 2
		} else {
			q = oneBeat
		}
	} else if quant == "fixed" {
		q = oneBeat / 4
	} else if quant == "pressure" {
		if ce.Z > 0.20 {
			q = oneBeat / 8
		} else if ce.Z > 0.10 {
			q = oneBeat / 4
		} else if ce.Z > 0.05 {
			q = oneBeat / 2
		} else {
			q = oneBeat
		}
	} else {
		Warn("Unrecognized quant","quant", quant)
	}
	q = Clicks(float64(q) / TempoFactor)
	return q
}
*/

/*
func (layer) *Layer) loopComb() {

	layer.loop.stepsMutex.Lock()
	defer layer.loop.stepsMutex.Unlock()

	// Create a map of the UP cursor events, so we only do completed notes
	upEvents := make(map[string]CursorStepEvent)
	for _, step := range layer.loop.steps {
		if step.events != nil && len(step.events) > 0 {
			for _, event := range step.events {
				if event.cursorStepEvent.Ddu == "up" {
					upEvents[event.cursorStepEvent.ID] = event.cursorStepEvent
				}
			}
		}
	}
	combme := 0
	combmod := 2 // should be a parameter
	for id := range upEvents {
		if combme == 0 {
			layer.loop.ClearID(id)
			layer.generateSoundFromCursor(upEvents[id])
		}
		combme = (combme + 1) % combmod
	}
}

func (layer) *Layer) loopQuant() {

	layer.loop.stepsMutex.Lock()
	defer layer.loop.stepsMutex.Unlock()

	// XXX - Need to make sure we have mutex for changing loop steps
	// XXX - DOES THIS EVEN WORK?

	Info("DOES LOOPQUANT WORK????")

	// Create a map of the UP cursor events, so we only do completed notes
	// Create a map of the DOWN events so we know how much to shift that cursor.
	quant := oneBeat / 2
	upEvents := make(map[string]CursorStepEvent)
	downEvents := make(map[string]CursorStepEvent)
	shiftOf := make(map[string]Clicks)
	for stepnum, step := range layer.loop.steps {
		if step.events != nil && len(step.events) > 0 {
			for _, e := range step.events {
				switch e.cursorStepEvent.Ddu {
				case "up":
					upEvents[e.cursorStepEvent.ID] = e.cursorStepEvent
				case "down":
					downEvents[e.cursorStepEvent.ID] = e.cursorStepEvent
					shift := layer.nextQuant(Clicks(stepnum), quant)
					shiftOf[e.cursorStepEvent.ID] = Clicks(shift)
				}
			}
		}
	}

	if len(shiftOf) == 0 {
		return
	}

	// We're going to create a brand new steps array
	newsteps := make([]*Step, len(layer.loop.steps))
	for stepnum, step := range layer.loop.steps {
		if step.events == nil || len(step.events) == 0 {
			continue
		}
		for _, e := range step.events {
			if !e.hasCursor {
				newsteps[stepnum].events = append(newsteps[stepnum].events, e)
			} else {
				Info("IS THIS CODE EVER EXECUTED?")
				id := e.cursorStepEvent.ID
				newstepnum := stepnum
				shift, ok := shiftOf[id]
				// It's shifted
				if ok {
					shiftStep := Clicks(stepnum) + shift
					newstepnum = int(shiftStep) % len(layer.loop.steps)
				}
				newsteps[newstepnum].events = append(newsteps[newstepnum].events, e)
			}
		}
	}
}
*/

/*

 */
