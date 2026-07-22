package kit

// Parameter definitions: the ParamDef types, the paramdefs.json loader, the
// enum lists for string parameters, and per-category def/init/rand queries.
// Runtime parameter *values* (ParamValues) live in params.go.

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

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
		list, ok := enumList.([]any)
		if !ok {
			return fmt.Errorf("loadParamEnums: enum %q is %T, expected an array", enumName, enumList)
		}
		var enums []string
		for _, e := range list {
			s, ok := e.(string)
			if !ok {
				return fmt.Errorf("loadParamEnums: enum %q contains %T, expected string", enumName, e)
			}
			enums = append(enums, s)
		}
		ParamEnums[enumName] = enums
	}

	// Special case: populate "synth" enum from Synths.json
	loadSynthEnums()
	loadPitchSetEnums()
	loadShapeEnums()

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

func loadPitchSetEnums() {
	pitchSetNames, err := PitchSetNamesFromConfig()
	if err != nil {
		LogWarn("loadPitchSetEnums: unable to load PitchSets.json", "err", err)
		return
	}
	ParamEnums["pitchset"] = pitchSetNames
}

func loadShapeEnums() {
	shapeNames, err := shapeNamesFromDir(ParamEnums["shape"], filepath.Join(PaletteDataPath(), "shapes"))
	if err != nil {
		LogWarn("loadShapeEnums: unable to load shapes", "err", err)
		return
	}
	ParamEnums["shape"] = shapeNames
}

func shapeNamesFromDir(base []string, shapesDir string) ([]string, error) {
	names := make([]string, 0, len(base))
	seen := make(map[string]bool, len(base))
	for _, name := range base {
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if seen[key] {
			continue
		}
		seen[key] = true
		names = append(names, name)
	}

	entries, err := os.ReadDir(shapesDir)
	if err != nil {
		return names, err
	}

	var svgNames []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filename := entry.Name()
		if !strings.EqualFold(filepath.Ext(filename), ".svg") {
			continue
		}
		name := strings.TrimSuffix(filename, filepath.Ext(filename))
		if name == "" {
			continue
		}
		svgNames = append(svgNames, name)
	}
	sort.Strings(svgNames)

	for _, name := range svgNames {
		key := strings.ToLower(name)
		if seen[key] {
			continue
		}
		seen[key] = true
		names = append(names, name)
	}
	return names, nil
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
		jmap, err := jsonMap(dat, "param "+name)
		if err != nil {
			return fmt.Errorf("LoadParamDefs: %w", err)
		}
		min, err := jsonString(jmap, "min")
		if err != nil {
			return fmt.Errorf("LoadParamDefs: param %s: %w", name, err)
		}
		max, err := jsonString(jmap, "max")
		if err != nil {
			return fmt.Errorf("LoadParamDefs: param %s: %w", name, err)
		}
		valuetype, err := jsonString(jmap, "valuetype")
		if err != nil {
			return fmt.Errorf("LoadParamDefs: param %s: %w", name, err)
		}
		initval, err := jsonString(jmap, "init")
		if err != nil {
			return fmt.Errorf("LoadParamDefs: param %s: %w", name, err)
		}

		pd := ParamDef{
			Category: category,
			Init:     initval,
			comment:  jsonStringOr(jmap, "comment", ""),
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
	// Candidates are drawn from the plain rand distribution, then one is
	// chosen using the learned Like/Avoid feedback (see randfeedback.go).
	randMap := PickRandomParamsForCategory(category)
	jsonBytes, err := json.Marshal(randMap)
	if err != nil {
		return "", fmt.Errorf("ParamRandomValuesForCategory: %w", err)
	}
	return string(jsonBytes), nil
}

// ParamEnumsAsJSON returns the ParamEnums map as a JSON string
func ParamEnumsAsJSON() (string, error) {
	loadShapeEnums()
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
		if PathExists(overridepath) {
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
