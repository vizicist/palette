package engine

import (
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

// This package is the external interface to register Plugins

type Plugin struct {
	Name     string
	Events   uint // see Event* bits
	callback PluginCallback
}

var Plugins = make(map[string]*Plugin)

func PluginSubject(pluginNUID string) string {
	return fmt.Sprintf("plugin.input.%s", pluginNUID)
}

// Bits for Events
const EventMidiInput = 0x01
const EventNoteOutput = 0x02
const EventCursor = 0x04
const EventAll = EventMidiInput | EventNoteOutput | EventCursor

type PluginCallback func(eventType string, eventData interface{})

// PluginRegister is intended for use by a plugin, in order to send
// a message (api call) to the engine over NATS, telling the engine
// to send particular event types to the named plugin.
func PluginRegister(pluginNUID string, name string, eventTypes string, callback PluginCallback) error {

	// equivalent to: palette register {name} {eventTypes}
	_, ok := Plugins[pluginNUID]
	if ok {
		return fmt.Errorf("PluginRegister: plugin %s is already registered", pluginNUID)
	}
	plugin := &Plugin{
		callback: callback,
	}

	subj := PluginSubject(pluginNUID)
	log.Printf("RegisterPlugin: subj=%s\n", subj)
	SubscribeNATS(subj, plugin.pluginCallback)

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

func (plugin *Plugin) pluginCallback(msg *nats.Msg) {
	data := string(msg.Data)
	args, err := StringMap(data)
	if err != nil {
		log.Printf("pluginCallback: err=%s\n", err)
		return
	}
	notestr, err := needStringArg("note", "pluginCallback", args)
	if err != nil {
		log.Printf("pluginCallback: err=%s\n", err)
		return
	}
	log.Printf("pluginCallback: notestr=%s\n", notestr)
	note, err := NoteFromString(notestr)
	if err != nil {
		log.Printf("pluginCallback: bad notestr - %s\n", notestr)
		return
	}
	plugin.callback("note", note)
}
