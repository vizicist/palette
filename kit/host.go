package kit

import (
	"github.com/hypebeast/go-osc/osc"
)

type Host interface {
	Init() error
	Start()
	SayDone()

	LogWarn(msg string, keysAndValues ...any)
	LogError(err error, keysAndValues ...any)
	LogInfo(msg string, keysAndValues ...any)
	// IsLogging(name string) bool
	// LogOfType(logtypes string, msg string, keysAndValues ...any)
	ArchiveLogs() error

	EngineHttpApi(api string, args ...string) (map[string]string, error)
	MmttHttpApi(api string) (map[string]string, error)
	HandleIncomingMidiEvent(me MidiEvent)
	ResetAudio()
	ActivateAudio()
	FileExists(path string) bool
	SendToOscClients(msg *osc.Message)
	GenerateVisualsFromCursor(ce CursorEvent, patchName string)
	InputEventLock()
	InputEventUnlock()
	OpenFakeChannelOutput(port string, channel int) *MIDIPortChannelState
	OpenChannelOutput(PortChannel) *MIDIPortChannelState
	GetPortChannelState(PortChannel) (*MIDIPortChannelState, error)
	SendMIDI(output any, bytes []byte) error
	StartRunning(process string) error
	KillProcess(process string) error
	SetMidiInput(name string) error
	ShowClip(clipNum int)

	GetConfigFileData(filename string) ([]byte, error)
	SavedFileList(category string) ([]string, error)
	SaveDataInFile(data []byte, category string, filename string) error
	GetSavedData(category string, filename string) ([]byte, error)

	EveryTick()

	ToFreeFramePlugin(patchName string, msg *osc.Message)
	SendEffectParam(patchName string, name string, value string)
	PortAndLayerNumForPatch(patchName string) (portnum int, layernum int)
	ShowText(msg string)
}

func FileExists(path string) bool {
	return TheHost.FileExists(path)
}

var OscOutput = true

func SendToOscClients(msg *osc.Message) {
	if OscOutput {
		TheHost.SendToOscClients(msg)
	}
}

func ArchiveLogs() error {
	return TheHost.ArchiveLogs()
}
