package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// ParamDef is a single parameter definition.
type ParamDef struct {
	TypedParamDef any
	Category      string
	Init          string
	comment       string
}

type ParamDefFloat struct {
	min  float32
	max  float32
	Init string
	// comment string
}

type ParamDefInt struct {
	min int
	max int
}

type ParamDefBool struct {
}

type ParamDefString struct {
	values []string
}

type ParamsMap map[string]any

// ParamDefs is the set of all parameter definitions
var ParamDefs map[string]ParamDef

// ParamValue is a single parameter value
// which could be any of the param*Value types
type ParamValue any

type paramValString struct {
	def   ParamDefString
	value string
}

type paramValInt struct {
	def   ParamDefInt
	value int
}

type paramValFloat struct {
	def   ParamDefFloat
	value float32
}

type paramValBool struct {
	def   ParamDefBool
	value bool
}

// ParamValues is the set of all parameter values
type ParamValues struct {
	mutex  sync.RWMutex
	values map[string]ParamValue
}

var GlobalParams *ParamValues

func NewParamValues() *ParamValues {
	// Note: it's ParamValue (not a pointer)
	return &ParamValues{values: map[string]ParamValue{}}
}

func InitParams() {
	err := LoadParamEnums()
	if err != nil {
		LogWarn("LoadParamEnums", "err", err)
		// might be fatal, but try to continue
	}

	err = LoadParamDefs()
	if err != nil {
		LogWarn("LoadParamDefs", "err", err)
		// might be fatal, but try to continue
	}
	GlobalParams = NewParamValues()
}

func (vals *ParamValues) DoForAllParams(f func(string, ParamValue)) {
	vals.mutex.RLock()
	defer vals.mutex.RUnlock()
	for nm, val := range vals.values {
		f(nm, val)
	}
}

func (vals *ParamValues) JsonValues() string {
	vals.mutex.RLock()
	defer vals.mutex.RUnlock()
	s := ""
	sep := ""
	for nm := range vals.values {
		valstr, _ := vals.paramValueAsString(nm) // error shouldn't happen
		s = s + sep + "        \"" + nm + "\":\"" + valstr + "\""
		sep = ",\n"
	}
	return s
}

// Currently, no errors are ever returned, but log messages are generated.
func (params *ParamValues) ApplyValuesFromMap(category string, paramsmap map[string]any, setfunc func(string, string) error) {

	for fullname, ival := range paramsmap {
		var value string
		value, ok := ival.(string)
		if !ok {
			// map value format is like {"value": "0.5", "enabled": "true", ...}
			mapval, ok := ival.(map[string]interface{})
			if !ok {
				LogWarn("value isn't a string or map in params json", "name", fullname, "value", ival)
				continue
			}
			value, ok = mapval["value"].(string)
			if !ok {
				LogWarn("No value entry in mapval", "name", fullname, "mapval", mapval)
				continue
			}
			LogInfo("New value format", "name", fullname, "value", value)
		}
		paramCategory, _ := SavedNameSplit(fullname)

		// Only include ones that match the category.
		// If the category is "patch" or "quad", match any of sound/visual/effect/misc.

		if category == paramCategory ||
			((category == "patch" || category == "quad") && IsPerPatchParam(fullname)) {

			// err := params.Set(fullname, value)
			err := setfunc(fullname, value)
			if err != nil {
				LogIfError(err)
				// Don't abort the whole load, i.e. we are tolerant
				// of unknown parameters or errors in the saved
			}
		}
	}
}

func (vals *ParamValues) ParamNames() []string {
	// Print the parameter values sorted by name
	vals.mutex.RLock()
	defer vals.mutex.RUnlock()
	sortedNames := make([]string, 0, len(vals.values))
	for k := range vals.values {
		if IsPerPatchParam(k) {
			sortedNames = append(sortedNames, k)
		}
	}
	sort.Strings(sortedNames)
	return sortedNames
}

// returns "" if parameter doesn't exist
func (vals *ParamValues) Get(name string) (string, error) {
	if !strings.Contains(name, ".") {
		return "", fmt.Errorf("parameters should always have a period, name=%s", name)
	}
	return vals.paramValueAsString(name)
}

func (vals *ParamValues) GetStringValue(name string, def string) string {
	val := vals.paramValue(name)
	if val == nil {
		return def
	}
	return val.(paramValString).value
}

func (vals *ParamValues) GetIntValue(name string) int {
	param := vals.paramValue(name)
	if param == nil {
		// Warn("No existing int value for param", "name", name)
		return 0
	}
	return param.(paramValInt).value
}

func (vals *ParamValues) GetFloatValue(name string) float32 {
	param := vals.paramValue(name)
	if param == nil {
		// Warn("No existing float value for param", "name", name)
		pd, ok := ParamDefs[name]
		if ok {
			f, err := strconv.ParseFloat(pd.Init, 64)
			if err == nil {
				LogIfError(err)
				return float32(f)
			}
		}
		return 0.0
	}
	f := (param).(paramValFloat).value
	return f
}

func (vals *ParamValues) GetBoolValue(name string) bool {
	param := vals.paramValue(name)
	if param == nil {
		// Warn("No existing paramvalue for", "name", name)
		return false
	}
	return (param).(paramValBool).value
}

func (vals *ParamValues) Save(category string, filename string) error {

	LogOfType("saved", "ParamValues.Save", "category", category, "filename", filename)

	path, err := WritableSavedFilePath(category, filename, ".json")
	if err != nil {
		LogIfError(err)
		return err
	}

	s := "{\n    \"params\": {\n"

	// Print the parameter values sorted by name
	fullNames := vals.values
	sortedNames := make([]string, 0, len(fullNames))
	// lookfor := category + "."
	for paramName := range fullNames {
		w := strings.SplitN(paramName, ".", 2)
		paramCategory := w[0]
		// Decide if this parameter should be included in the file
		if category == paramCategory {
			sortedNames = append(sortedNames, paramName)
		} else {
			if category == "patch" && IsPatchCategory(paramCategory) {
				sortedNames = append(sortedNames, paramName)
			}
		}
	}
	sort.Strings(sortedNames)

	sep := ""
	for _, fullName := range sortedNames {
		// The names are of the form "category.name",
		// and any parameters with a name starting with "_" are not saved.
		if strings.Contains(fullName, "._") {
			continue
		}
		valstring, e := vals.paramValueAsString(fullName)
		if e != nil {
			LogIfError(e)
			continue
		}
		s += fmt.Sprintf("%s        \"%s\":\"%s\"", sep, fullName, valstring)
		sep = ",\n"
	}
	s += "\n    }\n}"
	data := []byte(s)
	return os.WriteFile(path, data, 0644)
}

func IsPatchCategory(category string) bool {
	return (category == "visual" ||
		category == "sound" ||
		category == "effect" ||
		category == "misc")
}

func LoadParamsMapOfCategory(category string, filename string) (ParamsMap, error) {
	path, err := ReadableSavedFilePath(category, filename, ".json")
	if err != nil {
		LogIfError(err)
		return nil, err
	}
	paramsmap, err := LoadParamsMapFromPath(path)
	if err != nil {
		LogIfError(err)
		return nil, err
	}
	return paramsmap, nil
}

func LoadParamsMapFromPath(path string) (ParamsMap, error) {
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

func LoadParamsMapFromString(s string) (ParamsMap, error) {
	var f any
	err := json.Unmarshal([]byte(s), &f)
	if err != nil {
		return nil, fmt.Errorf("unable to Unmarshal, err=%s, s=%s", err, s)
	}
	toplevel, ok := f.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unable to convert to ParamsMap - %s", s)

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

func (vals *ParamValues) SetParamWithString(origname, value string) (err error) {

	vals.mutex.Lock()
	defer vals.mutex.Unlock()

	if origname == "pad" {
		return fmt.Errorf("ParamValues.Set rejects setting of pad value")
	}

	def, err := vals.paramDefOf(origname)
	if err != nil {
		return err
	}

	var paramVal ParamValue
	switch d := def.TypedParamDef.(type) {
	case ParamDefInt:
		valint, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		paramVal = paramValInt{def: d, value: valint}
	case ParamDefBool:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		paramVal = paramValBool{def: d, value: v}
	case ParamDefString:
		paramVal = paramValString{def: d, value: value}
	case ParamDefFloat:
		var v float32
		v, err := ParseFloat32(value, origname)
		if err != nil {
			return err
		}
		paramVal = paramValFloat{def: d, value: float32(v)}
	default:
		e := fmt.Errorf("ParamValues.Set: unknown TypedParamDef for name=%s type=%T", origname, def.TypedParamDef)
		LogIfError(e)
		return e
	}
	vals.values[origname] = paramVal
	return nil
}

// ParamEnums contains the lists of enumerated values for string parameters
var ParamEnums map[string][]string

// ParseFloat32 xxx
func ParseFloat32(s string, name string) (float32, error) {
	f, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0.0, fmt.Errorf("ParseFloat32 of parameter '%s' (%s) fails", name, s)
	}
	return float32(f), nil
}

// ParseInt xxx
func ParseInt(s string, name string) (int, error) {
	f, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("parseInt of parameter '%s' (%s) fails", name, s)
	}
	return int(f), nil
}

// ParseBool xxx
func ParseBool(s string, name string) (bool, error) {
	f, err := strconv.ParseBool(s)
	if err != nil {
		return false, fmt.Errorf("parseBool of parameter '%s' (%s) fails", name, s)
	}
	return f, nil
}

// LoadParamEnums initializes the list of enumerated parameter values
func LoadParamEnums() error {

	ParamEnums = make(map[string][]string)

	path := ConfigFilePath("paramenums.json")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("loadParamEnums: unable to read path=%s", path)
	}
	var f any
	err = json.Unmarshal(bytes, &f)
	if err != nil {
		return fmt.Errorf("loadParamEnums: unable to Unmarshal path=%s", path)
	}
	toplevel := f.(map[string]any)

	for enumName, enumList := range toplevel {
		var enums []string
		for _, e := range enumList.([]any) {
			enums = append(enums, e.(string))
		}
		ParamEnums[enumName] = enums
	}
	return nil
}

// LoadParamDefs initializes the list of parameters
func LoadParamDefs() error {

	ParamDefs = make(map[string]ParamDef)

	path := ConfigFilePath("paramdefs.json")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("unable to read %s, err=%s", path, err)
	}
	var f any
	err = json.Unmarshal(bytes, &f)
	if err != nil {
		return fmt.Errorf("unable to Unmarshal %s, err=%s", path, err)
	}
	params := f.(map[string]any)
	for name, dat := range params {
		w := strings.SplitN(name, ".", 2)
		if len(w) != 2 {
			return fmt.Errorf("LoadParamDefs: parameter has no category - %s", name)
		}
		category := w[0]
		jmap := dat.(map[string]any)
		min := jmap["min"].(string)
		max := jmap["max"].(string)
		valuetype := jmap["valuetype"].(string)

		pd := ParamDef{
			Category: category,
			Init:     jmap["init"].(string),
			comment:  jmap["comment"].(string),
		}

		switch valuetype {
		case "double", "float":
			fmin, err := ParseFloat32(min, "min")
			if err != nil {
				return err
			}
			fmax, err := ParseFloat32(max, "max")
			if err != nil {
				return err
			}
			pd.TypedParamDef = ParamDefFloat{
				min: fmin,
				max: fmax,
			}

		case "int":
			imin, err := ParseInt(min, "min")
			if err != nil {
				return err
			}
			imax, err := ParseInt(max, "max")
			if err != nil {
				return err
			}
			pd.TypedParamDef = ParamDefInt{
				min: imin,
				max: imax,
			}

		case "bool":
			pd.TypedParamDef = ParamDefBool{}

		case "string":
			// A bit of a hack - the "min" value of a
			// string parameter definition
			// is actually an "enum" type name
			enumName := min
			values := ParamEnums[enumName]
			pd.TypedParamDef = ParamDefString{
				values: values,
			}
		}

		if category == "effect" {
			// For effect parameters, the list only has
			// one instance of each Freeframe plugin, but
			// the Resolume configuration has 2 instances
			// of each plugin.
			name = strings.TrimPrefix(name, "effect.")
			ParamDefs["effect.1-"+name] = pd
			ParamDefs["effect.2-"+name] = pd
		} else {
			ParamDefs[name] = pd
		}

	}
	return nil
}

func (vals *ParamValues) paramValue(name string) ParamValue {
	vals.mutex.RLock()
	val, ok := vals.values[name]
	vals.mutex.RUnlock()
	if !ok {
		return nil
	}
	return val
}

func (vals *ParamValues) Exists(name string) bool {
	_, exists := vals.values[name]
	return exists
}

func (vals *ParamValues) paramValueAsString(name string) (string, error) {
	val := vals.paramValue(name)
	if val == nil {
		return "", fmt.Errorf("no parameter named %s", name)
	}
	s := ""
	switch v := val.(type) {
	case paramValString:
		s = v.value
	case paramValInt:
		s = fmt.Sprintf("%d", v.value)
	case paramValFloat:
		s = fmt.Sprintf("%f", v.value)
	case paramValBool:
		s = fmt.Sprintf("%v", v.value)
	default:
		s = "BADVALUETYPE"
	}
	return s, nil
}

func (vals *ParamValues) paramDefOf(name string) (ParamDef, error) {
	p, ok := ParamDefs[name]
	if !ok {
		return ParamDef{}, fmt.Errorf("paramDefOf: no parameter named %s", name)
	} else {
		return p, nil
	}
}

var overrideMap ParamsMap

func OverrideMap() ParamsMap {
	if overrideMap == nil {
		// If there's a _override.json file, use it
		overridepath := ConfigFilePath("paramoverrides.json")
		if fileExists(overridepath) {
			LogOfType("params", "Reading Overridemap", "overridepath", overridepath)
			m, err := LoadParamsMapFromPath(overridepath)
			if err != nil {
				LogError(err)
			} else {
				overrideMap = m
			}
		}
	}
	return overrideMap
}
