package kit

import (
	"fmt"
	"strings"
	"sync"

	"github.com/hypebeast/go-osc/osc"
)

type Patch struct {
	mutex  sync.RWMutex
	synth  *Synth
	name   string
	params *ParamValues
	// listeners []*PluginInstance
}

// In a quad file, the parameter names are of the form:
// {patchName}-{parametername}.
const PatchNameSeparator = "-"

var Patchs = map[string]*Patch{}

func PatchNames() []string {
	arr := []string{}
	for nm := range Patchs {
		arr = append(arr, nm)
	}
	return arr
}

func NewPatch(patchName string) *Patch {
	patch := &Patch{
		synth:  GetSynth(""),
		name:   patchName,
		params: NewParamValues(),
		// listeners: []*PluginInstance{},
	}
	patch.SetDefaultValues()
	Patchs[patchName] = patch
	return patch
}

func GetPatch(patchName string) *Patch {
	patch, ok := Patchs[patchName]
	if !ok {
		return nil
	}
	return patch
}

func IsPerPatchParam(name string) bool {
	return !strings.HasPrefix(name, "global.")
}

// func ApplyToAllPatchs(f func(patch *Patch)) {
// 	for _, patch := range Patchs {
// 		f(patch)
// 	}
// }

func (patch *Patch) Synth() *Synth {

	patch.mutex.RLock()
	defer patch.mutex.RUnlock()

	if patch.synth == nil {
		// This shouldn't happen
		LogWarn("Hey, Synth() finds patch.synth==nil?")
	}
	return patch.synth
}

// SetDefaultValues xxx
func (patch *Patch) SetDefaultValues() {
	for nm, d := range ParamDefs {
		if IsPerPatchParam(nm) {
			err := patch.SetParam(nm, d.Init)
			if err != nil {
				LogIfError(err)
			}
		}
	}
}

func (patch *Patch) MIDIChannel() uint8 {
	synth := patch.Synth()
	if synth == nil {
		LogWarn("MIDIChannel: synth is nil, returning 0")
		return 0
	}
	return synth.Channel()
}

func (patch *Patch) RefreshAllIfPortnumMatches(ffglportnum int) {
	portnum, _ := TheResolume().PortAndLayerNumForPatch(patch.name)
	if portnum == ffglportnum {
		patch.RefreshAllPatchValues()
	}
}

func (patch *Patch) RefreshAllPatchValues() {
	for _, paramName := range patch.ParamNames() {
		paramValue := patch.Get(paramName)
		patch.noticeValueChange(paramName, paramValue)
	}
}

func (patch *Patch) noticeValueChange(paramName string, paramValue string) {

	if strings.HasPrefix(paramName, "visual.") {
		name := strings.TrimPrefix(paramName, "visual.")
		msg := osc.NewMessage("/api")
		msg.Append("set_params")
		args := fmt.Sprintf("{\"%s\":\"%s\"}", name, paramValue)
		msg.Append(args)
		TheResolume().ToFreeFramePlugin(patch.Name(), msg)
	}

	if strings.HasPrefix(paramName, "effect.") {
		name := strings.TrimPrefix(paramName, "effect.")
		// Effect parameters get sent to Resolume
		TheResolume().SendEffectParam(patch.Name(), name, paramValue)
	}

	if paramName == "sound.synth" {
		synth := GetSynth(paramValue)
		patch.mutex.Lock()
		patch.synth = synth
		// LogInfo("Patch.noticeValueChange: changed synth", "patch", patch.name, "synth.name", synth.name)
		patch.mutex.Unlock()
	}
}

func (patch *Patch) SaveAndAlert(category string, filename string) error {

	LogOfType("saved", "Patch.SaveAndAlert", "category", category, "filename", filename)

	if filename == "_Current" && category != "quad" {
		err := fmt.Errorf(", we're only saving _Current in quad (and global)")
		LogIfError(err)
		return err
	}
	err := patch.params.Save(category, filename)
	if err != nil {
		LogIfError(err)
		return err
	}
	patch.RefreshAllPatchValues()
	return nil
}

func (patch *Patch) Name() string {
	return patch.name
}

func (patch *Patch) CursorToQuant(ce CursorEvent) Clicks {

	quantStyle := patch.Get("misc.quantstyle")

	q := Clicks(1)
	switch quantStyle {

	case "none", "":
		// q is 1

	case "frets":
		y := float64(ce.Pos.Y)

		high, err := GetParamFloat("global.timefret_high")
		if err != nil {
			LogIfError(err)
			return q
		}
		mid, err := GetParamFloat("global.timefret_mid")
		if err != nil {
			LogIfError(err)
			return q
		}
		low, err := GetParamFloat("global.timefret_low")
		if err != nil {
			LogIfError(err)
			return q
		}

		if y > high {
			q = OneBeat / 8
		} else if y > mid {
			q = OneBeat / 4
		} else if y > low {
			q = OneBeat / 2
		} else {
			q = OneBeat
		}

	case "fixed":
		q = OneBeat / 4

	case "pressure":
		LogWarn("CursortToQuant, pressure quant needs to use zmin/zmax")
		z := ce.Pos.Z
		if z > 0.20 {
			q = OneBeat / 8
		} else if z > 0.10 {
			q = OneBeat / 4
		} else if z > 0.05 {
			q = OneBeat / 2
		} else {
			q = OneBeat
		}

	default:
		LogWarn("Unrecognized quant", "quantstyle", quantStyle)
	}
	q = Clicks(float64(q) / TempoFactor)
	LogOfType("quant", "CursorToQuant", "q", q, "quantstyle", quantStyle)
	return q
}

func (patch *Patch) Api(api string, apiargs map[string]string) (string, error) {

	switch api {

	case "load":
		filename, okfilename := apiargs["filename"]
		if !okfilename {
			return "", fmt.Errorf("missing filename parameter")
		}
		category, okcategory := apiargs["category"]
		if !okcategory {
			return "", fmt.Errorf("missing category parameter")
		}

		// Note that if it's a "quad" file,
		// we're only loading the items
		// in that quad that match the patch
		LogInfo("Patch.Load", "patch", patch.Name(), "category", category, "filename", filename)

		paramsMap, err := LoadParamsMapOfCategory(category, filename)
		if err != nil {
			LogError(err)
			return "", err
		}

		err = patch.Load(category, paramsMap)
		if err != nil {
			return "", err
		}
		return "", patch.SaveQuadAndAlert()

	case "save":
		// This is a save of a single patch (in saved/patch/*)
		filename, okfilename := apiargs["filename"]
		if !okfilename {
			return "", fmt.Errorf("missing filename parameter")
		}
		category, okcategory := apiargs["category"]
		if !okcategory {
			return "", fmt.Errorf("missing category parameter")
		}
		return "", patch.SaveAndAlert(category, filename)

	case "set":
		name, value, err := GetNameValue(apiargs)
		if err != nil {
			return "", fmt.Errorf("executePatchApi: err=%s", err)
		}
		err = patch.SetParam(name, value)
		if err != nil {
			return "", err
		}
		patch.noticeValueChange(name, value)
		return "", patch.SaveQuadAndAlert()

	case "setparams":
		for name, value := range apiargs {
			err := patch.SetParam(name, value)
			if err != nil {
				// TODO: should we return as soon as we get the first error?
				LogIfError(err)
				return "", err
			}
			patch.noticeValueChange(name, value)
		}
		return "", patch.SaveQuadAndAlert()

	case "get":
		name, ok := apiargs["name"]
		if !ok {
			return "", fmt.Errorf("Patch.Api: missing name argument")
		}
		return patch.Get(name), nil

	case "clear":
		patch.loopClear()
		patch.clearGraphics()
		return "", nil

	case "fade":
		patch.loopFade()
		return "", nil

	case "filter":
		patch.loopFilter()
		return "", nil

	default:
		// ignore errors on these for the moment
		if strings.HasPrefix(api, "loop_") || strings.HasPrefix(api, "midi_") {
			return "", nil
		}
		err := fmt.Errorf("Patch.Api: unrecognized api=%s", api)
		LogIfError(err)
		return "", err
	}
}

func (patch *Patch) SaveQuadAndAlert() error {
	return TheQuad.saveQuad("_Current")
}

func (patch *Patch) SetParam(paramName string, paramValue string) error {

	if !IsPerPatchParam(paramName) {
		err := fmt.Errorf("Patch.Set: not per-patch param=%s", paramName)
		LogIfError(err)
		return err
	}
	return patch.params.SetParamWithString(paramName, paramValue)
}

// If no such parameter, return ""
func (patch *Patch) Get(paramName string) string {
	s, err := patch.params.Get(paramName)
	if err != nil {
		LogWarn("Patch.Get, error", "err", err)
		return ""
	}
	return s
}

func (patch *Patch) ParamNames() []string {
	return patch.params.ParamNames()
}

// func (patch *Patch) GetString(paramName string) string {
// 	return patch.params.GetStringValue(paramName, "")
// }

func (patch *Patch) GetInt(paramName string) int {
	return patch.params.GetIntValue(paramName)
}

func (patch *Patch) GetBool(paramName string) bool {
	return patch.params.GetBoolValue(paramName)
}

func (patch *Patch) GetFloat(paramName string) float64 {
	return patch.params.GetFloatValue(paramName)
}

func (patch *Patch) SetPatchValuesFromQuadMap(paramsmap map[string]any) error {

	for fullParamName, paramValue := range paramsmap {

		// Since we're applying a quad to a patch, we ignore
		// all the values that aren't patch parameters
		if !IsPerPatchParam(fullParamName) {
			continue
		}

		// Also, the parameter names in a quad file are of the form:
		// {patchName}-{parametername}.  We only want to apply
		// parmeters that are intended for this specific patch, so...
		i := strings.Index(fullParamName, PatchNameSeparator)
		if i < 0 {
			LogIfError(fmt.Errorf("applyQuadValuesFrom: no PatchNameSeparator"), "param", fullParamName)
			continue
		}
		patchOfParam := fullParamName[0:i]
		if patch.Name() != patchOfParam {
			continue
		}

		// the name give to patch.Set doesn't include the patch name
		paramName := fullParamName[i+1:]

		// We expect the parameter to be of the form
		// {category}.{parameter}, but old "saved" files
		// didn't include the category.
		if !strings.Contains(paramName, ".") {
			LogWarn("applyQuadValuesFrom: OLD format, not supported")
			return fmt.Errorf("")
		}

		value, ok := paramValue.(string)
		if !ok {
			LogIfError(fmt.Errorf("ApplyPatchValuesFromQuadMap: Needs to handle new value format"))
			return fmt.Errorf("value of name=%s isn't a string", fullParamName)
		}
		err := patch.SetParam(paramName, value)
		if err != nil {
			LogWarn("applyQuadValuesFrom", "name", paramName, "err", err)
			// Don't fail completely on individual failures,
			// some might be for parameters that no longer exist.
		}
	}

	// For any parameters that are in Paramdefs but are NOT in the loaded
	// saved, we put out the "init" values.  This happens when new parameters
	// are added which don't exist in existing saved files.
	for paramName, def := range ParamDefs {
		if !IsPerPatchParam(paramName) {
			continue
		}
		patchParamName := patch.Name() + PatchNameSeparator + paramName
		_, found := paramsmap[patchParamName]
		if !found {
			init := def.Init
			err := patch.SetParam(paramName, init)
			if err != nil {
				// a hack to eliminate errors on a parameter that still exists somewhere
				LogWarn("applyQuadValuesFrom", "nm", paramName, "err", err)
				// Don't fail completely on individual failures,
				// some might be for parameters that no longer exist.
			}
		}
	}
	return nil
}

func (patch *Patch) Load(category string, paramsMap ParamsMap) error {

	if category == "quad" {
		// this will only load things to this one patch
		err := patch.SetPatchValuesFromQuadMap(paramsMap)
		if err != nil {
			LogIfError(err)
			return err
		}
	} else {
		patch.params.ApplyValuesFromMap(category, paramsMap, patch.params.SetParamWithString)
	}

	// If there's a _override.json file, use it
	overridemap := OverrideMap()
	if overridemap != nil {
		patch.params.ApplyValuesFromMap(category, overridemap, patch.params.SetParamWithString)
	}

	// For any parameters that are in Paramdefs but are NOT in the loaded
	// saved, we put out the "init" values.  This happens when new parameters
	// are added which don't exist in existing saved files.
	for paramName, def := range ParamDefs {
		// Only include patch parameters
		if !IsPerPatchParam(paramName) {
			continue
		}
		if def.Category != category {
			continue
		}
		_, found := paramsMap[paramName]
		if !found {
			paramValue := def.Init
			err := patch.SetParam(paramName, paramValue)
			// err := params.Set(paramName, paramValue)
			if err != nil {
				LogWarn("patch.Set error", "saved", paramName, "err", err)
				// Don't fail completely
			}
		}
	}
	patch.RefreshAllPatchValues()
	return nil
}

func (patch *Patch) clearGraphics() {
	TheResolume().ToFreeFramePlugin(patch.Name(), osc.NewMessage("/clear"))
}

func (patch *Patch) Status() string {
	if patch == nil {
		return ""
	}
	nevents := TheScheduler.CountEventsWithTag(patch.name)
	if nevents == 0 {
		return ""
	}
	return fmt.Sprintf("%d", nevents)
}

func (patch *Patch) loopFade() {
	tag := patch.name

	// TheCursorManager.DeleteActiveCursorsForTag(tag)
	// LogInfo("loopClear before DeleteEvents")
	TheScheduler.FadeEventsWithTag(tag)
	// LogInfo("loopClear after DeleteEvents")

	TheScheduler.pendingMutex.Lock()
	clearPending := false
	for _, se := range TheScheduler.pendingScheduled {
		if se.Tag == tag {
			LogInfo("HEY!, saw pendingSchedule with tag prefix!", "prefix", tag, "se", se)
			clearPending = true
		}
	}
	if clearPending {
		// LogInfo("loopClear is clearing pendingScheduled")
		TheScheduler.pendingScheduled = nil
	}
	TheScheduler.pendingMutex.Unlock()
}

func (patch *Patch) loopFilter() {
	tag := patch.name

	// TheCursorManager.DeleteActiveCursorsForTag(tag)
	// LogInfo("loopClear before DeleteEvents")
	TheScheduler.FilterEventsWithTag(tag)
	// LogInfo("loopClear after DeleteEvents")

	TheScheduler.pendingMutex.Lock()
	clearPending := false
	for _, se := range TheScheduler.pendingScheduled {
		if se.Tag == tag {
			LogInfo("HEY!, saw pendingSchedule with tag prefix!", "prefix", tag, "se", se)
			clearPending = true
		}
	}
	if clearPending {
		// LogInfo("loopClear is clearing pendingScheduled")
		TheScheduler.pendingScheduled = nil
	}
	TheScheduler.pendingMutex.Unlock()
}

func (patch *Patch) loopClear() {
	tag := patch.name

	// TheCursorManager.DeleteActiveCursorsForTag(tag)
	// LogInfo("loopClear before DeleteEvents")
	TheScheduler.DeleteEventsWithTag(tag)
	// LogInfo("loopClear after DeleteEvents")

	TheScheduler.pendingMutex.Lock()
	clearPending := false
	for _, se := range TheScheduler.pendingScheduled {
		if se.Tag == tag {
			LogInfo("HEY!, saw pendingSchedule with tag prefix!", "prefix", tag, "se", se)
			clearPending = true
		}
	}
	if clearPending {
		// LogInfo("loopClear is clearing pendingScheduled")
		TheScheduler.pendingScheduled = nil
	}
	TheScheduler.pendingMutex.Unlock()

	/*
		// XXX - SHOULD BE USING DeleteEventsWhoseGidIs(cidToDelete string)
		TheScheduler.mutex.Lock()
		var nexti *list.Element
		for i := TheScheduler.schedList.Front(); i != nil; i = nexti {
			nexti = i.Next()
			se := i.Value.(*SchedElement)
			if se.Tag != tag {
				continue
			}
			switch v := se.Value.(type) {

			case *NoteOn:
				// LogInfo("loopClear Saw Noteon", "v", v)

			case *NoteOff:
				LogInfo("loopClear Saw Noteoff", "v", v)
				if patch.synth != nil {
					LogInfo("LOOP LOOPCLEAR SENDING NOTEOFF", "v", v)
					patch.synth.SendNoteToMidiOutput(v)
				}

			case CursorEvent:
				// LogInfo("loopClear Saw CursorEvent", "v", v)

			case MidiEvent:
				// LogInfo("loopClear Saw MidiEvent", "v", v)
			}
			TheScheduler.schedList.Remove(i)
		}
		// LogInfo("At end of loopClear", "schedList.Len", TheScheduler.schedList.Len())
		TheScheduler.mutex.Unlock()

		// LogOfType("loop", "Patch.loopClear end", "schdule", TheScheduler.ToString())
		// LogInfo("LOOP LOOPCLEAR END", "prefix", tag)
	*/
}
