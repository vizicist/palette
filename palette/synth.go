package palette

// SynthDef is the port and channel for a given synth
type SynthDef struct {
	port    string
	channel int
}

type synths struct {
	Synths []synth `json:"synths"`
}

type synth struct {
	Name    string `json:"name"`
	Port    string `json:"port"`
	Channel int    `json:"channel"`
}
