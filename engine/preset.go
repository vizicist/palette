package engine

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Preset struct {
	category string
	filename string
	params   *ParamValues
}

var Presets = make(map[string]*Preset)

func GetPreset(name string) *Preset {
	p, ok := Presets[name]
	if !ok {
		category, filename := PresetNameSplit(name)
		p = &Preset{
			category: category,
			filename: filename,
			params:   &ParamValues{},
		}
		Presets[name] = p
	}
	return p
}

func (p *Preset) loadQuadPreset(applyToRegion string) error {

	path := p.readableFilePath()
	paramsmap, err := LoadParamsMap(path)
	if err != nil {
		return err
	}

	log.Printf("loadQuadPreset: path=%s\n", path)

	// Here's where the params get applied,
	// which among other things
	// may result in sending OSC messages out.
	for name, ival := range paramsmap {
		value, ok := ival.(string)
		if !ok {
			return fmt.Errorf("value of name=%s isn't a string", name)
		}
		// In a quad file, the parameter names are of the form:
		// {region}-{parametername}
		words := strings.SplitN(name, "-", 2)
		regionOfParam := words[0]
		motor := MotorForRegion(regionOfParam)
		if motor == nil {
			return fmt.Errorf("Preset.loadQuadPreset: no region named %s", regionOfParam)
		}
		if applyToRegion != "*" && applyToRegion != regionOfParam {
			continue
		}
		// use words[1] so the motor doesn't see the region name
		parameterName := words[1]
		// We expect the parameter to be of the form
		// {category}.{parameter}, but old "quad" files
		// didn't include the category.
		if !strings.Contains(parameterName, ".") {
			log.Printf("loadQuadPreset: path=%s parameter=%s is in OLD format, not supported", path, parameterName)
			return fmt.Errorf("")
		}
		err = motor.SetOneParamValue(parameterName, value)
		if err != nil {
			if !OldParameterName(parameterName) {
				log.Printf("loadQuadPreset: name=%s err=%s\n", parameterName, err)
			}
			// Don't fail completely on individual failures,
			// some might be for parameters that no longer exist.
		}
	}

	// For any parameters that are in Paramdefs but are NOT in the loaded
	// preset, we put out the "init" values.  This happens when new parameters
	// are added which don't exist in existing preset files.
	// This is similar to code in Motor.applyPreset, except we
	// have to do it for all for pads
	for _, c := range TheEngine.Router.regionLetters {
		padName := string(c)
		motor := MotorForRegion(padName)
		for nm, def := range ParamDefs {
			paramName := string(padName) + "-" + nm
			_, found := paramsmap[paramName]
			if !found {
				init := def.Init
				err = motor.SetOneParamValue(nm, init)
				if err != nil {
					// a hack to eliminate errors on a parameter that
					// still exists in some presets.
					if !OldParameterName(nm) {
						log.Printf("loadQuadPreset: %s, param=%s, init=%s, err=%s\n", path, nm, init, err)
					}
					// Don't fail completely on individual failures,
					// some might be for parameters that no longer exist.
				}
			}
		}
	}

	return nil
}

// ReadablePresetFilePath xxx
func (p *Preset) readableFilePath() string {
	return p.presetFilePath()
}

// WritablePresetFilePath xxx
func (p *Preset) WriteableFilePath() string {
	path := p.presetFilePath()
	os.MkdirAll(filepath.Dir(path), 0777)
	return path
}

func PresetsDir() string {
	return "presets"
}

// presetFilePath returns the full path of a preset file.
func (p *Preset) presetFilePath() string {
	jsonfile := p.filename + ".json"
	localpath := filepath.Join(PaletteDataPath(), PresetsDir(), p.category, jsonfile)
	return localpath
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
		// log.Printf("Crawling: %#v\n", path)
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
		log.Printf("filepath.Walk: err=%s\n", err)
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

func LoadParamsMap(path string) (map[string]interface{}, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f interface{}
	err = json.Unmarshal(bytes, &f)
	if err != nil {
		return nil, fmt.Errorf("unable to Unmarshal path=%s, err=%s", path, err)
	}
	toplevel, ok := f.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unable to convert params to map[string]interface{}")

	}
	params, okparams := toplevel["params"]
	if !okparams {
		return nil, fmt.Errorf("no params value in json")
	}
	paramsmap, okmap := params.(map[string]interface{})
	if !okmap {
		return nil, fmt.Errorf("params value is not a map[string]string in jsom")
	}
	return paramsmap, nil
}
