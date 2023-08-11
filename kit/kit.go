package kit

type Kit struct {
	midiOctaveShift      int
	midisetexternalscale bool
	midithru             bool
	midiThruScadjust     bool
	midiNumDown          int
}

func InitKit() {

	TheCursorManager = NewCursorManager()
	TheScheduler = NewScheduler()
	TheAttractManager = NewAttractManager()
	TheQuadPro = NewQuadPro()

	enabled, err := GetParamBool("engine.attractenabled")
	LogIfError(err)
	TheAttractManager.SetAttractEnabled(enabled)

	InitSynths()
	TheQuadPro.Start()
	go TheScheduler.Start()

}

func NewKit() *Kit {
	return &Kit{
	}
}

func (k *Kit) ScheduleCursorEvent(ce CursorEvent) {

	// schedule CursorEvents rather than handle them right away.
	// This makes it easier to do looping, among other benefits.

	ScheduleAt(CurrentClick(), ce.Tag, ce)
}

func (k *Kit) HandleMidiEvent(me MidiEvent) {
	TheHost.SendToOscClients(MidiToOscMsg(me))

	if k.midithru {
		LogOfType("midi","PassThruMIDI", "msg", me.Msg)
		if k.midiThruScadjust {
			LogWarn("PassThruMIDI, midiThruScadjust needs work", "msg", me.Msg)
		}
		ScheduleAt(CurrentClick(), "midi", me.Msg)
	}
	if k.midisetexternalscale {
		k.handleMIDISetScaleNote(me)
	}

	err := TheQuadPro.onMidiEvent(me)
	LogIfError(err)

	if TheErae.enabled {
		TheErae.onMidiEvent(me)
	}
}

func (k *Kit) handleMIDISetScaleNote(me MidiEvent) {
	if !me.HasPitch() {
		return
	}
	pitch := me.Pitch()
	if me.Msg.Is(midi.NoteOnMsg) {

		if k.midiNumDown < 0 {
			// may happen when there's a Read error that misses a noteon
			k.midiNumDown = 0
		}

		// If there are no notes held down (i.e. this is the first), clear the scale
		if k.midiNumDown == 0 {
			LogOfType("scale", "Clearing external scale")
			ClearExternalScale()
		}
		SetExternalScale(pitch, true)
		k.midiNumDown++

		// adjust the octave shift based on incoming MIDI
		newshift := 0
		if pitch < 60 {
			newshift = -1
		} else if pitch > 72 {
			newshift = 1
		}
		if newshift != r.midiOctaveShift {
			k.midiOctaveShift = newshift
			LogOfType("midi","MidiOctaveShift changed", "octaveshift", r.midiOctaveShift)
		}
	} else if me.Msg.Is(midi.NoteOffMsg) {
		k.midiNumDown--
	}
}

