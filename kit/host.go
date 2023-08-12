package kit

import (
	"github.com/hypebeast/go-osc/osc"
)

var TheHost Host

type Host interface {

	Start()

	LogWarn(msg string, keysAndValues ...any)
	LogError(err error, keysAndValues ...any)
	LogInfo(msg string, keysAndValues ...any)
	IsLogging(name string) bool
	LogOfType(logtypes string, msg string, keysAndValues ...any)
	GetParam(name string) (string, error)
	HandleIncomingMidiEvent(me MidiEvent)
	ResetAudio()
	SendToOscClients(msg *osc.Message)
	GenerateVisualsFromCursor(ce CursorEvent, patchName string)
	SaveDataInFile(data []byte, category string, filename string) error
	SavedFileList(category string) ([]string, error)
	GetSavedData(category string, filename string) ([]byte, error)
	GetConfigFileData(filename string) ([]byte, error)
	InputEventLock()
	InputEventUnlock()
	OpenFakeChannelOutput(port string, channel int) *MIDIPortChannelState
	OpenChannelOutput(PortChannel) *MIDIPortChannelState
	GetPortChannelState(PortChannel) (*MIDIPortChannelState,error)
	SendMIDI([]byte)

	SaveCurrent() error
	Uptime() float64
	EveryTick()

	ToFreeFramePlugin(patchName string, msg *osc.Message)
	SendEffectParam(patchName string, name string, value string)
	PortAndLayerNumForPatch(patchName string) (portnum int, layernum int)
	ShowText(msg string)
}

func RegisterHost(host Host) {
	TheHost = host
}