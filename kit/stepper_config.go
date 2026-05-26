package kit

import "fmt"

type StepperRoute string

const (
	StepperRouteOff            StepperRoute = "off"
	StepperRouteBidule         StepperRoute = "bidule"
	StepperRouteSamplesplitter StepperRoute = "samplesplitter"
	StepperRouteBoth           StepperRoute = "both"
)

type StepperConfig struct{}

func NewStepperConfig() StepperConfig {
	return StepperConfig{}
}

func (c StepperConfig) SequencingEnabled() bool {
	return !IsBSSInitialPage()
}

func (c StepperConfig) CoercePlaying(playing bool) bool {
	if playing && !c.SequencingEnabled() {
		return false
	}
	return playing
}

func (c StepperConfig) SetRoute(patch string, route string) error {
	if !validStepperRoute(route) {
		return fmt.Errorf("stepper.setroute: bad route=%s", route)
	}
	p := GetPatch(patch)
	if p == nil {
		return fmt.Errorf("no such patch: %s", patch)
	}
	if err := p.SetParam("stepper.route", route); err != nil {
		return err
	}
	if err := p.SaveQuadAndAlert(); err != nil {
		return err
	}
	p.noticeValueChange("visual.shape", p.Get("visual.shape"))
	return nil
}

func (c StepperConfig) RouteForPatch(patch string) StepperRoute {
	p := GetPatch(patch)
	if p == nil {
		return StepperRouteOff
	}
	route := p.Get("stepper.route")
	if !validStepperRoute(route) {
		return StepperRouteSamplesplitter
	}
	return StepperRoute(route)
}

func (c StepperConfig) RouteIncludesBidule(patch string) bool {
	route := c.RouteForPatch(patch)
	return route == StepperRouteBidule || route == StepperRouteBoth
}

func (c StepperConfig) RouteIncludesSamples(patch string) bool {
	route := c.RouteForPatch(patch)
	return route == StepperRouteSamplesplitter || route == StepperRouteBoth
}

func (c StepperConfig) BiduleSynthForPatch(patch string, event StepperEvent) *Synth {
	p := GetPatch(patch)
	if p != nil {
		return p.Synth()
	}
	if event.SynthName != "" {
		return GetSynth(event.SynthName)
	}
	return nil
}

func (c StepperConfig) SamplesplitterSynthForPatch(patch string) *Synth {
	p := GetPatch(patch)
	if p == nil {
		return nil
	}
	synthName := p.Get("stepper.samplesplitter_synth")
	if synthName == "" {
		synthName = c.DefaultSamplesplitterSynthForPatch(patch)
	}
	if synthName == "P_16_C_01" && patch != "A" {
		synthName = c.DefaultSamplesplitterSynthForPatch(patch)
	}
	return GetSynth(synthName)
}

func (c StepperConfig) DefaultSamplesplitterSynthForPatch(patch string) string {
	switch patch {
	case "A":
		return "P_16_C_01"
	case "B":
		return "P_16_C_02"
	case "C":
		return "P_16_C_03"
	case "D":
		return "P_16_C_04"
	default:
		return "P_16_C_01"
	}
}

func validStepperRoute(route string) bool {
	switch StepperRoute(route) {
	case StepperRouteOff, StepperRouteBidule, StepperRouteSamplesplitter, StepperRouteBoth:
		return true
	default:
		return false
	}
}
