package kit

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// StringMap takes a JSON string and returns a map of elements
func StringMap(params string) (map[string]string, error) {
	// The enclosing curly braces are optional
	if params == "" || params[0] == '"' {
		params = "{ " + params + " }"
	}
	dec := json.NewDecoder(strings.NewReader(params))
	t, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if t != json.Delim('{') {
		LogWarn("no curly", "params", params)
		return nil, errors.New("expected '{' delimiter")
	}
	values := make(map[string]string)
	for dec.More() {
		name, err := dec.Token()
		if err != nil {
			return nil, err
		}
		if !dec.More() {
			return nil, errors.New("incomplete JSON?")
		}
		value, err := dec.Token()
		if err != nil {
			return nil, err
		}
		// The name and value Tokens can be floats or strings or ...
		n := fmt.Sprintf("%v", name)
		v := fmt.Sprintf("%v", value)
		values[n] = v
	}
	return values, nil
}

func MapString(amap map[string]string) string {
	final := ""
	sep := ""
	for _, val := range amap {
		final = final + sep + "\"" + val + "\""
		sep = ","
	}
	return final
}

// ExtractAndRemoveValueOf removes a named value from a map and returns it.
// If the value doesn't exist, "" is returned.
func ExtractAndRemoveValueOf(valName string, argsmap map[string]string) string {
	val, ok := argsmap[valName]
	if !ok {
		val = ""
	}
	delete(argsmap, valName)
	return val
}

// ResultResponse returns a JSON 2.0 result response
func ResultResponse(resultObj any) string {
	bytes, err := json.Marshal(resultObj)
	if err != nil {
		LogWarn("ResultResponse: unable to marshal resultObj")
		return ""
	}
	result := string(bytes)
	if result == "" {
		result = "\"0\""
	}
	return `{ "result": ` + result + ` }`
}

func jsonEscape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\") // has to be first
	s = strings.ReplaceAll(s, "\b", "\\b")
	s = strings.ReplaceAll(s, "\f", "\\f")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// ErrorResponse return an error response
func ErrorResponse(err error) string {
	escaped := jsonEscape(err.Error())
	return `{ "error": "` + escaped + `" }`
}

