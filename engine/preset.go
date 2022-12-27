package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

/*
type OldPreset struct {
	Category  string
	filename  string
	paramsmap map[string]any
}

var Presets = make(map[string]*OldPreset)

func GetPreset(name string) *OldPreset {

	p, ok := Presets[name]
	if !ok {
		category, filename := PresetNameSplit(name)
		p = &OldPreset{
			Category:  category,
			filename:  filename,
			paramsmap: make(map[string]any),
		}
		Presets[name] = p
	}
	return p
}
*/

/*
func LoadPreset(name string) (*Preset, error) {
	p := GetPreset(name)
	err := p.loadPreset()
	if err != nil {
		return nil, err
	}
	return p, nil
}
*/

func ApplyParamsMap(presetType string, paramsmap map[string]any, params *ParamValues) error {

	// Currently, no errors are ever returned, but log messages are generated.

	for name, ival := range paramsmap {
		val, okval := ival.(string)
		if !okval {
			LogWarn("value isn't a string in params json", "name", name, "value", val)
			continue
		}
		fullname := name
		thisCategory, _ := PresetNameSplit(fullname)
		// Only include ones that match the presetType
		if presetType != "snap" && thisCategory != presetType {
			continue
		}
		// This is where the parameter values get applied,
		// which may trigger things (like sending OSC)
		err := params.Set(fullname, val)
		if err != nil {
			LogError(err)
			// Don't abort the whole load, i.e. we are tolerant
			// of unknown parameters or errors in the preset
		}
	}
	return nil
}

func PresetsDir() string {
	return "presets"
}

func PresetNameSplit(preset string) (string, string) {
	words := strings.SplitN(preset, ".", 2)
	if len(words) == 1 {
		return "", words[0]
	} else {
		return words[0], words[1]
	}
}

// PresetMap returns a map of preset names to file paths
func PresetMap(wantCategory string) (map[string]string, error) {

	result := make(map[string]string, 0)

	walker := func(walkedpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		// Only look at .json files
		if !strings.HasSuffix(walkedpath, ".json") {
			return nil
		}
		path := strings.TrimSuffix(walkedpath, ".json")
		// the last two components of the path are category and preset
		thisCategory := ""
		thisPreset := ""
		lastslash2 := -1
		lastslash := strings.LastIndex(path, "\\")
		if lastslash >= 0 {
			thisPreset = path[lastslash+1:]
			path2 := path[0:lastslash]
			lastslash2 = strings.LastIndex(path2, "\\")
			if lastslash2 >= 0 {
				thisCategory = path2[lastslash2+1:]
			}
		}
		if wantCategory == "*" || thisCategory == wantCategory {
			result[thisCategory+"."+thisPreset] = walkedpath
		}
		return nil
	}

	presetsDir1 := filepath.Join(PaletteDataPath(), PresetsDir())
	err := filepath.Walk(presetsDir1, walker)
	if err != nil {
		LogWarn("filepath.Walk", "err", err)
		return nil, err
	}
	return result, nil
}

// PresetArray returns a list of preset filenames, wantCategory can be "*"
func PresetArray(wantCategory string) ([]string, error) {

	presetMap, err := PresetMap(wantCategory)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0)
	for name := range presetMap {
		result = append(result, name)
	}
	return result, nil
}

func PresetList(apiargs map[string]string) (string, error) {

	wantCategory := optionalStringArg("category", apiargs, "*")
	result := "["
	sep := ""

	presetMap, err := PresetMap(wantCategory)
	if err != nil {
		return "", err
	}
	for name := range presetMap {
		thisCategory, _ := PresetNameSplit(name)
		if wantCategory == "*" || thisCategory == wantCategory {
			result += sep + "\"" + name + "\""
			sep = ","
		}
	}
	result += "]"
	return result, nil
}

func LoadParamsMap(path string) (map[string]any, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f any
	err = json.Unmarshal(bytes, &f)
	if err != nil {
		return nil, fmt.Errorf("unable to Unmarshal path=%s, err=%s", path, err)
	}
	toplevel, ok := f.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unable to convert params to map[string]any")

	}
	params, okparams := toplevel["params"]
	if !okparams {
		return nil, fmt.Errorf("no params value in json")
	}
	paramsmap, okmap := params.(map[string]any)
	if !okmap {
		return nil, fmt.Errorf("params value is not a map[string]string in jsom")
	}
	return paramsmap, nil
}

// PresetFilePath returns the full path of a preset file.
func PresetFilePath(category string, filename string) string {
	jsonfile := filename + ".json"
	localpath := filepath.Join(PaletteDataPath(), PresetsDir(), category, jsonfile)
	return localpath
}
