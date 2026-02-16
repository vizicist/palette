package kit

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	json "github.com/goccy/go-json"
)

// ParamDef is a single parameter definition.
type ParamDef struct {
	TypedParamDef any
	Category      string
	Init          string
	comment       string
}

type ParamDefFloat struct {
	min     float64
	max     float64
	randmin float64
	randmax float64
	hasRand bool // true if randmin or randmax was specified
	Init    string
	// comment string
}

type ParamDefInt struct {
	min     int
	max     int
	randmin int
	randmax int
	hasRand bool // true if randmin or randmax was specified
}

type ParamDefBool struct {
	randmax float64 // probability of being true (0.0-1.0)
	hasRand bool    // true if randmax was specified
}

type ParamDefString struct {
	values  []string
	randmax string // if set, always use this value for rand
	hasRand bool   // true if randmax was specified
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

	LogInfo("params GetWithPrefix", "prefix", prefix)
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
		valstring, e := vals.ParamValueAsString(fullName)
		if e != nil {
			LogIfError(e)
			continue
		}
		s += fmt.Sprintf("%s        \"%s\":\"%s\"", sep, fullName, valstring)
		sep = ",\n"
	}
	s += "\n    }\n}\n"
	data := []byte(s)
	return os.WriteFile(path, data, 0644)
}

func IsPatchCategory(category string) bool {
	return (category == "visual" ||
		category == "sound" ||
		category == "effect" ||
		category == "misc")
}

func LoadParamValuesOfCategory(category string, filename string) (*ParamValues, error) {
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

// ParamEnums contains the lists of enumerated values for string parameters
var ParamEnums map[string][]string

func ParseFloat(s string, name string) (float64, error) {
	f, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0.0, fmt.Errorf("ParseFloat of parameter '%s' (%s) fails", name, s)
	}
	return f, nil
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
	var toplevel map[string]any
	err = json.Unmarshal(bytes, &toplevel)
	if err != nil {
		return fmt.Errorf("loadParamEnums: unable to Unmarshal path=%s", path)
	}

	for enumName, enumList := range toplevel {
		var enums []string
		for _, e := range enumList.([]any) {
			enums = append(enums, e.(string))
		}
		ParamEnums[enumName] = enums
	}

	// Special case: populate "synth" enum from Synths.json
	loadSynthEnums()

	return nil
}

// loadSynthEnums reads Synths.json and populates the "synth" enum with all synth names
func loadSynthEnums() {
	path := ConfigFilePath("Synths.json")
	bytes, err := os.ReadFile(path)
	if err != nil {
		LogWarn("loadSynthEnums: unable to read Synths.json", "err", err)
		return
	}

	var data struct {
		Synths []struct {
			Name string `json:"name"`
		} `json:"synths"`
	}
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		LogWarn("loadSynthEnums: unable to parse Synths.json", "err", err)
		return
	}

	var synthNames []string
	for _, s := range data.Synths {
		synthNames = append(synthNames, s.Name)
	}
	ParamEnums["synth"] = synthNames
}

// LoadParamDefs initializes the list of parameters
func LoadParamDefs() error {

	ParamDefs = make(map[string]ParamDef)

	path := ConfigFilePath("paramdefs.json")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("unable to read %s, err=%w", path, err)
	}
	var params map[string]any
	err = json.Unmarshal(bytes, &params)
	if err != nil {
		return fmt.Errorf("unable to Unmarshal %s, err=%w", path, err)
	}
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
			fmin, err := ParseFloat(min, "min")
			if err != nil {
				return err
			}
			fmax, err := ParseFloat(max, "max")
			if err != nil {
				return err
			}
			// Check if randmin/randmax are specified
			frandmin := fmin
			frandmax := fmax
			hasRand := false
			if rm, ok := jmap["randmin"].(string); ok {
				if v, err := ParseFloat(rm, "randmin"); err == nil {
					frandmin = v
					hasRand = true
				}
			}
			if rm, ok := jmap["randmax"].(string); ok {
				if v, err := ParseFloat(rm, "randmax"); err == nil {
					frandmax = v
					hasRand = true
				}
			}
			pd.TypedParamDef = ParamDefFloat{
				min:     fmin,
				max:     fmax,
				randmin: frandmin,
				randmax: frandmax,
				hasRand: hasRand,
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
			// Check if randmin/randmax are specified
			irandmin := imin
			irandmax := imax
			hasRand := false
			if rm, ok := jmap["randmin"].(string); ok {
				if v, err := ParseInt(rm, "randmin"); err == nil {
					irandmin = v
					hasRand = true
				}
			}
			if rm, ok := jmap["randmax"].(string); ok {
				if v, err := ParseInt(rm, "randmax"); err == nil {
					irandmax = v
					hasRand = true
				}
			}
			pd.TypedParamDef = ParamDefInt{
				min:     imin,
				max:     imax,
				randmin: irandmin,
				randmax: irandmax,
				hasRand: hasRand,
			}

		case "bool":
			// randmax for bool is probability of being true (0.0-1.0)
			randprob := 0.5
			hasRand := false
			if rm, ok := jmap["randmax"].(string); ok {
				if v, err := ParseFloat(rm, "randmax"); err == nil {
					randprob = v
					hasRand = true
				}
			}
			pd.TypedParamDef = ParamDefBool{
				randmax: randprob,
				hasRand: hasRand,
			}

		case "string":
			// A bit of a hack - the "min" value of a
			// string parameter definition
			// is actually an "enum" type name
			enumName := min
			values := ParamEnums[enumName]
			// randmax for string is a specific value to always use
			randmax := ""
			hasRand := false
			if rm, ok := jmap["randmax"].(string); ok {
				randmax = rm
				hasRand = true
			}
			pd.TypedParamDef = ParamDefString{
				values:  values,
				randmax: randmax,
				hasRand: hasRand,
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

var overrideMap ParamsMap

// ParamDefsForCategory returns a newline-separated list of parameter names for a category
func ParamDefsForCategory(category string) (string, error) {
	var names []string
	for name, def := range ParamDefs {
		if def.Category == category || category == "*" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return strings.Join(names, "\n"), nil
}

// ParamInitValuesForCategory returns a JSON object mapping param names to their init values
func ParamInitValuesForCategory(category string) (string, error) {
	initMap := make(map[string]string)
	for name, def := range ParamDefs {
		if def.Category == category || category == "*" {
			initMap[name] = def.Init
		}
	}
	jsonBytes, err := json.Marshal(initMap)
	if err != nil {
		return "", fmt.Errorf("ParamInitValuesForCategory: %w", err)
	}
	return string(jsonBytes), nil
}

// RandomValueForParam generates a random value for a parameter based on its definition.
// Returns empty string if the parameter has no randmin/randmax specified.
func RandomValueForParam(def ParamDef) string {
	switch td := def.TypedParamDef.(type) {
	case ParamDefFloat:
		if !td.hasRand {
			return ""
		}
		val := td.randmin + rand.Float64()*(td.randmax-td.randmin)
		return fmt.Sprintf("%f", val)
	case ParamDefInt:
		if !td.hasRand {
			return ""
		}
		rangeSize := td.randmax - td.randmin + 1
		if rangeSize <= 0 {
			return fmt.Sprintf("%d", td.randmin)
		}
		val := td.randmin + rand.Intn(rangeSize)
		return fmt.Sprintf("%d", val)
	case ParamDefBool:
		if !td.hasRand {
			return ""
		}
		// randmax is probability of being true
		if rand.Float64() < td.randmax {
			return "true"
		}
		return "false"
	case ParamDefString:
		if !td.hasRand {
			return ""
		}
		// If randmax is set, always use that value
		if td.randmax != "" {
			return td.randmax
		}
		if len(td.values) == 0 {
			return ""
		}
		return td.values[rand.Intn(len(td.values))]
	default:
		return ""
	}
}

// ParamRandomValuesForCategory returns a JSON object mapping param names to random values.
// Only includes params that have randmin/randmax specified in their definition.
func ParamRandomValuesForCategory(category string) (string, error) {
	randMap := make(map[string]string)
	for name, def := range ParamDefs {
		if def.Category == category || category == "*" {
			randVal := RandomValueForParam(def)
			if randVal != "" {
				randMap[name] = randVal
			}
		}
	}
	jsonBytes, err := json.Marshal(randMap)
	if err != nil {
		return "", fmt.Errorf("ParamRandomValuesForCategory: %w", err)
	}
	return string(jsonBytes), nil
}

// ParamEnumsAsJSON returns the ParamEnums map as a JSON string
func ParamEnumsAsJSON() (string, error) {
	jsonBytes, err := json.Marshal(ParamEnums)
	if err != nil {
		return "", fmt.Errorf("ParamEnumsAsJSON: %w", err)
	}
	return string(jsonBytes), nil
}

// ParamDefsAsJSON returns paramdefs.json content directly
func ParamDefsAsJSON() (string, error) {
	path := ConfigFilePath("paramdefs.json")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("ParamDefsAsJSON: %w", err)
	}
	return string(bytes), nil
}

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
