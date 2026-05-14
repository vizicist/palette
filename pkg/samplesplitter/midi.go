package samplesplitter

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	midi "gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
)

type MIDIManager struct {
	mu    sync.Mutex
	state *State
	audio *AudioManager
	in    drivers.In
	stop  func()
}

func NewMIDIManager(state *State, audio *AudioManager) (*MIDIManager, error) {
	return &MIDIManager{state: state, audio: audio}, nil
}

func (m *MIDIManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopLocked()
	return nil
}

func (m *MIDIManager) Ports() ([]string, error) {
	if m == nil {
		return nil, errors.New("MIDI backend is not initialized")
	}
	ins := midi.GetInPorts()
	ports := make([]string, len(ins))
	for i, in := range ins {
		ports[i] = in.String()
	}
	return ports, nil
}

func (m *MIDIManager) Start(portName string) (string, error) {
	if m == nil {
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
	stop, err := midi.ListenTo(in, m.handleMessage, midi.HandleError(func(err error) {
		if m.state != nil {
			m.state.SetMIDIStatus("", err)
		}
	}))
	if err != nil {
		_ = in.Close()
		if m.state != nil {
			m.state.SetMIDIStatus("", err)
		}
		return "", err
	}
	m.in = in
	m.stop = stop
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
	if m.stop != nil {
		m.stop()
		m.stop = nil
	}
	if m.in != nil {
		_ = m.in.Close()
		m.in = nil
	}
}

func (m *MIDIManager) resolveInput(portName string) (string, drivers.In, error) {
	ins := midi.GetInPorts()
	var exact drivers.In
	var matches []drivers.In
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

func (m *MIDIManager) handleMessage(msg midi.Message, _ int32) {
	var channel, key, velocity uint8
	switch {
	case msg.GetNoteOn(&channel, &key, &velocity):
		m.handleNoteOn(int(channel), int(key), int(velocity))
	case msg.GetNoteOff(&channel, &key, &velocity):
		m.handleNoteOff(int(channel), int(key), int(velocity))
	case len(msg) >= 3 && msg[0]&0xf0 == 0xe0:
		value := int(msg[1]) | (int(msg[2]) << 7)
		m.handlePitchBend(int(msg[0]&0x0f), value-8192)
	}
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
