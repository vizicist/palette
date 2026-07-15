package kit

import "strings"

const (
	modeBSS  = "bss"
	modePro  = "pro"
	modePro2 = "pro2"
)

func normalizeMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case modeBSS:
		return modeBSS
	case modePro2:
		return modePro2
	default:
		return modePro
	}
}

func CurrentMode() string {
	if GlobalParams == nil {
		return modePro
	}
	mode, err := GetParam("global.mode")
	if err != nil {
		return modePro
	}
	return normalizeMode(mode)
}

func IsBSSMode() bool {
	return CurrentMode() == modeBSS
}

// IsPro2Mode reports whether the engine is running in the pro2 mode, which
// starts as a clone of pro but is free to diverge in future work.
func IsPro2Mode() bool {
	return CurrentMode() == modePro2
}

func IsBSSInitialPage() bool {
	return IsBSSMode()
}

func CoerceStepperRouteForMode(route StepperRoute) StepperRoute {
	if !validStepperRoute(string(route)) {
		route = StepperRouteSamplesplitter
	}
	if IsBSSMode() {
		return route
	}
	switch route {
	case StepperRouteSamplesplitter, StepperRouteBoth:
		return StepperRouteBidule
	default:
		return route
	}
}

func ApplyMode() {
	if !IsBSSMode() {
		SendAllNotesOffToSynths()
		if theStepper != nil {
			theStepper.SetPlaying(false)
			theStepper.SetAllRecording(false)
		}
	}
	ApplyModeStepperRoutes()
	SyncSamplesplitterProcessForMode()
}

func ApplyModeStepperRoutes() {
	changed := false
	for _, patchName := range patchNames {
		patch := GetPatch(patchName)
		if patch == nil {
			continue
		}
		route := StepperRoute(patch.Get("stepper.route"))
		coerced := CoerceStepperRouteForMode(route)
		if coerced == route {
			continue
		}
		if err := patch.SetParam("stepper.route", string(coerced)); err != nil {
			LogWarn("ApplyModeStepperRoutes", "patch", patchName, "err", err)
			continue
		}
		patch.noticeValueChange("visual.shape", patch.Get("visual.shape"))
		changed = true
	}
	if changed {
		NotifyStepperChanged()
	}
}
