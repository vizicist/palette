package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

// parameter definitions

// ParamDef is a single parameter definition.
type ParamDef struct {
	typedParamDef interface{}
	Category      string
	Init          string
	comment       string
}

type paramDefFloat struct {
	min  float32
	max  float32
	Init string
	// comment string
}

type paramDefInt struct {
	min int
	max int
}

type paramDefBool struct {
}

type paramDefString struct {
	values []string
}

// ParamDefs is the set of all parameter definitions
var ParamDefs map[string]ParamDef

// parameter values

// ParamValue is a single parameter value
// which could be any of the param*Value types
type ParamValue interface{}

type paramValString struct {
	def   paramDefString
	value string
}

type paramValInt struct {
	def   paramDefInt
	value int
}

type paramValFloat struct {
	def   paramDefFloat
	value float32
}

type paramValBool struct {
	def   paramDefBool
	value bool
}

// ParamValues is the set of all parameter values
type ParamValues struct {
	mutex  sync.RWMutex
	values map[string]ParamValue
}

// NewParamValues creates a new ParamValues
func NewParamValues() *ParamValues {
	return &ParamValues{
		values: make(map[string]ParamValue),
	}
}

// SetDefaultValues xxx
func (vals *ParamValues) SetDefaultValues() {
	vals.mutex.Lock()
	defer vals.mutex.Unlock()
	for nm, d := range ParamDefs {
		err := vals.internalSetParamValueWithString(nm, d.Init, nil, false)
		if err != nil {
			LogError(err)
		}
	}
}

// ParamCallback is the callback when setting parameter values
type ParamCallback func(name string, value string) error

func (vals *ParamValues) paramDefOf(name string) (ParamDef, error) {
	p, ok := ParamDefs[name]
	if !ok {
		return ParamDef{}, fmt.Errorf("paramDefOf: no parameter named %s", name)
	} else {
		return p, nil
	}
}

// SetParamValueWithString xxx
func (vals *ParamValues) SetParamValueWithString(name, value string, callback ParamCallback) error {
	return vals.internalSetParamValueWithString(name, value, callback, true)
}

func (vals *ParamValues) internalSetParamValueWithString(origname, value string, callback ParamCallback, lockit bool) (err error) {

	if origname == "pad" {
		return fmt.Errorf("ParamValues.SetParamValueWithString rejects setting of pad value")
	}

	def, err := vals.paramDefOf(origname)
	if err != nil {
		return err
	}

	var paramVal ParamValue
	switch d := def.typedParamDef.(type) {
	case paramDefInt:
		valint, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		paramVal = paramValInt{def: d, value: valint}
	case paramDefBool:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		paramVal = paramValBool{def: d, value: v}
	case paramDefString:
		paramVal = paramValString{def: d, value: value}
	case paramDefFloat:
		var v float32
		v, err := ParseFloat32(value, origname)
		if err != nil {
			return err
		}
		paramVal = paramValFloat{def: d, value: float32(v)}
	default:
		e := fmt.Errorf("SetParamValueWithString: unknown type of ParamDef for name=%s", origname)
		LogError(e)
		return e
	}

	// Perhaps the callback should be inside the Lock?
	if callback != nil {
		err := callback(origname, value)
		if err != nil {
			return err
		}
	}

	if lockit {
		vals.mutex.Lock()
		defer vals.mutex.Unlock()
	}
	vals.values[origname] = paramVal
	return nil
}

// ParamEnums contains the lists of enumerated values for string parameters
var ParamEnums map[string][]string

// ResolumeJSON is an unmarshalled version of the resolume.json file
var ResolumeJSON map[string]interface{}

// LoadResolumeJSON returns an unmarshalled version of the resolume.json file
func LoadResolumeJSON() error {
	path := ConfigFilePath("resolume.json")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("unable to read resolume.json, err=%s", err)
	}
	var f interface{}
	err = json.Unmarshal(bytes, &f)
	if err != nil {
		return fmt.Errorf("unable to Unmarshal %s", path)
	}
	ResolumeJSON = f.(map[string]interface{})
	return nil
}

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
	var f interface{}
	err = json.Unmarshal(bytes, &f)
	if err != nil {
		return fmt.Errorf("loadParamEnums: unable to Unmarshal path=%s", path)
	}
	toplevel := f.(map[string]interface{})

	for enumName, enumList := range toplevel {
		var enums []string
		for _, e := range enumList.([]interface{}) {
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
	var f interface{}
	err = json.Unmarshal(bytes, &f)
	if err != nil {
		return fmt.Errorf("unable to Unmarshal %s, err=%s", path, err)
	}
	params := f.(map[string]interface{})
	for name, dat := range params {
		w := strings.SplitN(name, ".", 2)
		if len(w) != 2 {
			return fmt.Errorf("LoadParamDefs: parameter has no category - %s", name)
		}
		category := w[0]
		jmap := dat.(map[string]interface{})
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
			pd.typedParamDef = paramDefFloat{
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
			pd.typedParamDef = paramDefInt{
				min: imin,
				max: imax,
			}

		case "bool":
			pd.typedParamDef = paramDefBool{}

		case "string":
			// A bit of a hack - the "min" value of a
			// string parameter definition
			// is actually an "enum" type name
			enumName := min
			values := ParamEnums[enumName]
			pd.typedParamDef = paramDefString{
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

func (vals *ParamValues) paramValueAsString(name string) (string, error) {
	val := vals.paramValue(name)
	if val == nil {
		return "", fmt.Errorf("paramValueAsString: no parameter named %s", name)
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

// ParamStringValue xxx
func (vals *ParamValues) ParamStringValue(name string, def string) string {
	val := vals.paramValue(name)
	if val == nil {
		return def
	}
	return val.(paramValString).value
}

// ParamIntValue xxx
func (vals *ParamValues) ParamIntValue(name string) int {
	param := vals.paramValue(name)
	if param == nil {
		Warn("**** No existing int value for param", "name", name)
		return 0
	}
	return param.(paramValInt).value
}

// ParamFloatValue xxx
func (vals *ParamValues) ParamFloatValue(name string) float32 {
	param := vals.paramValue(name)
	if param == nil {
		Warn("No existing float value for param", "name", name)
		pd, ok := ParamDefs[name]
		if ok {
			f, err := strconv.ParseFloat(pd.Init, 64)
			if err == nil {
				LogError(err)
				return float32(f)
			}
		}
		return 0.0
	}
	f := (param).(paramValFloat).value
	return f
}

// ParamBoolValue xxx
func (vals *ParamValues) ParamBoolValue(name string) bool {
	param := vals.paramValue(name)
	if param == nil {
		Warn("No existing paramvalue for", "name", name)
		return false
	}
	return (param).(paramValBool).value
}
