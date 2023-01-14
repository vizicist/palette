package engine

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hypebeast/go-osc/osc"
)

type Patch struct {
	Synth     *Synth
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
		Synth:     GetSynth("default"),
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
	return !strings.HasPrefix(name, "global.")
}

func ApplyToAllPatchs(f func(patch *Patch)) {
	for _, patch := range Patchs {
		f(patch)
	}
}

// SetDefaultValues xxx
func (patch *Patch) SetDefaultValues() {
	for nm, d := range ParamDefs {
		if IsPerPatchParam(nm) {
			err := patch.SetNoAlert(nm, d.Init)
			// err := patch.params.SetParamValueWithString(nm, d.Init)
			if err != nil {
				LogError(err)
			}
		}
	}
}

func (patch *Patch) MIDIChannel() uint8 {
	return patch.Synth.Channel()
}

// func (patch *Patch) ReAlertListenersOnAllParameters() {
// 	patch.AlertListenersToRefreshAll()
// }

func (patch *Patch) RefreshAllIfPortnumMatches(ffglportnum int) {
	portnum, _ := TheResolume().PortAndLayerNumForPatch(patch.name)
	if portnum == ffglportnum {
		patch.AlertListenersToRefreshAll()
	}
}

func (patch *Patch) AlertListenersOfSet(paramName string, paramValue string) {

	LogOfType("listeners", "AlertListenersOf", "param", paramName, "value", paramValue, "patch", patch.Name())
	for _, listener := range patch.listeners {
		args := map[string]string{
			"event": "notification_of_patch_set",
			"name":  paramName,
			"value": paramValue,
			"patch": patch.Name(),
		}
		_, err := listener.api(listener, "event", args)
		if err != nil {
			LogError(err)
		}
	}
}

func (patch *Patch) AlertListenersToRefreshAll() {

	LogOfType("listeners", "AlertListenersToRefreshAll", "patch", patch.Name())
	for _, listener := range patch.listeners {
		args := map[string]string{
			"event": "notification_of_patch_refresh_all",
			"patch": patch.Name(),
		}
		_, err := listener.api(listener, "event", args)
		if err != nil {
			LogError(err)
		}
	}
}

func (patch *Patch) AlertListenersOnSave() {

	LogOfType("listeners", "AlertListenersOnSave")
	for _, listener := range patch.listeners {
		args := map[string]string{
			"event": "alertonsave",
		}
		_, err := listener.api(listener, "event", args)
		if err != nil {
			LogError(err)
		}
	}
}

func (patch *Patch) AddListener(ctx *PluginContext) {
	patch.listeners = append(patch.listeners, ctx)
}

func (patch *Patch) Name() string {
	return patch.name
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

		err := patch.Load(category, filename)
		if err != nil {
			return "", err
		}
		patch.SaveCurrent()
		return "", nil

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
		// Doing SaveCurrent here might be superfluous, since any
		// other change to a patch's settings should already invoke SaveCurrent.
		patch.SaveCurrent()
		return "", patch.params.Save(category, filename)

	case "set":
		name, value, err := GetNameValue(apiargs)
		if err != nil {
			return "", fmt.Errorf("executePatchAPI: err=%s", err)
		}
		err = patch.SetAndAlert(name, value)
		patch.SaveCurrent()
		return "", err

	case "setparams":
		var err error
		for name, value := range apiargs {
			e := patch.SetNoAlert(name, value)
			if e != nil {
				err = e
			}
		}
		patch.AlertListenersToRefreshAll()
		patch.SaveCurrent()
		return "", err

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

func (patch *Patch) SetNoAlert(paramName string, paramValue string) error {
	return patch.set(paramName, paramValue, false)
}

func (patch *Patch) SetAndAlert(paramName string, paramValue string) error {
	return patch.set(paramName, paramValue, true)
}

func (patch *Patch) set(paramName string, paramValue string, alert bool) error {

	if !IsPerPatchParam(paramName) {
		err := fmt.Errorf("Patch.Set: not per-patch param=%s", paramName)
		LogError(err)
		return err
	}
	err := patch.params.Set(paramName, paramValue)
	if err != nil {
		return err
	}
	if alert {
		patch.AlertListenersOfSet(paramName, paramValue)
	}
	return nil
}

// If no such parameter, return ""
func (patch *Patch) Get(paramName string) string {
	return patch.params.Get(paramName)
}

// func (patch *Patch) Params() *ParamValues {
// 	return patch.params
// }

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

func (patch *Patch) SaveCurrent() {
	patch.AlertListenersOnSave()
	// We don't really need to save patch._Current because
	// quad._Current is being saved and restored.
	// However, if there's only one Patch, patch._Current may be needed.
	//  patch.params.Save("patch", "_Current")
}

func (patch *Patch) ApplyQuadValuesFrom(paramsmap map[string]any) error {

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
			return fmt.Errorf("value of name=%s isn't a string", fullParamName)
		}
		err := patch.SetNoAlert(paramName, value)
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
			err := patch.SetNoAlert(paramName, init)
			if err != nil {
				// a hack to eliminate errors on a parameter that still exists somewhere
				LogWarn("applyQuadValuesFrom", "nm", paramName, "err", err)
				// Don't fail completely on individual failures,
				// some might be for parameters that no longer exist.
			}
		}
	}

	// patch.AlertListenersToRefreshAll()

	return nil
}

// ReadableSavedFilePath xxx
func (patch *Patch) readableFilePath(category string, filename string) string {
	return SavedFilePath(category, filename)
}

func (patch *Patch) Load(category string, filename string) error {

	// category, filename := SavedNameSplit(savedName)
	path := patch.readableFilePath(category, filename)
	paramsmap, err := LoadParamsMap(path)
	if err != nil {
		LogError(err)
		return err
	}
	// params := patch.params
	if category == "quad" {
		// this will only load things to this one patch
		err = patch.ApplyQuadValuesFrom(paramsmap)
		if err != nil {
			LogError(err)
			return err
		}
	} else {
		patch.params.ApplyParamsTo(category, paramsmap)
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
		patch.params.ApplyParamsTo(category, overridemap)
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
			err := patch.SetNoAlert(paramName, paramValue)
			// err := params.Set(paramName, paramValue)
			if err != nil {
				LogWarn("patch.Set error", "saved", paramName, "err", err)
				// Don't fail completely
			}
		}
	}
	patch.AlertListenersToRefreshAll()
	return nil
}

func (patch *Patch) clearGraphics() {
	// send an OSC message to Resolume
	TheResolume().ToFreeFramePlugin(patch.Name(), osc.NewMessage("/clear"))
}
