package engine

import (
	"time"
)

// This package is the external interface to register Plugins

type Plugin struct {
	Name   string
	Events uint // see Event* bits
}

type PluginRef struct {
	Name   string
	Events uint // see Event* bits
	Active bool
}

func NewPluginRef(name string) *PluginRef {
	return &PluginRef{
		Name:   name,
		Active: false,
	}
}

// Bits for Events
const EventMidiInput = 0x01
const EventMidiOutput = 0x02
const EventCursor = 0x04
const EventAll = EventMidiInput | EventMidiOutput | EventCursor

type PluginCallback func(eventType string, eventData string)

// RegisterPlugin is intended for use by a plugin, in order to send
// a message (api call) to the engine over NATS, telling the engine
// to send particular event types to the named plugin.
func RegisterPlugin(name string, eventTypes string, callback PluginCallback) error {

	// equivalent to: palette register {name} {eventTypes}
	params := "{ " +
		"\"plugin\": \"" + name + "\", " +
		"\"events\": \"" + eventTypes + "\"" +
		"}"
	args := "{ " +
		"\"nuid\": \"" + MyNUID() + "\", " +
		"\"api\": \"" + "register" + "\", " +
		"\"params\": \"" + jsonEscape(params) +
		"\" }"
	timeout := 60 * time.Second
	_, err := NATSRequest("palette.api", args, timeout)
	return err
}
