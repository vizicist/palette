package kit

import (
	"github.com/hypebeast/go-osc/osc"
)

type Host interface {

	Init() error
	Start()

	LogWarn(msg string, keysAndValues ...any)
	LogError(err error, keysAndValues ...any)
	LogInfo(msg string, keysAndValues ...any)
	IsLogging(name string) bool
	LogOfType(logtypes string, msg string, keysAndValues ...any)
	HandleIncomingMidiEvent(me MidiEvent)
	ResetAudio()
	ActivateAudio()
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
	EveryTick()

	ToFreeFramePlugin(patchName string, msg *osc.Message)
	SendEffectParam(patchName string, name string, value string)
	PortAndLayerNumForPatch(patchName string) (portnum int, layernum int)
	ShowText(msg string)
}