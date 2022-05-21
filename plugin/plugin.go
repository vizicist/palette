package plugin

import "fmt"

// This package lets plugins interact with the Palette engine.

type Plugin struct {
	Name   string
	Events uint // see Event* bits
}

// Bits for Events
const EventMidiInput = 0x01
const EventMidiOutput = 0x02
const EventCursor = 0x04
const EventAll = EventMidiInput | EventMidiOutput | EventCursor

type Callback func(eventType string, eventData string)

func Register(name string, eventTypes uint, callback Callback) error {
	// Should send NATS api to Palette engine
	return fmt.Errorf("plugini.Register needs implementation")
}
