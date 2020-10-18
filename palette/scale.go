package palette

import "log"

// Scale says whether a pitch is in a scale
type Scale struct {
	hasNote [128]bool
}

// Scales maps a name to a Scale
var Scales map[string]*Scale

// GetScale xxx
func GetScale(name string) *Scale {
	s, ok := Scales[name]
	if !ok {
		log.Printf("No scale named %s, assuming newage\n", name)
		s = Scales["newage"]
	}
	return s
}

// ClearExternalScale xxx
func ClearExternalScale() {
	Scales["external"] = makeScale()
}

// SetExternalScale xxx
func SetExternalScale(pitch int, on bool) {
	s := Scales["external"]
	for p := pitch; p < 128; p += 12 {
		s.hasNote[p] = on
	}
}

// InitScales xxx
func InitScales() {

	Scales = make(map[string]*Scale)
	Scales["raga1"] = makeScale(0, 1, 4, 5, 7, 8, 11)
	Scales["raga2"] = makeScale(0, 2, 4, 6, 7, 9, 11)
	Scales["raga3"] = makeScale(0, 2, 3, 5, 9, 10)
	Scales["raga4"] = makeScale(0, 1, 4, 6, 7, 8, 11)

	Scales["arabian"] = makeScale(0, 1, 4, 5, 7, 8, 10)
	Scales["newage"] = makeScale(0, 3, 5, 7, 10)
	Scales["ionian"] = makeScale(0, 2, 4, 5, 7, 9, 11)
	Scales["dorian"] = makeScale(0, 2, 3, 5, 7, 9, 10)
	Scales["phrygian"] = makeScale(0, 1, 3, 5, 7, 8, 10)
	Scales["lydian"] = makeScale(0, 2, 4, 6, 7, 9, 11)
	Scales["mixolydian"] = makeScale(0, 2, 4, 5, 7, 9, 10)
	Scales["aeolian"] = makeScale(0, 2, 3, 5, 7, 8, 10)
	Scales["locrian"] = makeScale(0, 1, 3, 5, 6, 8, 10)
	Scales["octaves"] = makeScale(0)
	Scales["harminor"] = makeScale(0, 2, 3, 5, 7, 8, 11)
	Scales["melminor"] = makeScale(0, 2, 3, 5, 7, 9, 11)
	Scales["chromatic"] = makeScale(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11)
	Scales["fifths"] = makeScale(0, 7)

	Scales["external"] = makeScale()
}

func makeScale(pitches ...int) *Scale {
	s := &Scale{}
	for _, pitch := range pitches {
		for p := pitch; p < 128; p += 12 {
			s.hasNote[p] = true
		}
	}
	return s
}

// ClosestTo xxx
func (s *Scale) ClosestTo(pitch uint8) uint8 {
	closestpitch := 0
	closestdelta := 9999
	for i := 0; i < 128; i++ {
		if !s.hasNote[i] {
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
