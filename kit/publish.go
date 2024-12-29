package kit

import (
	"fmt"
	"time"
)

var time0 = time.Now()

// PublishCursorEvent xxx
func PublishCursorEvent(ce CursorEvent) {
	if !natsIsConnected {
		return // silent
	}
	dt := time.Since(time0)
	regionvalue := ""
	if ce.Tag != "" {
		regionvalue = "\"tag\": \"" + ce.Tag + "\", "
	}
	params := "{ " +
		// "\"tag\": \"" + ce.Tag + "\", " +
		"\"cid\": \"" + fmt.Sprintf("%d", ce.Gid) + "\", " +
		regionvalue +
		"\"ddu\": \"" + ce.Ddu + "\", " +
		"\"millisecs\": \"" + fmt.Sprintf("%d", dt.Milliseconds()) + "\", " +
		"\"x\": \"" + fmt.Sprintf("%f", ce.Pos.X) + "\", " +
		"\"y\": \"" + fmt.Sprintf("%f", ce.Pos.Y) + "\", " +
		"\"z\": \"" + fmt.Sprintf("%f", ce.Pos.Z) + "\", " +
		"\"area\": \"" + fmt.Sprintf("%f", ce.Area) + "\" }"

	NatsPublishFromEngine("event.cursor", params)
}

// PublishMIDIDeviceEvent xxx
func PublishMIDIDeviceEvent(me MidiEvent) {
	if !natsIsConnected {
		return // silent
	}
	dt := time.Since(time0)
	// NOTE: we ignore the Timestamp on the MIDIDeviceEvent
	// and use our own, so the timestamps are consistent with
	// the ones on Cursor events
	params := "{ " +
		"\"host\": \"" + Hostname() + "\", " +
		"\"event\": \"" + "midi" + "\", " +
		// "\"timestamp\": \"" + fmt.Sprintf("%d", me.Timestamp) + "\", " +
		"\"millisecs\": \"" + fmt.Sprintf("%d", dt.Milliseconds()) + "\", " +
		"\"bytes\": \"" + fmt.Sprintf("%v", me.Msg.Bytes()) + "\" }"

	NatsPublishFromEngine("event.midi", params)
}

// PublishSpriteEvent xxx
func PublishSpriteEvent(x, y, z float32) {
	if !natsIsConnected {
		return // silent
	}
	params := "{ " +
		"\"host\": \"" + Hostname() + "\", " +
		"\"x\": \"" + fmt.Sprintf("%f", x) + "\", " +
		"\"y\": \"" + fmt.Sprintf("%f", y) + "\", " +
		"\"z\": \"" + fmt.Sprintf("%f", z) + "\" }"

	NatsPublishFromEngine("event.sprite", params)
}
