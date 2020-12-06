package engine

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
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
	min     float32
	max     float32
	Init    string
	comment string
}

type paramDefInt struct {
	min int
	max int
}

type paramDefBool struct {
}

type paramDefString struct {
	values []string
	// callback func(router *Router, reactor *Reactor, name, value string) error
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
	for nm, d := range ParamDefs {
		// log.Printf("setDefault nm=%s val=%v\n", nm, d.Init)
		vals.realSetParamValueWithString(nm, d.Init, nil, false /*no lock*/)
	}
	vals.mutex.Unlock()
}

// ParamCallback is the callback when setting parameter values
type ParamCallback func(name string, value string) error

// SetParamValueWithString xxx
func (vals *ParamValues) SetParamValueWithString(name, value string, callback ParamCallback) error {
	return vals.realSetParamValueWithString(name, value, callback, true)
}

func (vals *ParamValues) paramDefOf(origname string) (ParamDef, error) {

	realParamName := origname
	if strings.HasPrefix(origname, "effect.") {

		// We expect effect param names to be of the form
		// effect.{#}-{base} or effect.{#}-{base}:{param}

		restof := origname[len("effect."):]
		if len(restof) > 1 && restof[1] != '-' {
			restof = "1-" + restof
		}
		var base string
		var effnum int
		n := strings.Index(restof, ":")
		withoutcolon := restof
		if n > 0 {
			withoutcolon = restof[0:n]
		}
		// If it's a name like effect.mirror:x
		n, err := fmt.Sscanf(withoutcolon, "%d-%s", &effnum, &base)
		toreplace := withoutcolon
		if err != nil {
			return ParamDef{}, fmt.Errorf("ParamValues.SetParamValueWithString err=%s", err)
		}
		realParamName = strings.Replace(origname, toreplace, base, 1)
	}
	return ParamDefs[realParamName], nil
}

// realSetParamValueWithString xxx
func (vals *ParamValues) realSetParamValueWithString(origname, value string, callback ParamCallback, lockit bool) (err error) {

	// log.Printf("realSetParamValueWithString: %s %s\n", name, value)
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
		log.Printf("SetParamValueWithString: unknown type of ParamDef for name=%s", origname)
		return fmt.Errorf("SetParamValueWithString: unknown type of ParamDef for name=%s", origname)
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
	}
	vals.values[origname] = paramVal
	if lockit {
		vals.mutex.Unlock()
	}
	return nil
}

// ParamEnums contains the lists of enumerated values for string parameters
var ParamEnums map[string][]string

// ResolumeJSON is an unmarshalled version of the resolume.json file
var ResolumeJSON map[string]interface{}

// LoadResolumeJSON returns an unmarshalled version of the resolume.json file
func LoadResolumeJSON() error {
	path := ConfigFilePath("resolume.json")
	bytes, err := ioutil.ReadFile(path)
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
	bytes, err := ioutil.ReadFile(path)
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
	bytes, err := ioutil.ReadFile(path)
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

		ParamDefs[name] = pd
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
		log.Printf("**** No existing int value for param name=%s ??\n", name)
		return 0
	}
	return param.(paramValInt).value
}

// ParamFloatValue xxx
func (vals *ParamValues) ParamFloatValue(name string) float32 {
	param := vals.paramValue(name)
	if param == nil {
		log.Printf("**** No existing float value for param name=%s ??\n", name)
		return 0.0
	}
	f := (param).(paramValFloat).value
	return f
}

// ParamBoolValue xxx
func (vals *ParamValues) ParamBoolValue(name string) bool {
	param := vals.paramValue(name)
	if param == nil {
		log.Printf("**** No existing paramvalue for %s ??\n", name)
		return false
	}
	return (param).(paramValBool).value
}
