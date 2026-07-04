package kit

import "fmt"

// Helpers for safely reading values out of unmarshalled JSON maps
// (map[string]any). Raw type assertions on such maps panic on malformed
// config files; these return errors that name the offending key instead.

// jsonString returns m[key] as a string, or an error naming the key.
func jsonString(m map[string]any, key string) (string, error) {
	v, ok := m[key]
	if !ok {
		return "", fmt.Errorf("missing %q", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%q is %T, expected string", key, v)
	}
	return s, nil
}

// jsonStringOr returns m[key] as a string, or def if the key is absent
// or not a string.
func jsonStringOr(m map[string]any, key string, def string) string {
	if s, err := jsonString(m, key); err == nil {
		return s
	}
	return def
}

// jsonMap returns v as a map[string]any, or an error using what for context.
func jsonMap(v any, what string) (map[string]any, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s is %T, expected a JSON object", what, v)
	}
	return m, nil
}
