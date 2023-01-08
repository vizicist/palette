package engine

import (
	"fmt"
	"time"

	midi "gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
)

type PluginFunc func(ctx *PluginContext, api string, apiargs map[string]string) (string, error)

type PluginContext struct {
	api PluginFunc
	// params        *ParamValues
	sources map[string]bool
}

func (ctx *PluginContext) Uptime() float64 {
	return Uptime()
}

func (ctx *PluginContext) GetArgsXYZ(args map[string]string) (x, y, z float32, err error) {
	return GetArgsXYZ(args)
}

func (ctx *PluginContext) ClearCursors() {
	TheRouter().cursorManager.clearCursors()
}

/*
func (ctx *PluginContext) LogInfo(msg string, keysAndValues ...any) {
	LogInfo(msg, keysAndValues...)
}

func (ctx *PluginContext) LogWarn(msg string, keysAndValues ...any) {
	LogWarn(msg, keysAndValues...)
}

func (ctx *PluginContext) LogError(err error, keysAndValues ...any) {
	LogError(err, keysAndValues...)
}
*/

func (ctx *PluginContext) PaletteDir() string {
	return PaletteDir()
}

func (ctx *PluginContext) ConfigValueWithDefault(name, dflt string) string {
	return ConfigValueWithDefault(name, dflt)
}

func (ctx *PluginContext) ConfigBoolWithDefault(name string, dflt bool) bool {
	return ConfigBoolWithDefault(name, dflt)
}

func (ctx *PluginContext) ConfigValue(name string) string {
	return ConfigValue(name)
}

func (ctx *PluginContext) ConfigFilePath(name string) string {
	return ConfigFilePath(name)
}

func (ctx *PluginContext) StartExecutableLogOutput(logName string, fullexe string, background bool, args ...string) error {
	return StartExecutableLogOutput(logName, fullexe, background, args...)
}

func (ctx *PluginContext) AddProcess(process string, info *ProcessInfo) {
	TheEngine().ProcessManager.AddProcess(process, info)
}

func (ctx *PluginContext) AddProcessBuiltIn(process string) {
	TheEngine().ProcessManager.AddProcessBuiltIn(process)
}

func (ctx *PluginContext) CheckAutostartProcesses() {
	TheEngine().ProcessManager.CheckAutostartProcesses()
}

func (ctx *PluginContext) StartRunning(process string) error {
	return TheEngine().ProcessManager.StartRunning(process)
}

func (ctx *PluginContext) StopRunning(process string) error {
	return TheEngine().ProcessManager.StopRunning(process)
}

func (ctx *PluginContext) KillExecutable(exe string) error {
	return killExecutable(exe)
}

func (ctx *PluginContext) IsRunningExecutable(exe string) bool {
	return isRunningExecutable(exe)
}

func (ctx *PluginContext) FileExists(path string) bool {
	return FileExists(path)
}

func (ctx *PluginContext) GetLayer(layerName string) *Layer {
	return GetLayer(layerName)
}

func (ctx *PluginContext) SaveParams(params *ParamValues, category string, filename string) error {
	return params.Save(category, filename)
}

func (ctx *PluginContext) AllowSource(source ...string) {
	var ok bool
	for _, name := range source {
		_, ok = ctx.sources[name]
		if ok {
			LogInfo("AllowSource: already set?", "source", name)
		} else {
			ctx.sources[name] = true
		}
	}
}

func (ctx *PluginContext) IsSourceAllowed(source string) bool {
	_, ok := ctx.sources[source]
	return ok
}

/*
func (ctx *PluginContext) MidiEventToNote(me MidiEvent) (note any, error) {
	pe, err := ctx.midiEventToPhraseElement(me)
	if err != nil {
		return nil, err
	}
	phr := NewPhrase().InsertElement(pe)
	return phr, nil
}
*/

func (ctx *PluginContext) MidiEventToNote(me MidiEvent) (any, error) {

	bytes := me.Msg.Bytes()
	lng := len(bytes)
	if lng == 0 {
		err := fmt.Errorf("no bytes in midi.Message")
		LogError(err)
		return nil, err
	}
	if lng == 1 {
		return &PhraseElement{Value: NewNoteSystem(bytes[0])}, nil
	}

	var data0, data1, data2 byte
	// NOTE: channel on incoming MIDI is ignored
	// it uses whatever sound the ctx is using.
	status := bytes[0] & 0xf0
	if lng == 2 {
		switch status {
		case 0xd0: // channel pressure
			return &PhraseElement{Value: NewChanPressure(bytes[0], bytes[1])}, nil
		}

	}

	switch len(bytes) {
	case 0:
	case 1:
		data0 = bytes[0]
		status = bytes[0] & 0xf0
	case 2:
		data0 = bytes[0]
		status = bytes[0]
		data1 = bytes[1]
	case 3:
		data0 = bytes[0]
		status = bytes[0]
		data1 = bytes[1]
		data2 = bytes[2]
	default:
	}

	//	if (status == 0x90 || status == 0x80) && ctx.MIDIThruScadjust {
	//		scale := ctx.getScale()
	//		pitch = scale.ClosestTo(pitch)
	//	}

	var val any
	switch status {
	case 0x90:
		val = NewNoteOn(data0, data1, data2)
	case 0x80:
		val = NewNoteOff(data0, data1, data2)
	case 0xB0:
		val = NewController(data0, data1, data2)
	case 0xC0:
		val = NewProgramChange(data0, data1)
	case 0xD0:
		val = NewChanPressure(data0, data1)
	case 0xE0:
		val = NewPitchBend(data0, data1, data2)
	default:
		err := fmt.Errorf("MidiEventToPhraseElement: unable to handle status=0x%02x", status)
		LogError(err, "status", status)
		return nil, err
	}

	return val, nil
}

func (ctx *PluginContext) CurrentClick() Clicks {
	return CurrentClick()
}

func (ctx *PluginContext) ScheduleMidi(layer *Layer, msg midi.Message, click Clicks) {
	go func() {
		se := &SchedElement{
			AtClick: click,
			layer:   layer,
			Value:   MidiSchedValue{msg},
		}
		TheEngine().Scheduler.cmdInput <- SchedulerElementCmd{se}
	}()
}

/*
func (ctx *PluginContext) SubmitCommand(command Command) {
	go func() {
		TheEngine().Scheduler.cmdInput <- command
	}()
}
*/

/*
func (ctx *PluginContext) ParamIntValue(name string) int {
	return ctx.params.GetIntValue(name)
}

func (ctx *PluginContext) ParamBoolValue(name string) bool {
	return ctx.params.GetBoolValue(name)
}
*/

/*
func (ctx *EngineContext) GetScale() *Scale {
	return ctx.scale
}

func (ctx *EngineContext) LayerParams(layerName string) *ParamValues {
	params, ok := ctx.layerParams[layerName]
	if !ok {
		params = NewParamValues()
		ctx.layerParams[layerName] = params
	}
	return params
}
*/

/*
func (ctx *EngineContext) setOneLayerParamValue(layerName, fullname, value string) error {

	layer := GetLayer(layerName)
	params := layer.params
	err := params.SetParamValueWithString(fullname, value)
	if err != nil {
		return err
	}

	if strings.HasPrefix(fullname, "visual.") {
		name := strings.TrimPrefix(fullname, "visual.")
		msg := osc.NewMessage("/api")
		msg.Append("set_params")
		args := fmt.Sprintf("{\"%s\":\"%s\"}", name, value)
		msg.Append(args)
		layer.toFreeFramePlugin(msg)
	}

	if strings.HasPrefix(fullname, "effect.") {
		name := strings.TrimPrefix(fullname, "effect.")
		// Effect parameters get sent to Resolume
		ctx.sendEffectParam(layer, name, value)
	}

	return nil
}
*/

/*
 */

// to avoid unused warning
// var p *Layer

/*
// getScale xxx
func (ctx *EngineContext) getScale() *Scale {
	var scaleName string
	var scale *Scale

	// if Layer.MIDIUseScale {
	// 	scale = Layer.externalScale
	// } else {
	scaleName = ctx.pluginParams.ParamStringValue("misc.scale", "newage")
	scale = GlobalScale(scaleName)
	// }
	return scale
}
*/

/*
func (ctx *EngineContext) sendEffectParam(name string, value string) {
	// Effect parameters that have ":" in their name are plugin parameters
	i := strings.Index(name, ":")
	if i > 0 {
		effectName := name[0:i]
		paramName := name[i+1:]
		ctx.sendPadOneEffectParam(effectName, paramName, value)
	} else {
		onoff, err := strconv.ParseBool(value)
		if err != nil {
			LogError(err)
			onoff = false
		}
		ctx.sendPadOneEffectOnOff(name, onoff)
	}
}
*/

func (ctx *PluginContext) handleMIDITimeReset() {
	LogWarn("HandleMIDITimeReset!! needs implementation")
}

func (ctx *PluginContext) GetSynth(synthName string) *Synth {
	return GetSynth(synthName)
}

/*
func (ctx *EngineContext) LayerApplySaved(layerName, savedName string) error {
	saved, err := LoadSaved(savedName)
	if err != nil {
		LogError(err, "saved", savedName)
		return err
	}
	saved.ApplyTo(ctx.LayerParams(layerName))
	return fmt.Errorf("LayerApplySaved needs work")
}
*/

func (ctx *PluginContext) OpenMIDIOutput(name string) drivers.In {
	return nil
}

/*
// GetSaved is guaranteed to return non=nil
func (ctx *PluginContext) GetSaved(savedName string) *OldSaved {
	saved := GetSaved(savedName)
	return saved
}
*/

/*
// ExecuteAPI xxx
func (ctx *PluginContext) ExecuteAPI(api string, args map[string]string, rawargs string) (result string, err error) {

	LogOfType("api", "Plugin.ExecutAPI called", "api", api, "args", args)

	// The caller can provide rawargs if it's already known, but if not provided, we create it
	// if rawargs == "" {
	// 	rawargs = MapString(args)
	// }

	// ALL visual.* APIs get forwarded to the FreeFrame plugin inside Resolume
	if strings.HasPrefix(api, "visual.") {
		LogInfo("ExecuteAPI: visual.* apis need work")
		// msg := osc.NewMessage("/api")
		// msg.Append(strings.TrimPrefix(api, "visual."))
		// msg.Append(rawargs)
		// layer := "A"
		// ctx.toFreeFramePluginForLayer(layer, msg)
	}

	switch api {

	//
	// case "send":
	// ctx.sendAllParameters()
	// return "", err

	//	case "loop_recording":
	//		v, e := needBoolArg("onoff", api, args)
	//		if e == nil {
	//			ctx.loopIsRecording = v
	//		} else {
	//			err = e
	//		}
	//
	//	case "loop_playing":
	//		v, e := needBoolArg("onoff", api, args)
	//		if e == nil && v != layer.loopIsPlaying {
	//			layer.loopIsPlaying = v
	//			TheEngine().Scheduler.SendAllPendingNoteoffs()
	//		} else {
	//			err = e
	//		}
	//
	//	case "loop_clear":
	//		// layer.loop.Clear()
	//		// layer.clearGraphics()
	//		layer.sendANO()
	//
	//	case "loop_length":
	//		i, e := needIntArg("value", api, args)
	//		if e == nil {
	//			nclicks := Clicks(i)
	//			if nclicks != layer.loopLength {
	//				layer.loopLength = nclicks
	//				// layer.loopSetLength(nclicks)
	//			}
	//		} else {
	//			err = e
	//		}
	//
	//	case "loop_fade":
	//		f, e := needFloatArg("fade", api, args)
	//		if e == nil {
	//			layer.fadeLoop = f
	//		} else {
	//			err = e
	//		}
	//
	//	case "ANO":
	//		layer.sendANO()

		case "midi_thru":
			v, e := needBoolArg("onoff", api, args)
			if e == nil {
				layer.MIDIThru = v
			} else {
				err = e
			}

		case "midi_setscale":
			v, e := needBoolArg("onoff", api, args)
			if e == nil {
				layer.MIDISetScale = v
			} else {
				err = e
			}

		case "clearexternalscale":
			layer.clearExternalScale()
			layer.MIDINumDown = 0

		case "midi_quantized":
			v, err := needBoolArg("onoff", api, args)
			if err == nil {
				layer.MIDIQuantized = v
			}

		case "midi_thruscadjust":
			v, e := needBoolArg("onoff", api, args)
			if e == nil {
				layer.MIDIThruScadjust = v
			} else {
				err = e
			}

		case "set_transpose":
			v, e := needIntArg("value", api, args)
			if e == nil {
				layer.TransposePitch = v
				Info("layer API set_transpose", "pitch", v)
			} else {
				err = e
			}

	default:
		err = fmt.Errorf("Layer.ExecuteAPI: unknown api=%s", api)
	}

	return result, err
}
*/

func (ctx *PluginContext) GenerateCursorGesture(cid string, noteDuration time.Duration, x0, y0, z0, x1, y1, z1 float32) {
	TheRouter().cursorManager.generateCursorGesture(cid, noteDuration, x0, y0, z0, x1, y1, z1)
}

func (ctx *PluginContext) GetCursorState(cid string) *CursorState {
	return TheRouter().cursorManager.GetCursorState(cid)
}
