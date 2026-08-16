package main

import "testing"

// The palette name is interpolated into the NATS configuration inside a quoted
// string and twice as a subject token (to_palette.<name>.> and
// from_palette.<name>.>). Anything that can close the string, add a line, or
// widen the subject has to be refused before it reaches the config file.
func TestPaletteUserNameRejectsInjection(t *testing.T) {
	bad := map[string]string{
		"closes the quoted string":     `a", permissions: {subscribe: ">"}, x: "`,
		"adds a configuration line":    "a\"}\n        {user: \"admin\", password: \"\"",
		"subject wildcard >":           ">",
		"subject wildcard *":           "*",
		"wildcard inside a name":       "spacepalette.>",
		"token separator":              "a.b",
		"space":                        "space palette",
		"carriage return":              "a\rb",
		"newline":                      "a\nb",
		"empty":                        "",
		"backslash":                    `a\b`,
		"single quote":                 "a'b",
		"too long (65 characters)":     "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"leading dollar (NATS system)": "$SYS",
	}
	for what, name := range bad {
		if paletteUserName.MatchString(name) {
			t.Errorf("%s: %q was accepted", what, name)
		}
	}
}

// The names actually in use have to keep working.
func TestPaletteUserNameAcceptsRealNames(t *testing.T) {
	good := []string{
		"spacepalette37",
		"spacepalette",
		"palette-2",
		"palette_2",
		"A1",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", // 64
	}
	for _, name := range good {
		if !paletteUserName.MatchString(name) {
			t.Errorf("%q was rejected", name)
		}
	}
}
