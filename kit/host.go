package kit

import (
	"github.com/hypebeast/go-osc/osc"
)

var TheHost Host

type Host interface {
	LogWarn(msg string, keysAndValues ...any)
	LogError(err error, keysAndValues ...any)
	LogInfo(msg string, keysAndValues ...any)
	LogOfType(logtypes string, msg string, keysAndValues ...any)
	GetParam(name string) (string, error)
	HandleIncomingMidiEvent(me MidiEvent)
	ResetAudio()
	SendToOscClients(msg *osc.Message)
}

func RegisterHost(host Host) {
	TheHost = host
}
