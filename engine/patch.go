package engine

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hypebeast/go-osc/osc"
)

type Patch struct {
	synth     *Synth
	name      string
	params    *ParamValues
	listeners []*PluginContext
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
		synth:     GetSynth("default"),
		name:      patchName,
		params:    NewParamValues(),
		listeners: []*PluginContext{},
	}
	LogOfType("patch", "NewPatch", "patch", patchName)
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
	return !strings.HasPrefix(name, "engine.")
}

func ApplyToAllPatchs(f func(patch *Patch)) {
	for _, patch := range Patchs {
		f(patch)
	}
}

func (patch *Patch) Synth() *Synth {
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
			err := patch.Set(nm, d.Init)
			if err != nil {
				LogError(err)
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

func (patch *Patch) RefreshAllIfPortnumMatches(ffglportnum int) error {
	portnum, _ := TheResolume().PortAndLayerNumForPatch(patch.name)
	if portnum != ffglportnum {
		return nil
	}
	return patch.AlertListeners("patch_refresh_all")
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
		if synth == nil {
			LogWarn("QuadPro. no synth, using default", "synth", paramValue)
			synth = GetSynth("default")
		}
		patch.synth = synth
	}
}

func (patch *Patch) AlertListeners(event string) error {

	LogOfType("listeners", "AlertListeners", "patch", patch.Name(), "event", event)
	args := map[string]string{
		"event": event,
		"patch": patch.Name(),
	}
	return patch.AlertListenersOf(args)
}

func (patch *Patch) AlertListenersOf(args map[string]string) error {
	var err error
	for _, listener := range patch.listeners {
		_, e := listener.api(listener, "event", args)
		if e != nil {
			LogError(e) // make sure multiple ones are all logged
			err = e
		}
	}
	return err
}

/*
func (patch *Patch) AlertListenersOfSet(paramName string, paramValue string) error {

	LogOfType("listeners", "AlertListenersOf", "param", paramName, "value", paramValue, "patch", patch.Name())
	args := map[string]string{
		"event": "patch_set",
		"name":  paramName,
		"value": paramValue,
		"patch": patch.Name(),
	}
	return patch.AlertListenersOf(args)
}
*/

func (patch *Patch) SaveAndAlert(category string, filename string) error {

	LogOfType("saved", "Patch.SaveAndAlert", "category", category, "filename", filename)

	if filename == "_Current" && category != "quad" {
		err := fmt.Errorf("Hey, we're only saving _Current in quad (and global)")
		LogError(err)
		return err
	}
	err := patch.params.Save(category, filename)
	if err != nil {
		LogError(err)
		return err
	}
	return patch.AlertListeners("patch_refresh_all")
}

func (patch *Patch) AddListener(ctx *PluginContext) {
	patch.listeners = append(patch.listeners, ctx)
}

func (patch *Patch) Name() string {
	return patch.name
}

func (patch *Patch) CursorToQuant(ce CursorEvent) Clicks {

	quantStyle := patch.GetString("misc.quantstyle")

	q := Clicks(1)
	switch quantStyle {

	case "none", "":
		// q is 1

	case "frets":
		if ce.Y > 0.85 {
			q = OneBeat / 8
		} else if ce.Y > 0.55 {
			q = OneBeat / 4
		} else if ce.Y > 0.25 {
			q = OneBeat / 2
		} else {
			q = OneBeat
		}

	case "fixed":
		q = OneBeat / 4

	case "pressure":
		if ce.Z > 0.20 {
			q = OneBeat / 8
		} else if ce.Z > 0.10 {
			q = OneBeat / 4
		} else if ce.Z > 0.05 {
			q = OneBeat / 2
		} else {
			q = OneBeat
		}

	default:
		LogWarn("Unrecognized quant", "quantstyle", quantStyle)
	}
	q = Clicks(float64(q) / TempoFactor)
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
		err := patch.Load(category, filename)
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
			return "", fmt.Errorf("executePatchAPI: err=%s", err)
		}
		err = patch.Set(name, value)
		if err != nil {
			return "", err
		}
		patch.noticeValueChange(name, value)
		return "", patch.SaveQuadAndAlert()

	case "setparams":
		var err error
		for name, value := range apiargs {
			e := patch.Set(name, value)
			if e != nil {
				LogError(e)
				err = e
			}
		}
		if err != nil {
			return "", err
		}
		return "", patch.SaveQuadAndAlert()

	case "get":
		name, ok := apiargs["name"]
		if !ok {
			return "", fmt.Errorf("executePatchAPI: missing name argument")
		}
		return patch.Get(name), nil

	case "clear":
		patch.clearGraphics()

		return "", nil

	default:
		// ignore errors on these for the moment
		if strings.HasPrefix(api, "loop_") || strings.HasPrefix(api, "midi_") {
			return "", nil
		}
		err := fmt.Errorf("Patch.API: unrecognized api=%s", api)
		LogError(err)
		return "", err
	}
}

func (patch *Patch) SaveQuadAndAlert() error {
	return patch.AlertListeners("quad_save_current")
}

func (patch *Patch) Set(paramName string, paramValue string) error {

	if !IsPerPatchParam(paramName) {
		err := fmt.Errorf("Patch.Set: not per-patch param=%s", paramName)
		LogError(err)
		return err
	}
	return patch.params.Set(paramName, paramValue)
}

// If no such parameter, return ""
func (patch *Patch) Get(paramName string) string {
	return patch.params.Get(paramName)
}

func (patch *Patch) ParamNames() []string {
	// Print the parameter values sorted by name
	fullNames := patch.params.values
	sortedNames := make([]string, 0, len(fullNames))
	for k := range fullNames {
		if IsPerPatchParam(k) {
			sortedNames = append(sortedNames, k)
		}
	}
	sort.Strings(sortedNames)
	return sortedNames
}

func (patch *Patch) GetString(paramName string) string {
	return patch.params.GetStringValue(paramName, "")
}

func (patch *Patch) GetInt(paramName string) int {
	return patch.params.GetIntValue(paramName)
}

func (patch *Patch) GetBool(paramName string) bool {
	return patch.params.GetBoolValue(paramName)
}

func (patch *Patch) GetFloat(paramName string) float32 {
	return patch.params.GetFloatValue(paramName)
}

func (patch *Patch) ApplyPatchValuesFromQuadMap(paramsmap map[string]any) error {

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
			LogError(fmt.Errorf("applyQuadValuesFrom: no PatchNameSeparator"), "param", fullParamName)
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
			LogError(fmt.Errorf("ApplyPatchValuesFromQuadMap: Needs to handle new value format"))
			return fmt.Errorf("value of name=%s isn't a string", fullParamName)
		}
		err := patch.Set(paramName, value)
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
			err := patch.Set(paramName, init)
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

// ReadableSavedFilePath xxx
func (patch *Patch) readableFilePath(category string, filename string) string {
	return SavedFilePath(category, filename)
}

func (patch *Patch) Load(category string, filename string) error {

	LogOfType("saved", "patch.Load", "patch", patch.Name(), "category", category, "filename", filename)

	// category, filename := SavedNameSplit(savedName)
	path := patch.readableFilePath(category, filename)
	paramsmap, err := LoadParamsMap(path)
	if err != nil {
		LogError(err)
		return err
	}
	if category == "quad" {
		// this will only load things to this one patch
		err = patch.ApplyPatchValuesFromQuadMap(paramsmap)
		if err != nil {
			LogError(err)
			return err
		}
	} else {
		patch.params.ApplyValuesFromMap(category, paramsmap)
	}

	// If there's a _override.json file, use it
	overrideFilename := "._override"
	overridepath := patch.readableFilePath(category, overrideFilename)
	if fileExists(overridepath) {
		LogOfType("saved", "applySaved using", "overridepath", overridepath)
		overridemap, err := LoadParamsMap(overridepath)
		if err != nil {
			return err
		}
		patch.params.ApplyValuesFromMap(category, overridemap)
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
		_, found := paramsmap[paramName]
		if !found {
			paramValue := def.Init
			err := patch.Set(paramName, paramValue)
			// err := params.Set(paramName, paramValue)
			if err != nil {
				LogWarn("patch.Set error", "saved", paramName, "err", err)
				// Don't fail completely
			}
		}
	}
	return patch.AlertListeners("patch_refresh_all")
}

func (patch *Patch) clearGraphics() {
	// send an OSC message to Resolume
	TheResolume().ToFreeFramePlugin(patch.Name(), osc.NewMessage("/clear"))
}
