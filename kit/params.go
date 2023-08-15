package kit

import (
	"encoding/json"
	"fmt"
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

// These Params are global to the kit package
var Params *ParamValues

func NewParamValues() *ParamValues {
	// Note: it's ParamValue (not a pointer)
	return &ParamValues{values: map[string]ParamValue{}}
}

func InitParams() {
	Params = NewParamValues()

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

func (vals *ParamValues) Set(name, value string) error {
	return vals.SetParamValueWithString(name, value)
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

func SavedNameSplit(saved string) (string, string) {
	words := strings.SplitN(saved, ".", 2)
	if len(words) == 1 {
		return "", words[0]
	} else {
		return words[0], words[1]
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

func GetParam(nm string) (string, error) {
	s, err := Params.Get(nm)
	if err != nil {
		LogError(err) // may duplicate errors in the users of this func
		s = ""
	}
	return s, err
}

func GetParamInt(nm string) (int, error) {
	s, err := GetParam(nm)
	if err != nil {
		return 0, err
	}
	var val int
	nfound, err := fmt.Sscanf(s, "%d", &val)
	if err != nil || nfound == 0 {
		return 0, fmt.Errorf("bad format of integer parameter name=%s", nm)
	}
	return val, nil
}

// IsTrueValue returns true if the value is some version of true, and false otherwise.
func IsTrueValue(value string) bool {
	switch value {
	case "True":
		return true
	case "true":
		return true
	case "1":
		return true
	case "on":
		return true
	case "False":
		return false
	case "false":
		return false
	case "0":
		return false
	case "off":
		return false
	default:
		LogIfError(fmt.Errorf("IsTrueValue: invalid boolean value (%s), assuming false", value))
		return false
	}
}

func GetParamBool(nm string) (bool, error) {
	v, err := GetParam(nm)
	if err != nil {
		return false, err
	} else {
		return IsTrueValue(v), nil
	}
}

func GetParamFloat(nm string) (float64, error) {
	s, err := GetParam(nm)
	if err != nil {
		return 0.0, err
	}
	f, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0.0, fmt.Errorf("bad format of float parameter name=%s", nm)
	}
	return f, nil
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

// Save parameters values to a file.
// If filename doesn't have a .json suffix, it's added.
func (vals *ParamValues) Save(category string, filename string) error {
	data := vals.persistentDataOf(category)
	if ! strings.HasSuffix(filename,".json") {
		filename += ".json"
	}
	return TheHost.SaveDataInFile(data, category, filename)
}

func (vals *ParamValues) persistentDataOf(category string) (data []byte) {

	LogOfType("saved", "ParamValues.Save", "category", category)

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
	return []byte(s)
}

func IsPatchCategory(category string) bool {
	return (category == "visual" ||
		category == "sound" ||
		category == "effect" ||
		category == "misc")
}

func LoadParamsMap(bytes []byte) (ParamsMap, error) {
	var f any
	err := json.Unmarshal(bytes, &f)
	if err != nil {
		return nil, fmt.Errorf("unable to Unmarshal, err=%s", err)
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

func (vals *ParamValues) SetParamValueWithString(origname, value string) (err error) {

	vals.mutex.Lock()
	defer vals.mutex.Unlock()

	if origname == "pad" {
		return fmt.Errorf("ParamValues.SetParamValueWithString rejects setting of pad value")
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
		e := fmt.Errorf("SetParamValueWithString: unknown type of ParamDef for name=%s", origname)
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

	bytes, err := TheHost.GetConfigFileData("paramenums.json")
	if err != nil {
		return err
	}
	var f any
	err = json.Unmarshal(bytes, &f)
	if err != nil {
		return err
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

	bytes, err := TheHost.GetConfigFileData("paramdefs.json")
	if err != nil {
		return err
	}
	var f any
	err = json.Unmarshal(bytes, &f)
	if err != nil {
		return err
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
		bytes, err := TheHost.GetConfigFileData("paramoverrides.json")
		if err == nil {
			m, err := LoadParamsMap(bytes)
			if err != nil {
				LogError(err)
			} else {
				overrideMap = m
			}
		}
	}
	return overrideMap
}
