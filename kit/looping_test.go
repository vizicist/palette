package kit

import "testing"

func TestLoopingGlobalParamsStoredBySetAndApply(t *testing.T) {
	oldParamDefs := ParamDefs
	oldGlobalParams := GlobalParams
	defer func() {
		ParamDefs = oldParamDefs
		GlobalParams = oldGlobalParams
	}()

	ParamDefs = map[string]ParamDef{
		"global.looping_override": {
			Category:      "global",
			Init:          "false",
			TypedParamDef: ParamDefBool{},
		},
		"global.looping_fade": {
			Category: "global",
			Init:     "0.0",
			TypedParamDef: ParamDefFloat{
				min: 0,
				max: 1,
			},
		},
		"global.looping_beats": {
			Category: "global",
			Init:     "8",
			TypedParamDef: ParamDefInt{
				min: 1,
				max: 128,
			},
		},
	}
	GlobalParams = NewParamValues()

	tests := map[string]struct {
		value string
		want  string
	}{
		"global.looping_override": {value: "true", want: "true"},
		"global.looping_fade":     {value: "0.5", want: "0.500000"},
		"global.looping_beats":    {value: "16", want: "16"},
	}

	for name, tt := range tests {
		if err := SetAndApplyGlobalParam(name, tt.value); err != nil {
			t.Fatalf("SetAndApplyGlobalParam(%q) err = %v", name, err)
		}
		got, err := GlobalParams.Get(name)
		if err != nil {
			t.Fatalf("GlobalParams.Get(%q) err = %v", name, err)
		}
		if got != tt.want {
			t.Fatalf("GlobalParams.Get(%q) = %q, want %q", name, got, tt.want)
		}
	}
}
