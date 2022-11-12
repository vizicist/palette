package engine

// Scale says whether a pitch is in a scale
type Scale struct {
	HasNote [128]bool
}

// Scales maps a name to a Scale
var Scales map[string]*Scale

// GlobalScale xxx
func GlobalScale(name string) *Scale {
	s, ok := Scales[name]
	if !ok {
		Warn("No such scale", "name", name)
		s = Scales["newage"]
	}
	return s
}

// InitScales xxx
func InitScales() {

	Scales = make(map[string]*Scale)
	Scales["raga1"] = MakeScale(0, 1, 4, 5, 7, 8, 11)
	Scales["raga2"] = MakeScale(0, 2, 4, 6, 7, 9, 11)
	Scales["raga3"] = MakeScale(0, 2, 3, 5, 9, 10)
	Scales["raga4"] = MakeScale(0, 1, 4, 6, 7, 8, 11)

	Scales["arabian"] = MakeScale(0, 1, 4, 5, 7, 8, 10)
	Scales["newage"] = MakeScale(0, 3, 5, 7, 10)
	Scales["ionian"] = MakeScale(0, 2, 4, 5, 7, 9, 11)
	Scales["dorian"] = MakeScale(0, 2, 3, 5, 7, 9, 10)
	Scales["phrygian"] = MakeScale(0, 1, 3, 5, 7, 8, 10)
	Scales["lydian"] = MakeScale(0, 2, 4, 6, 7, 9, 11)
	Scales["mixolydian"] = MakeScale(0, 2, 4, 5, 7, 9, 10)
	Scales["aeolian"] = MakeScale(0, 2, 3, 5, 7, 8, 10)
	Scales["locrian"] = MakeScale(0, 1, 3, 5, 6, 8, 10)
	Scales["octaves"] = MakeScale(0)
	Scales["harminor"] = MakeScale(0, 2, 3, 5, 7, 8, 11)
	Scales["melminor"] = MakeScale(0, 2, 3, 5, 7, 9, 11)
	Scales["chromatic"] = MakeScale(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11)
	Scales["fifths"] = MakeScale(0, 7)
}

func MakeScale(pitches ...int) *Scale {
	s := &Scale{}
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
