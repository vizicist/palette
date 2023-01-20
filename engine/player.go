package engine

/*
// HandleMIDIInput xxx
func (layer *Patch) HandleMidiEvent(me MidiEvent) {
	LogInfo("Patch.HandleMidiEvent", "me", me)
	for n, agent := range layer.agents {
		ctx := layer.agentsContext[n]
		agent.OnMidiEvent(ctx, me)
	}

	layer.midiInputMutex.Lock()
	defer layer.midiInputMutex.Unlock()

	LogOfType("midi", "Router.HandleMIDIInput", "event", e)

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

func (layer *Patch) SetParam(fullname, value string) error {
	return layer.SetOneParamValue(fullname, value)
}

/*
// ClearExternalScale xxx
func (layer *Patch) clearExternalScale() {
	LogOfType("scale", "clearExternalScale", "pad", layer.patchName)
	layer.externalScale = MakeScale()
}
*/

/*
 */

/*
func (layer) *Patch) cursorToPitch(ce CursorStepEvent) uint8 {
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
func (layer) *Patch) cursorToDuration(ce CursorStepEvent) int {
	return 92
}

*/

/*
func (layer) *Patch) loopQuant() {

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
