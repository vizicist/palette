package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
	"path/filepath"

	"github.com/vizicist/palette/engine"
	"github.com/0xcafed00d/joystick"

	midi "gitlab.com/gomidi/midi/v2"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // autoregisters driver
)

func readJoystick(js joystick.Joystick) {
	jinfo, err := js.Read()

	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		return
	}

	for button := 0; button < js.ButtonCount(); button++ {
		if jinfo.Buttons&(1<<uint32(button)) != 0 {
			fmt.Printf("BUTTON %d\n", button)
		}
	}
}

func main() {

	engine.InitLog("monitor")

	jsid := 0
	if len(os.Args) > 1 {
		i, err := strconv.Atoi(os.Args[1])
		if err != nil {
			fmt.Println(err)
			return
		}
		jsid = i
	}

	pm := engine.NewProcessManager()

	pm.AddProcess("engine",EngineProcessInfo())

	go processCheck(pm)

	go joystickMonitor(jsid)

	go midiMonitor("Logidy UMI3")

	select {}
}

func EngineProcessInfo() *engine.ProcessInfo {
	fullpath := filepath.Join(engine.PaletteDir(), "bin", "palette_engine.exe")
	if !engine.FileExists(fullpath) {
		engine.LogWarn("Engine not found in default location", "fullpath", fullpath)
		return nil
	}
	exe := filepath.Base(fullpath)
	return engine.NewProcessInfo(exe, fullpath, "", nil)
}

func processCheck(pm *engine.ProcessManager) {
	tick := time.NewTicker(time.Second*5)
	for true {
		if ! pm.IsRunning("engine") {
			engine.LogInfo("processCheck: engine is NOT running, restarting it")
			err := pm.StartRunning("engine")
			engine.LogIfError(err)
		}
		tm := <- tick.C
		_ = tm
	}
}

func joystickMonitor(jsid int) {

	js, err := joystick.Open(jsid)

	engine.LogInfo("joystickMonitor", "name", js.Name(), "buttoncount", js.ButtonCount())
	if err != nil {
		engine.LogIfError(err)
		return
	}

	ticker := time.NewTicker(time.Millisecond * 100)

	for {
		readJoystick(js)
		tm := <-ticker.C
		_ = tm
	}
}

func midiMonitor(port string) {

	defer midi.CloseDriver()

	in, err := midi.FindInPort(port)
	if err != nil {
		engine.LogWarn("midiMonitor: Unable to find port", "port", port)
		return
	}

	stop, err := midi.ListenTo(in, func(msg midi.Message, timestampms int32) {
		var ch, key, vel uint8
		switch {
		// var bt []byte
		// case msg.GetSysEx(&bt):
		// 	fmt.Printf("got sysex: % X\n", bt)
		case msg.GetNoteStart(&ch, &key, &vel):
			engine.LogInfo("NoteOn: note %v channel %v velocity %v", midi.Note(key), ch, vel)
		case msg.GetNoteEnd(&ch, &key):
			engine.LogInfo("NoteOff: note %v channel %v", midi.Note(key), ch)
		default:
			// ignore
		}
	}, midi.UseSysEx())

	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return
	}

	forever := true
	if forever {
		select {}
	} else {
		time.Sleep(time.Second)
		stop()
	}
}