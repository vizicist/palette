package kit

// Scale says whether a pitch is in a scale
type Scale struct {
	HasNote [128]bool
}

// Scales maps a name to a Scale
var scales map[string]*Scale

func ClearExternalScale() {
	initScales()
	LogOfType("scale", "Clearing external scale")
	scales["external"] = MakeScale()
}

func SetExternalScale(pitch uint8, on bool) {
	initScales()
	pitch = pitch % 12 // start from lowest octave
	s := scales["external"]
	LogOfType("scale", "Adding to external scale", "pitch", pitch, "on", on)
	for p := pitch; p < 128; p += 12 {
		s.HasNote[p] = on
	}
}

// GetScale xxx
func GetScale(name string) *Scale {
	initScales()
	s, ok := scales[name]
	if !ok {
		LogWarn("No such scale", "name", name)
		s = scales["newage"]
	}
	return s
}

// initScales xxx
func initScales() {

	if scales != nil {
		return
	}
	scales = make(map[string]*Scale)
	scales["raga1"] = MakeScale(0, 1, 4, 5, 7, 8, 11)
	scales["raga2"] = MakeScale(0, 2, 4, 6, 7, 9, 11)
	scales["raga3"] = MakeScale(0, 2, 3, 5, 9, 10)
	scales["raga4"] = MakeScale(0, 1, 4, 6, 7, 8, 11)

	scales["arabian"] = MakeScale(0, 1, 4, 5, 7, 8, 10)
	scales["newage"] = MakeScale(0, 3, 5, 7, 10)
	scales["ionian"] = MakeScale(0, 2, 4, 5, 7, 9, 11)
	scales["dorian"] = MakeScale(0, 2, 3, 5, 7, 9, 10)
	scales["phrygian"] = MakeScale(0, 1, 3, 5, 7, 8, 10)
	scales["lydian"] = MakeScale(0, 2, 4, 6, 7, 9, 11)
	scales["mixolydian"] = MakeScale(0, 2, 4, 5, 7, 9, 10)
	scales["aeolian"] = MakeScale(0, 2, 3, 5, 7, 8, 10)
	scales["locrian"] = MakeScale(0, 1, 3, 5, 6, 8, 10)
	scales["octaves"] = MakeScale(0)
	scales["harminor"] = MakeScale(0, 2, 3, 5, 7, 8, 11)
	scales["melminor"] = MakeScale(0, 2, 3, 5, 7, 9, 11)
	scales["chromatic"] = MakeScale(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11)
	scales["fifths"] = MakeScale(0, 7)

	// The external scale is initialized to chromatic,
	// but will be changed by MIDI input when global.setexternalscale is on.
	scales["external"] = MakeScale(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11)
}

func MakeScale(pitches ...int) *Scale {
	s := &Scale{
		HasNote: [128]bool{},
	}
	for _, pitch := range pitches {
		for p := pitch; p < 128; p += 12 {
			s.HasNote[p] = true
		}
	}
	return s
}

// ClosestTo xxx
func (s *Scale) ClosestTo(pitch uint8) uint8 {
	closestpitch := 0
	closestdelta := 9999
	for i := 0; i < 128; i++ {
		if !s.HasNote[i] {
			continue
		}
		delta := int(pitch) - i
		if delta < 0 {
			delta = -delta
		}
		if delta < closestdelta {
			closestdelta = delta
			closestpitch = i
		}
	}
	return uint8(closestpitch)
}
