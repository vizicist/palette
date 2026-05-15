package kit

import (
	"fmt"
	"math"

	ss "github.com/vizicist/palette/pkg/samplesplitter"
)

const (
	transmissionSampleCount       = 48
	minTransmissionVelocity       = 76
	defaultTransmissionQuantBeats = 0.5
)

type TransmissionDomain struct {
	patch *Patch
}

type TransmissionPlayback struct {
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

func NewTransmissionDomain(patch *Patch) TransmissionDomain {
	return TransmissionDomain{patch: patch}
}

func (d TransmissionDomain) Enabled() bool {
	return IsBSS2InitialPage() && d.route() == "samplesplitter"
}

func (d TransmissionDomain) route() string {
	if theStepper == nil {
		return "bidule"
	}
	return string(theStepper.config.RouteForPatch(d.patchName()))
}

func (d TransmissionDomain) RouteIncludesBidule() bool {
	route := d.route()
	return route == "bidule" || route == "both"
}

func (d TransmissionDomain) RouteIncludesSamples() bool {
	route := d.route()
	return route == "samplesplitter" || route == "both"
}

func (d TransmissionDomain) PlaybackFromCursor(ce CursorEvent) *TransmissionPlayback {
	x := boundValueZeroToOne(ce.Pos.X)
	sampleSelector := int(math.Min(transmissionSampleCount-1, math.Floor(x*transmissionSampleCount)))
	velocity := transmissionVelocityFromPressure(d.patch, ce.Pos.Z)
	return &TransmissionPlayback{
		Patch:          d.patchName(),
		SigilChannel:   SamplePlaybackChannelForPatch(d.patchName()),
		SampleSelector: sampleSelector,
		Velocity:       int(velocity),
		PitchBend:      d.pitchBendValue(ce),
		VoiceKey:       transmissionVoiceKey(ce),
	}
}

func (d TransmissionDomain) HandleCursor(ce CursorEvent, ac *ActiveCursor) {
	if ac == nil {
		LogWarn("TransmissionDomain.HandleCursor: no active cursor", "gid", ce.GID)
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
		oldPlayback := ac.TransmissionPlayback
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
		if ac.TransmissionPlayback != nil {
			LogInfo("Transmission cursor up", "patch", ac.TransmissionPlayback.Patch, "sigilChannel", ac.TransmissionPlayback.SigilChannel, "sampleSelector", ac.TransmissionPlayback.SampleSelector, "click", CurrentClick())
			d.cancelPendingStarts(ce.Tag, ac.TransmissionPlayback.SigilChannel)
			d.scheduleStop(ce.Tag, ac.TransmissionPlayback, CurrentClick())
			ac.TransmissionPlayback = nil
		} else {
			sigilChannel := SamplePlaybackChannelForPatch(d.patchName())
			LogInfo("Transmission cursor up without active playback", "patch", d.patchName(), "sigilChannel", sigilChannel, "click", CurrentClick())
			d.cancelPendingStarts(ce.Tag, sigilChannel)
			ScheduleAt(CurrentClick(), ce.Tag, &SamplePlaybackStop{Patch: d.patchName(), SigilChannel: sigilChannel, SampleSelector: -1})
			ScheduleAt(CurrentClick()+1, ce.Tag, &SamplePlaybackPitch{Patch: d.patchName(), SigilChannel: sigilChannel, Value: MidiPitchBendCenter})
		}
	}
}

func (d TransmissionDomain) scheduleStart(ac *ActiveCursor, ce CursorEvent, playback *TransmissionPlayback, atClick Clicks) {
	if playback == nil {
		return
	}
	if ac.TransmissionPlayback != nil {
		d.scheduleStop(ce.Tag, ac.TransmissionPlayback, atClick)
	}
	ScheduleAt(atClick, ce.Tag, playback.StartEvent())
	ac.TransmissionPlayback = playback
}

func (d TransmissionDomain) scheduleStop(tag string, playback *TransmissionPlayback, atClick Clicks) {
	if playback == nil {
		return
	}
	ScheduleAt(atClick, tag, playback.StopEvent())
	ScheduleAt(atClick+1, tag, playback.ResetPitchEvent())
}

func (d TransmissionDomain) schedulePitch(tag string, playback *TransmissionPlayback, value int, atClick Clicks) {
	if playback == nil {
		return
	}
	ScheduleAt(atClick, tag, &SamplePlaybackPitch{Patch: playback.Patch, SigilChannel: playback.SigilChannel, Value: value})
}

func (d TransmissionDomain) cancelPendingStarts(tag string, sigilChannel int) {
	theScheduler.DeleteSamplePlaybackStarts(tag, sigilChannel)
}

func (d TransmissionDomain) nextQuant(t Clicks) Clicks {
	return nextQuant(t, d.quant())
}

func (d TransmissionDomain) quant() Clicks {
	quantBeats, err := GetParamFloat("global.transmissionquant")
	if err != nil {
		LogIfError(err)
		quantBeats = defaultTransmissionQuantBeats
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

func (d TransmissionDomain) pitchBendValue(ce CursorEvent) int {
	p := boundValueZeroToOne(ce.Pos.Y)
	return int(math.Round(p * 16383.0))
}

func (d TransmissionDomain) patchName() string {
	if d.patch == nil {
		return ""
	}
	return d.patch.Name()
}

func (p *TransmissionPlayback) StartEvent() *SamplePlaybackStart {
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

func (p *TransmissionPlayback) StopEvent() *SamplePlaybackStop {
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

func (p *TransmissionPlayback) ResetPitchEvent() *SamplePlaybackPitch {
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
	if !withInEngineSamplesplitter(func(service *ss.Service) {
		LogInfo("SamplePlaybackStart.Trigger", "patch", event.Patch, "sigilChannel", event.SigilChannel, "sampleSelector", event.SampleSelector, "velocity", event.Velocity, "pitchbend", event.PitchBend, "voiceKey", event.VoiceKey)
		service.MIDIPitchBend(event.SigilChannel, event.PitchBend)
		if err := service.NoteOnVoice(event.SigilChannel, event.SampleSelector, event.Velocity, event.VoiceKey); err != nil {
			LogWarn("SamplePlaybackStart", "err", err, "patch", event.Patch, "sigilChannel", event.SigilChannel, "sampleSelector", event.SampleSelector)
		}
	}) {
		LogWarn("SamplePlaybackStart: in-engine service is not running", "patch", event.Patch)
	}
}

func (event *SamplePlaybackStop) Trigger() {
	if event == nil {
		return
	}
	if !withInEngineSamplesplitter(func(service *ss.Service) {
		if event.VoiceKey != "" {
			LogInfo("SamplePlaybackStop.Trigger StopVoice", "patch", event.Patch, "sigilChannel", event.SigilChannel, "sampleSelector", event.SampleSelector, "voiceKey", event.VoiceKey)
			service.StopVoice(event.VoiceKey)
			return
		}
		if event.SampleSelector < 0 {
			LogInfo("SamplePlaybackStop.Trigger StopChannel", "patch", event.Patch, "sigilChannel", event.SigilChannel)
			service.StopChannel(event.SigilChannel)
			return
		}
		LogInfo("SamplePlaybackStop.Trigger", "patch", event.Patch, "sigilChannel", event.SigilChannel, "sampleSelector", event.SampleSelector)
		service.NoteOff(event.SigilChannel, event.SampleSelector)
	}) {
		LogWarn("SamplePlaybackStop: in-engine service is not running", "patch", event.Patch)
	}
}

func (event *SamplePlaybackPitch) Trigger() {
	if event == nil {
		return
	}
	if !withInEngineSamplesplitter(func(service *ss.Service) {
		LogInfo("SamplePlaybackPitch.Trigger", "patch", event.Patch, "sigilChannel", event.SigilChannel, "value", event.Value)
		service.MIDIPitchBend(event.SigilChannel, event.Value)
	}) {
		LogWarn("SamplePlaybackPitch: in-engine service is not running", "patch", event.Patch)
	}
}

func transmissionVoiceKey(ce CursorEvent) string {
	return fmt.Sprintf("cursor-%s-%d", ce.Tag, ce.GID)
}

func transmissionVelocityFromPressure(patch *Patch, pressure float64) uint8 {
	scaledPressure := boundValueZeroToOne(pressure)
	if patch != nil {
		zmin := patch.GetFloat("sound._controllerzmin")
		zmax := patch.GetFloat("sound._controllerzmax")
		scaledPressure = BoundAndScaleFloat(pressure, zmin, zmax, 0.0, 1.0)
	}
	velocity := minTransmissionVelocity + int(math.Round(scaledPressure*float64(127-minTransmissionVelocity)))
	if velocity > 127 {
		velocity = 127
	}
	return uint8(velocity)
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
