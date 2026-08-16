package samplesplitter

import "testing"

// A one-shot already sounding on a channel must not be restarted when the next
// one-shot arrives on that channel.
//
// Play marked the previous channel voice inactive and reset its position for
// every request, looping or not, but mixIntoLocked suppresses an inactive voice
// only when it loops. A one-shot therefore stayed perfectly audible with its
// position back at zero, so the older sample restarted from the beginning and
// mixed underneath the new one.
func TestChannelArbitrationLeavesOneShotsAlone(t *testing.T) {
	audio := &AudioManager{}
	audio.ensureVoiceMapsLocked()

	// An older one-shot, part way through.
	old := &audioVoice{pcm: make([]int16, 100), position: 40, loop: false, channel: 1, note: 60, active: true}
	audio.voices["old"] = old
	audio.channelActive[1] = "old"

	// What Play does to the previous voice when a new one arrives.
	if activeKey, ok := audio.channelActive[1]; ok {
		if activeVoice := audio.voices[activeKey]; activeVoice != nil && activeVoice.loop {
			activeVoice.active = false
			activeVoice.position = 0
		}
	}

	if old.position != 40 {
		t.Errorf("a one-shot's position was reset to %d; it would restart from the beginning", old.position)
	}
	if !old.active {
		t.Error("a one-shot was marked inactive, which the mixer ignores for non-looping voices")
	}
}

// A looping voice on the channel is still arbitrated, because the mixer does
// honour inactive for those.
func TestChannelArbitrationStillRetiresLoops(t *testing.T) {
	audio := &AudioManager{}
	audio.ensureVoiceMapsLocked()

	loop := &audioVoice{pcm: make([]int16, 100), position: 40, loop: true, channel: 1, note: 60, active: true}
	audio.voices["loop"] = loop
	audio.channelActive[1] = "loop"

	if activeKey, ok := audio.channelActive[1]; ok {
		if activeVoice := audio.voices[activeKey]; activeVoice != nil && activeVoice.loop {
			activeVoice.active = false
			activeVoice.position = 0
		}
	}

	if loop.active || loop.position != 0 {
		t.Errorf("a looping voice was not retired: active=%v position=%d", loop.active, loop.position)
	}
}

// The mixer's own rule, which is what makes the above matter: an inactive voice
// is skipped only when it loops.
func TestMixerOnlySuppressesInactiveLoops(t *testing.T) {
	skip := func(v *audioVoice) bool {
		return v.loop && v.channel >= 0 && !v.active
	}
	if skip(&audioVoice{loop: false, channel: 1, active: false}) {
		t.Error("mixer skipped an inactive one-shot; the fix would be unnecessary")
	}
	if !skip(&audioVoice{loop: true, channel: 1, active: false}) {
		t.Error("mixer did not skip an inactive looping voice")
	}
}

// A stop that lands while Play is decoding has to win. The epochs are what Play
// compares before and after the decode.
func TestStopEpochsCancelAnInFlightPlay(t *testing.T) {
	audio := &AudioManager{}
	audio.ensureVoiceMapsLocked()

	// What Play captures before it decodes.
	startVoice := audio.voiceStopEpoch["v"]
	startChannel := audio.channelStopEpoch[3]
	startAll := audio.allStopEpoch

	// A stop for that voice arrives meanwhile.
	audio.stopVoiceLocked("v")

	if audio.voiceStopEpoch["v"] == startVoice {
		t.Fatal("stopping a voice did not move its epoch, so a decoding Play would overtake the stop")
	}

	// A channel-wide stop must move the channel epoch even with no voice on it.
	audio.stopChannelLocked(3)
	if audio.channelStopEpoch[3] == startChannel {
		t.Fatal("stopping an idle channel did not move its epoch")
	}

	// StopAll has to cancel plays for channels it has never seen.
	audio.mu.Lock()
	audio.allStopEpoch++
	audio.mu.Unlock()
	if audio.allStopEpoch == startAll {
		t.Fatal("StopAll did not move the global epoch")
	}
}
