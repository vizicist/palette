package hostwin

import (
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/hypebeast/go-osc/osc"
	"github.com/vizicist/palette/kit"
	"gitlab.com/gomidi/midi/v2/drivers"
)

type HostWin struct {
	logName          string
	done             chan bool

	// recordingIndex int
	// recordingFile  *os.File
	// recordingPath  string
	// recordingMutex sync.RWMutex
	// currentPitchOffset  *atomic.Int32

}

var TheWinHost *HostWin
var TheRand *rand.Rand
var LocalAddress = "127.0.0.1"

func NewHost(logname string) *HostWin {

	h := &HostWin{
		logName:          logname,
	}
	BiduleClient = osc.NewClient(LocalAddress, BidulePort)
	TheWinHost = h
	InitLog(h.logName)
	TheProcessManager = NewProcessManager()
	return h
}

func (h HostWin) Init() error {

	TheProcessManager.AddBuiltins()

	// Fixed rand sequence, better for testing
	TheRand = rand.New(rand.NewSource(1))

	TheMidiIO = NewMidiIO()

	return nil
}

func (h HostWin) Start() {

	// This need to be done after engine parameters are loaded

	h.done = make(chan bool)

	go TheMidiIO.Start()

	go h.ResetAudio()

	// if ParamBool("mmtt.depth") {
	// 	go DepthRunForever()
	// }
}
func Start() {

	err := LoadMorphs()
	if err != nil {
		LogWarn("StartCursorInput: LoadMorphs", "err", err)
	}

	go StartMorph(kit.ScheduleCursorEvent, 1.0)
}


func (h HostWin) EveryTick() {
	TheProcessManager.checkProcess()
}

func (h HostWin) SaveDataInFile(data []byte, category string, filename string) error {
	path, err := WritableSavedFilePath(category, filename)
	if err != nil {
		LogIfError(err)
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (h HostWin) GetSavedData(category string, filename string) (bytes []byte, err error) {
	if !strings.HasSuffix(filename, ".json") {
		filename += ".json"
	}
	path, err := ReadableSavedFilePath(category, filename)
	if err != nil {
		LogIfError(err)
		return nil, err
	}
	return os.ReadFile(path)
}

func (h HostWin) GetConfigFileData(filename string) ([]byte, error) {
	path := h.ConfigFilePath(filename)
	return os.ReadFile(path)
}

func (h HostWin) GenerateVisualsFromCursor(ce kit.CursorEvent, patchName string) {
	// send an OSC message to Resolume
	msg := kit.CursorToOscMsg(ce)
	kit.ToFreeFramePlugin(patchName, msg)

}

func (h HostWin) HandleIncomingMidiEvent(me kit.MidiEvent) {
	err := fmt.Errorf("HandleIncomingMidiEvent needs work")
	kit.LogIfError(err)
}

func (h HostWin) LogError(err error, keysAndValues ...any) {
	LogError(err, keysAndValues...)
}

func (h HostWin) LogIfError(err error, keysAndValues ...any) {
	LogIfError(err, keysAndValues...)
}

func (h HostWin) LogInfo(msg string, keysAndValues ...any) {
	LogInfo(msg, keysAndValues...)
}

func (h HostWin) LogWarn(msg string, keysAndValues ...any) {
	LogWarn(msg, keysAndValues...)
}

// func (h HostWin) LogOfType(logtypes string, msg string, keysAndValues ...any) {
// 	LogOfType(logtypes, msg, keysAndValues...)
// }

func (h HostWin) SendMIDI(output any, bytes []byte) error {
	driverout, ok := output.(drivers.Out)
	if !ok {
		LogWarn("unable to convert output to drivers.Out")
		return fmt.Errorf("unable to convert output to drivers.Out")
	}
	return driverout.Send(bytes)
}

func (h HostWin) WaitTillDone() {
	<-h.done
}

func (h HostWin) SayDone() {
	h.done <- true
}

func (h HostWin) StartRunning(process string) (err error) {
	return TheProcessManager.StartRunning(process)
}

func (h HostWin) KillProcess(process string) (err error) {
	return TheProcessManager.KillProcess(process)
}

func (h HostWin) SetMidiInput(midiInputName string) error {
	return TheMidiIO.SetMidiInput(midiInputName)
}
