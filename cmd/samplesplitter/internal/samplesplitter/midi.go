package samplesplitter

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	oldmidi "gitlab.com/gomidi/midi"
	"gitlab.com/gomidi/midi/reader"
	"gitlab.com/gomidi/rtmididrv"
)

type MIDIManager struct {
	mu     sync.Mutex
	state  *State
	audio  *AudioManager
	driver *rtmididrv.Driver
	in     oldmidi.In
}

func NewMIDIManager(state *State, audio *AudioManager) (*MIDIManager, error) {
	driver, err := rtmididrv.New()
	if err != nil {
		return nil, err
	}
	return &MIDIManager{state: state, audio: audio, driver: driver}, nil
}

func (m *MIDIManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopLocked()
	if m.driver == nil {
		return nil
	}
	return m.driver.Close()
}

func (m *MIDIManager) Ports() ([]string, error) {
	if m == nil || m.driver == nil {
		return nil, errors.New("MIDI backend is not initialized")
	}
	ins, err := m.driver.Ins()
	if err != nil {
		return nil, err
	}
	ports := make([]string, len(ins))
	for i, in := range ins {
		ports[i] = in.String()
	}
	return ports, nil
}

func (m *MIDIManager) Start(portName string) (string, error) {
	if m == nil || m.driver == nil {
		return "", errors.New("MIDI backend is not initialized")
	}
	if portName == "" {
		portName = DefaultMIDIPortName
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	resolved, in, err := m.resolveInput(portName)
	if err != nil {
		if m.state != nil {
			m.state.SetMIDIStatus("", err)
		}
		return "", err
	}

	m.stopLocked()
	if err := in.Open(); err != nil {
		if m.state != nil {
			m.state.SetMIDIStatus("", err)
		}
		return "", err
	}
	midiReader := reader.New(
		reader.NoLogger(),
		reader.NoteOn(func(_ *reader.Position, channel, key, velocity uint8) {
			m.handleNoteOn(int(channel), int(key), int(velocity))
		}),
		reader.NoteOff(func(_ *reader.Position, channel, key, velocity uint8) {
			m.handleNoteOff(int(channel), int(key), int(velocity))
		}),
		reader.Pitchbend(func(_ *reader.Position, channel uint8, value int16) {
			m.handlePitchBend(int(channel), int(value))
		}),
	)
	if err := midiReader.ListenTo(in); err != nil {
		_ = in.Close()
		if m.state != nil {
			m.state.SetMIDIStatus("", err)
		}
		return "", err
	}
	m.in = in
	if m.state != nil {
		m.state.SetMIDIStatus(resolved, nil)
	}
	return resolved, nil
}

func (m *MIDIManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopLocked()
}

func (m *MIDIManager) stopLocked() {
	if m.in != nil {
		_ = m.in.StopListening()
		_ = m.in.Close()
		m.in = nil
	}
}

func (m *MIDIManager) resolveInput(portName string) (string, oldmidi.In, error) {
	ins, err := m.driver.Ins()
	if err != nil {
		return "", nil, err
	}
	var exact oldmidi.In
	var matches []oldmidi.In
	for _, in := range ins {
		name := in.String()
		if name == portName {
			exact = in
			break
		}
		if strings.HasPrefix(name, portName) {
			matches = append(matches, in)
		}
	}
	if exact != nil {
		return exact.String(), exact, nil
	}
	if len(matches) == 1 {
		return matches[0].String(), matches[0], nil
	}
	if len(matches) > 1 {
		names := make([]string, len(matches))
		for i, in := range matches {
			names[i] = in.String()
		}
		return "", nil, fmt.Errorf("ambiguous port %q: %s", portName, strings.Join(names, ", "))
	}
	return "", nil, fmt.Errorf("unknown port %q", portName)
}

func (m *MIDIManager) handleNoteOn(channel, key, velocity int) {
	if m.state == nil {
		return
	}
	m.state.RecordMIDIActivity()
	if velocity == 0 {
		m.state.PlanNoteOff(key, channel)
		if m.audio != nil {
			m.audio.StopNote(channel, key)
		}
		return
	}
	req, err := m.state.PlanNoteOn(key, velocity, channel)
	if err != nil {
		return
	}
	if m.audio != nil {
		if err := m.audio.Play(req); err != nil {
			m.state.SetAudioStatus(false, err)
		}
	}
}

func (m *MIDIManager) handleNoteOff(channel, key, _ int) {
	if m.state == nil {
		return
	}
	m.state.RecordMIDIActivity()
	m.state.PlanNoteOff(key, channel)
	if m.audio != nil {
		m.audio.StopNote(channel, key)
	}
}

func (m *MIDIManager) handlePitchBend(channel, bend int) {
	if m.state == nil {
		return
	}
	m.state.RecordMIDIActivity()
	m.state.SetPitchBend(channel, bend)
}
