package kit

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	json "github.com/goccy/go-json"
)

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
	value float64
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

	// Set all the default global.* values
	for nm, pd := range ParamDefs {
		if pd.Category == "global" {
			err := GlobalParams.SetParamWithString(nm, pd.Init)
			if err != nil {
				LogError(err)
			}
		}
	}
}

func GetGlobalParams() map[string]string {
	s := map[string]string{}
	GlobalParams.DoForAllParams(func(nm string, val ParamValue) {
		s[nm], _ = GlobalParams.ParamValueAsString(nm)
	})
	return s
}

func (vals *ParamValues) DoForAllParams(f func(string, ParamValue)) {
	vals.mutex.RLock()
	defer vals.mutex.RUnlock()
	for nm, val := range vals.values {
		f(nm, val)
	}
}

func (vals *ParamValues) JSONValues() string {
	vals.mutex.RLock()
	defer vals.mutex.RUnlock()
	s := ""
	sep := ""
	for nm := range vals.values {
		valstr, _ := vals.ParamValueAsString(nm) // error shouldn't happen
		s = s + sep + "        \"" + nm + "\":\"" + valstr + "\""
		sep = ",\n"
	}
	return s
}

// ApplyValuesFromMap - Currently, no errors are ever returned, but log messages are generated.
func (vals *ParamValues) ApplyValuesFromMap(category string, paramsmap map[string]any, setfunc func(string, string) error) {

	for fullname, ival := range paramsmap {
		var value string
		value, ok := ival.(string)
		if !ok {
			// map value format is like {"value": "0.5", "enabled": "true", ...}
			mapval, ok := ival.(map[string]any)
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
		if paramCategory == "global" {
			fullname = canonicalGlobalParamName(fullname)
		}

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

// Get - returns "" if parameter doesn't exist
func (vals *ParamValues) Get(name string) (string, error) {
	if !strings.Contains(name, ".") {
		return "", fmt.Errorf("parameters should always have a period, name=%s", name)
	}
	return vals.ParamValueAsString(name)
}

// GetWithPrefix returns all parameters that start with the given prefix
// The return value is newline-separated list of "name = value" strings.
func (vals *ParamValues) GetWithPrefix(prefix string) (string, error) {
	vals.mutex.RLock()
	defer vals.mutex.RUnlock()

	LogOfType("params", "params GetWithPrefix", "prefix", prefix)
	// Collect matching names and sort them
	var names []string
	for name := range vals.values {
		if strings.HasPrefix(name, prefix) {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	// Build result string
	var result strings.Builder
	for _, name := range names {
		valstr, err := vals.ParamValueAsString(name)
		if err != nil {
			continue
		}
		result.WriteString(name)
		result.WriteString("=")
		result.WriteString(valstr)
		result.WriteString("\n")
	}
	return strings.TrimSuffix(result.String(), "\n"), nil
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

func (vals *ParamValues) GetFloatValue(name string) float64 {
	param := vals.paramValue(name)
	if param == nil {
		// Warn("No existing float value for param", "name", name)
		pd, ok := ParamDefs[name]
		if ok {
			f, err := strconv.ParseFloat(pd.Init, 64)
			if err == nil {
				return f
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

	params := make(map[string]string, len(sortedNames))
	for _, fullName := range sortedNames {
		// The names are of the form "category.name",
		// and any parameters with a name starting with "_" are not saved.
		if strings.Contains(fullName, "._") {
			continue
		}
		valstring, e := vals.ParamValueAsString(fullName)
		if e != nil {
			LogIfError(e)
			continue
		}
		params[fullName] = valstring
	}
	data, err := json.MarshalIndent(struct {
		Params map[string]string `json:"params"`
	}{Params: params}, "", "    ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

func IsPatchCategory(category string) bool {
	return (category == "visual" ||
		category == "sound" ||
		category == "effect" ||
		category == "stepper" ||
		category == "misc")
}

// LoadBootParamValues loads the global "_Boot" preset into a fresh ParamValues.
func LoadBootParamValues() (*ParamValues, error) {
	paramsMap, err := LoadParamsMapOfCategory("global", "_Boot")
	if err != nil {
		return nil, err
	}
	params := NewParamValues()
	params.ApplyValuesFromMap("global", paramsMap, params.SetParamWithString)
	return params, nil
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
	pmap, err := MakeParamsMapFromBytes(bytes)
	if err != nil {
		return nil, fmt.Errorf("unable to load params from path=%s err=%w", path, err)
	}
	return pmap, nil
}

func MakeParamsMapFromBytes(bytes []byte) (ParamsMap, error) {
	var toplevel map[string]any
	err := json.Unmarshal(bytes, &toplevel)
	if err != nil {
		return nil, fmt.Errorf("unable to Unmarshal bytes, err=%w", err)
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
	return MakeParamsMapFromBytes([]byte(s))
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
		var v float64
		v, err := ParseFloat(value, origname)
		if err != nil {
			return err
		}
		paramVal = paramValFloat{def: d, value: v}
	default:
		e := fmt.Errorf("ParamValues.Set: unknown TypedParamDef for name=%s type=%T", origname, def.TypedParamDef)
		LogIfError(e)
		return e
	}
	vals.values[origname] = paramVal
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
	vals.mutex.RLock()
	_, exists := vals.values[name]
	vals.mutex.RUnlock()
	return exists
}

func (vals *ParamValues) ParamValueAsString(name string) (string, error) {
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
