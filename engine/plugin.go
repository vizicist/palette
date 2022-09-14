package engine

import (
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
)

// This package is the external interface that Plugins use
// to register themselves, receive events, and do things
// like play notes or spawn sprites.
// Should probably be a separate package.

type Plugin struct {
	onCursorEvent func(e CursorDeviceEvent)
}

func NewPlugin() *Plugin {
	p := &Plugin{}
	return p
}

func (p *Plugin) OnCursorEvent(f func(e CursorDeviceEvent)) {
	p.onCursorEvent = f
}

func PluginOutputSubject(id string) string {
	return fmt.Sprintf("plugin.output.%s", id)
}

type PluginCallback func(args map[string]string)

func (p *Plugin) RunForever() error {

	err := SubscribeNATS("palette.output.event", p.pluginNATSCallback)
	if err != nil {
		return err
	}
	select {} // block forever
}

func (plugin *Plugin) pluginNATSCallback(msg *nats.Msg) {
	data := string(msg.Data)
	args, err := StringMap(data)
	if err != nil {
		log.Printf("natsCallback: err=%s\n", err)
		return
	}
	if Debug.NATS {
		log.Printf("pluginNATSCallback args=%+v\n", args)
	}
	switch args["event"] {
	case "cursor_down", "cursor_drag", "cursor_up":
		ce := ArgsToCursorDeviceEvent(args)
		if plugin.onCursorEvent != nil {
			plugin.onCursorEvent(ce)
		}
	}
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

/*
// PlayNote is intended for use by a plugin, to play a Note
func PlayNote(note *Note, source string) error {

	params := JsonObject("source", source, "note", note.String())
	args := JsonObject(
		// "nuid", MyNUID(),
		"api", "sound.playnote",
		"params", jsonEscape(params),
	)
	return NATSPublish("palette.api", args)
}
*/
