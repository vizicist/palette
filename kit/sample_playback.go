package kit

import (
	"fmt"
	"math"

	ss "github.com/vizicist/palette/pkg/samplesplitter"
)

const (
	samplePlaybackSampleCount       = 48
	minSamplePlaybackVelocity       = 76
	defaultSamplePlaybackQuantBeats = 0
	defaultSamplePlaybackVolume     = 1.0
)

type SamplePlaybackDomain struct {
	patch *Patch
}

type ActiveSamplePlayback struct {
	Patch          string
	SigilChannel   int
	SampleSelector int
	Velocity       int
	PitchBend      int
	VoiceKey       string
}

type SamplePlaybackStart struct {
	Patch          string
	SigilChannel   int
	SampleSelector int
	Velocity       int
	PitchBend      int
	VoiceKey       string
}

type SamplePlaybackStop struct {
	Patch          string
	SigilChannel   int
	SampleSelector int
	VoiceKey       string
}

type SamplePlaybackPitch struct {
	Patch        string
	SigilChannel int
	Value        int
}

func NewSamplePlaybackDomain(patch *Patch) SamplePlaybackDomain {
	return SamplePlaybackDomain{patch: patch}
}

func (d SamplePlaybackDomain) Enabled() bool {
	return IsBSSInitialPage() && d.route() == "samplesplitter"
}

func (d SamplePlaybackDomain) route() string {
	if theStepper == nil {
		return "bidule"
	}
	return string(theStepper.config.RouteForPatch(d.patchName()))
}

func (d SamplePlaybackDomain) RouteIncludesBidule() bool {
	route := d.route()
	return route == "bidule" || route == "both"
}

func (d SamplePlaybackDomain) RouteIncludesSamples() bool {
	route := d.route()
	return route == "samplesplitter" || route == "both"
}

func (d SamplePlaybackDomain) PlaybackFromCursor(ce CursorEvent) *ActiveSamplePlayback {
	x := boundValueZeroToOne(ce.Pos.X)
	sampleSelector := int(math.Min(samplePlaybackSampleCount-1, math.Floor(x*samplePlaybackSampleCount)))
	velocity := samplePlaybackVelocityFromPressure(d.patch, ce.Pos.Z)
	return &ActiveSamplePlayback{
		Patch:          d.patchName(),
		SigilChannel:   SamplePlaybackChannelForPatch(d.patchName()),
		SampleSelector: sampleSelector,
		Velocity:       int(velocity),
		PitchBend:      d.pitchBendValue(ce),
		VoiceKey:       samplePlaybackVoiceKey(ce),
	}
}

func (d SamplePlaybackDomain) HandleCursor(ce CursorEvent, ac *ActiveCursor) {
	if ac == nil {
		LogWarn("SamplePlaybackDomain.HandleCursor: no active cursor", "gid", ce.GID)
		return
	}

	switch ce.Ddu {
	case "down":
		playback := d.PlaybackFromCursor(ce)
		if playback == nil {
			return
		}
		atClick := d.nextQuant(CurrentClick())
		d.scheduleStart(ac, ce, playback, atClick)
		ac.NoteOnClick = atClick

	case "drag":
		oldPlayback := ac.ActiveSamplePlayback
		if oldPlayback == nil {
			playback := d.PlaybackFromCursor(ce)
			if playback == nil {
				return
			}
			atClick := d.nextQuant(CurrentClick())
			d.scheduleStart(ac, ce, playback, atClick)
			ac.NoteOnClick = atClick
			return
		}
		newPlayback := d.PlaybackFromCursor(ce)
		if newPlayback == nil {
			return
		}
		if newPlayback.SampleSelector != oldPlayback.SampleSelector {
			now := CurrentClick()
			onClick := d.nextQuant(now)
			d.cancelPendingStarts(ce.Tag, oldPlayback.SigilChannel)
			d.scheduleStart(ac, ce, newPlayback, onClick)
			ac.NoteOnClick = onClick
		} else {
			d.schedulePitch(ce.Tag, oldPlayback, d.pitchBendValue(ce), CurrentClick())
		}

	case "up":
		if ac.ActiveSamplePlayback != nil {
			LogOfType("sampleplayback", "SamplePlayback cursor up", "patch", ac.ActiveSamplePlayback.Patch, "sigilChannel", ac.ActiveSamplePlayback.SigilChannel, "sampleSelector", ac.ActiveSamplePlayback.SampleSelector, "click", CurrentClick())
			d.cancelPendingStarts(ce.Tag, ac.ActiveSamplePlayback.SigilChannel)
			d.scheduleStop(ce.Tag, ac.ActiveSamplePlayback, CurrentClick())
			ac.ActiveSamplePlayback = nil
		} else {
			sigilChannel := SamplePlaybackChannelForPatch(d.patchName())
			LogOfType("sampleplayback", "SamplePlayback cursor up without active playback", "patch", d.patchName(), "sigilChannel", sigilChannel, "click", CurrentClick())
			d.cancelPendingStarts(ce.Tag, sigilChannel)
			ScheduleAt(CurrentClick(), ce.Tag, &SamplePlaybackStop{Patch: d.patchName(), SigilChannel: sigilChannel, SampleSelector: -1})
			ScheduleAt(CurrentClick()+1, ce.Tag, &SamplePlaybackPitch{Patch: d.patchName(), SigilChannel: sigilChannel, Value: MidiPitchBendCenter})
		}
	}
}

func (d SamplePlaybackDomain) scheduleStart(ac *ActiveCursor, ce CursorEvent, playback *ActiveSamplePlayback, atClick Clicks) {
	if playback == nil {
		return
	}
	if ac.ActiveSamplePlayback != nil {
		d.scheduleStop(ce.Tag, ac.ActiveSamplePlayback, atClick)
	}
	ScheduleAt(atClick, ce.Tag, playback.StartEvent())
	ac.ActiveSamplePlayback = playback
}

func (d SamplePlaybackDomain) scheduleStop(tag string, playback *ActiveSamplePlayback, atClick Clicks) {
	if playback == nil {
		return
	}
	ScheduleAt(atClick, tag, playback.StopEvent())
	ScheduleAt(atClick+1, tag, playback.ResetPitchEvent())
}

func (d SamplePlaybackDomain) schedulePitch(tag string, playback *ActiveSamplePlayback, value int, atClick Clicks) {
	if playback == nil {
		return
	}
	ScheduleAt(atClick, tag, &SamplePlaybackPitch{Patch: playback.Patch, SigilChannel: playback.SigilChannel, Value: value})
}

func (d SamplePlaybackDomain) cancelPendingStarts(tag string, sigilChannel int) {
	theScheduler.DeleteSamplePlaybackStarts(tag, sigilChannel)
}

func (d SamplePlaybackDomain) nextQuant(t Clicks) Clicks {
	return nextQuant(t, d.quant())
}

func (d SamplePlaybackDomain) quant() Clicks {
	quantBeats, err := getSamplePlaybackFloatParam("global.sampleplaybackquant", defaultSamplePlaybackQuantBeats)
	if err != nil {
		LogIfError(err)
	}
	if quantBeats <= 0 {
		return 1
	}
	if quantBeats > 1 {
		quantBeats = 1
	}
	factor := TempoFactor
	if factor <= 0 {
		factor = 1
	}
	return Clicks(math.Max(1, float64(OneBeat)*quantBeats/factor))
}

func (d SamplePlaybackDomain) pitchBendValue(ce CursorEvent) int {
	return samplePlaybackPitchBendFromCursor(ce)
}

func samplePlaybackPitchBendFromCursor(ce CursorEvent) int {
	p := boundValueZeroToOne(ce.Pos.Y)
	return int(math.Round(p * 16383.0))
}

func (d SamplePlaybackDomain) patchName() string {
	if d.patch == nil {
		return ""
	}
	return d.patch.Name()
}

func (p *ActiveSamplePlayback) StartEvent() *SamplePlaybackStart {
	if p == nil {
		return nil
	}
	return &SamplePlaybackStart{
		Patch:          p.Patch,
		SigilChannel:   p.SigilChannel,
		SampleSelector: p.SampleSelector,
		Velocity:       p.Velocity,
		PitchBend:      p.PitchBend,
		VoiceKey:       p.VoiceKey,
	}
}

func (p *ActiveSamplePlayback) StopEvent() *SamplePlaybackStop {
	if p == nil {
		return nil
	}
	return &SamplePlaybackStop{
		Patch:          p.Patch,
		SigilChannel:   p.SigilChannel,
		SampleSelector: p.SampleSelector,
		VoiceKey:       p.VoiceKey,
	}
}

func (p *ActiveSamplePlayback) ResetPitchEvent() *SamplePlaybackPitch {
	if p == nil {
		return nil
	}
	return &SamplePlaybackPitch{
		Patch:        p.Patch,
		SigilChannel: p.SigilChannel,
		Value:        MidiPitchBendCenter,
	}
}

func SamplePlaybackChannelForPatch(patch string) int {
	switch patch {
	case "A":
		return 0
	case "B":
		return 1
	case "C":
		return 2
	case "D":
		return 3
	default:
		return 0
	}
}

func (event *SamplePlaybackStart) Trigger() {
	if event == nil {
		return
	}
	if !withSamplePlaybackService(func(service *ss.Service) {
		LogOfType("sampleplayback", "SamplePlaybackStart.Trigger", "patch", event.Patch, "sigilChannel", event.SigilChannel, "sampleSelector", event.SampleSelector, "velocity", event.Velocity, "pitchbend", event.PitchBend, "voiceKey", event.VoiceKey)
		service.MIDIPitchBend(event.SigilChannel, event.PitchBend)
		if err := service.NoteOnVoice(event.SigilChannel, event.SampleSelector, event.Velocity, event.VoiceKey); err != nil {
			LogWarn("SamplePlaybackStart", "err", err, "patch", event.Patch, "sigilChannel", event.SigilChannel, "sampleSelector", event.SampleSelector)
		}
	}) {
		LogWarn("SamplePlaybackStart: sample playback service is not running", "patch", event.Patch)
	}
}

func (event *SamplePlaybackStop) Trigger() {
	if event == nil {
		return
	}
	if !withSamplePlaybackService(func(service *ss.Service) {
		if event.VoiceKey != "" {
			LogOfType("sampleplayback", "SamplePlaybackStop.Trigger StopVoice", "patch", event.Patch, "sigilChannel", event.SigilChannel, "sampleSelector", event.SampleSelector, "voiceKey", event.VoiceKey)
			service.StopVoice(event.VoiceKey)
			return
		}
		if event.SampleSelector < 0 {
			LogOfType("sampleplayback", "SamplePlaybackStop.Trigger StopChannel", "patch", event.Patch, "sigilChannel", event.SigilChannel)
			service.StopChannel(event.SigilChannel)
			return
		}
		LogOfType("sampleplayback", "SamplePlaybackStop.Trigger", "patch", event.Patch, "sigilChannel", event.SigilChannel, "sampleSelector", event.SampleSelector)
		service.NoteOff(event.SigilChannel, event.SampleSelector)
	}) {
		LogWarn("SamplePlaybackStop: sample playback service is not running", "patch", event.Patch)
	}
}

func (event *SamplePlaybackPitch) Trigger() {
	if event == nil {
		return
	}
	if !withSamplePlaybackService(func(service *ss.Service) {
		LogOfType("sampleplayback", "SamplePlaybackPitch.Trigger", "patch", event.Patch, "sigilChannel", event.SigilChannel, "value", event.Value)
		service.MIDIPitchBend(event.SigilChannel, event.Value)
	}) {
		LogWarn("SamplePlaybackPitch: sample playback service is not running", "patch", event.Patch)
	}
}

func stopSamplePlaybackChannelForPatch(patch string, reason string) bool {
	sigilChannel := SamplePlaybackChannelForPatch(patch)
	stopped := withSamplePlaybackService(func(service *ss.Service) {
		LogOfType("sampleplayback", "stopSamplePlaybackChannelForPatch", "reason", reason, "patch", patch, "sigilChannel", sigilChannel)
		service.StopChannel(sigilChannel)
		service.MIDIPitchBend(sigilChannel, MidiPitchBendCenter)
	})
	return stopped
}

func samplePlaybackVoiceKey(ce CursorEvent) string {
	return fmt.Sprintf("cursor-%s-%d", ce.Tag, ce.GID)
}

func samplePlaybackVelocityFromPressure(patch *Patch, pressure float64) uint8 {
	globalPressure := globalPressureShape(pressure, "sound")
	scaledPressure := globalPressure.Scaled
	velocity := int(pressureToVelocity(scaledPressure, minSamplePlaybackVelocity, 127))
	velocity = int(math.Round(float64(velocity) * samplePlaybackVolume()))
	if velocity > 127 {
		velocity = 127
	}
	if velocity < 0 {
		velocity = 0
	}
	return uint8(velocity)
}

func samplePlaybackVolume() float64 {
	if GlobalParams == nil {
		return defaultSamplePlaybackVolume
	}
	volume, err := getSamplePlaybackFloatParam("global.sampleplaybackvolume", defaultSamplePlaybackVolume)
	if err != nil {
		LogIfError(err)
	}
	if volume < 0 {
		return 0
	}
	return volume
}

func getSamplePlaybackFloatParam(name string, dflt float64) (float64, error) {
	if GlobalParams == nil {
		return dflt, nil
	}
	value, err := GetParamFloat(name)
	if err == nil {
		return value, nil
	}
	return dflt, err
}

func nextQuant(t Clicks, q Clicks) Clicks {
	// The algorithm below is the same as KeyKit's nextquant.
	if q <= 1 {
		return t
	}
	tq := t
	rem := tq % q
	if (rem * 2) > q {
		tq += (q - rem)
	} else {
		tq -= rem
	}
	if tq < t {
		tq += q
	}
	return tq
}
