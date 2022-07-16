package engine

import (
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nuid"
)

// This package is the external interface that Plugins use
// to register themselves, receive events, and do things
// like play notes or spawn sprites.
// Should probably be a separate package.

type Plugin struct {
	id       string
	callback PluginCallback
}

// This allows for multiple plugins to be registered in a single
var Plugins = make(map[string]*Plugin)

func PluginInputSubject(id string) string {
	return fmt.Sprintf("plugin.input.%s", id)
}

func PluginOutputSubject(id string) string {
	return fmt.Sprintf("plugin.output.%s", id)
}

func (plugin *Plugin) ID() string {
	return plugin.id
}

type PluginCallback func(pluginid string, eventType string, eventData interface{})

// PluginRegister is intended for use by a plugin, in order to send
// a message (api call) to the engine over NATS, telling the engine
// to send particular event types to the named plugin.
func PluginRegister(eventTypes string, callback PluginCallback) (*Plugin, error) {

	pluginid := nuid.Next()
	_, ok := Plugins[pluginid]
	if ok {
		return nil, fmt.Errorf("PluginRegister: %s is already registered!?", pluginid)
	}
	plugin := &Plugin{
		id:       pluginid,
		callback: callback,
	}
	Plugins[pluginid] = plugin

	inputSubject := PluginInputSubject(pluginid)

	log.Printf("PluginRegister: pluginid=%s subj=%s eventType=%s\n", pluginid, inputSubject, eventTypes)

	SubscribeNATS(inputSubject, plugin.natsCallback)

	params := JsonObject(
		"pluginid", pluginid,
		"events", eventTypes,
	)
	args := JsonObject(
		"nuid", MyNUID(),
		"api", "register",
		"params", jsonEscape(params),
	)
	timeout := 60 * time.Second
	_, err := NATSRequest("palette.api", args, timeout)
	return plugin, err
}

func (plugin *Plugin) natsCallback(msg *nats.Msg) {
	data := string(msg.Data)
	args, err := StringMap(data)
	if err != nil {
		log.Printf("natsCallback: err=%s\n", err)
		return
	}
	notestr, err := needStringArg("note", "natsCallback", args)
	if err != nil {
		log.Printf("natsCallback: err=%s\n", err)
		return
	}
	// log.Printf("natsCallback: notestr=%s\n", notestr)
	note, err := NoteFromString(notestr)
	if err != nil {
		log.Printf("natsCallback: bad notestr - %s\n", notestr)
		return
	}
	plugin.callback(plugin.ID(), "note", note)
}

func JsonObject(args ...string) string {
	if len(args)%2 != 0 {
		log.Printf("ApiParams: odd number of arguments, args=%v\n", args)
		return "{}"
	}
	params := ""
	sep := ""
	for n := range args {
		if n%2 == 0 {
			params = params + sep + "\"" + args[n] + "\": \"" + args[n+1] + "\""
		}
		sep = ", "
	}
	return "{" + params + "}"
}

// PlayNote is intended for use by a plugin, to play a Note
func (plugin *Plugin) PlayNote(note *Note) error {

	params := JsonObject("source", plugin.ID(), "note", note.String())
	args := JsonObject(
		"nuid", MyNUID(),
		"api", "sound.playnote",
		"params", jsonEscape(params),
	)
	return NATSPublish("palette.api", args)
}
