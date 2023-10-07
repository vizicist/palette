package kit

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

	HandleIncomingMidiEvent(me MidiEvent)
	ResetAudio()
	ActivateAudio()
	FileExists(path string) bool
	GenerateVisualsFromCursor(ce CursorEvent, patchName string)
	OpenFakeChannelOutput(port string, channel int) *MIDIPortChannelState
	OpenChannelOutput(PortChannel) *MIDIPortChannelState
	GetPortChannelState(PortChannel) (*MIDIPortChannelState, error)
	SendMIDI(output any, bytes []byte) error
	StartRunning(process string) error
	KillProcess(process string) error
	SetMidiInput(name string) error
	// ShowClip(clipNum int)

	GetConfigFileData(filename string) ([]byte, error)
	ConfigFilePath(filename string) string
	SavedFileList(category string) ([]string, error)
	SaveDataInFile(data []byte, category string, filename string) error
	GetSavedData(category string, filename string) ([]byte, error)

	EveryTick()

	// ToFreeFramePlugin(patchName string, msg *osc.Message)
	// SendEffectParam(patchName string, name string, value string)
	// ShowText(msg string)
}

func FileExists(path string) bool {
	return TheHost.FileExists(path)
}

var OscOutput = true

/*
func SendToOscClients(msg *osc.Message) {
	if OscOutput {
		SendToOscClients(msg)
	}
}
*/

func ArchiveLogs() error {
	return TheHost.ArchiveLogs()
}
