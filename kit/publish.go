package kit

import (
	"time"
)

var time0 = time.Now()

// PublishCursorEvent xxx
func PublishCursorEvent(ce CursorEvent) {
	if !natsIsConnected {
		return // silent
	}
	data := map[string]any{
		"cid":       ce.GID,
		"ddu":       ce.Ddu,
		"millisecs": time.Since(time0).Milliseconds(),
		"x":         ce.Pos.X,
		"y":         ce.Pos.Y,
		"z":         ce.Pos.Z,
		"area":      ce.Area,
	}
	if ce.Tag != "" {
		data["tag"] = ce.Tag
	}

	NatsPublishFromEngine("event.cursor", data)
}

// PublishMIDIDeviceEvent xxx
func PublishMIDIDeviceEvent(me MidiEvent) {
	if !natsIsConnected {
		return // silent
	}
	// NOTE: we ignore the Timestamp on the MIDIDeviceEvent
	// and use our own, so the timestamps are consistent with
	// the ones on Cursor events
	data := map[string]any{
		"host":      Hostname(),
		"event":     "midi",
		"millisecs": time.Since(time0).Milliseconds(),
		"bytes":     me.Msg.Bytes(),
	}

	NatsPublishFromEngine("event.midi", data)
}

// PublishSpriteEvent xxx
func PublishSpriteEvent(x, y, z float32) {
	if !natsIsConnected {
		return // silent
	}
	data := map[string]any{
		"host": Hostname(),
		"x":    x,
		"y":    y,
		"z":    z,
	}
	NatsPublishFromEngine("event.sprite", data)
}
