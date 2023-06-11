package main

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/0xcafed00d/joystick"
	"github.com/vizicist/palette/engine"

	midi "gitlab.com/gomidi/midi/v2"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // autoregisters driver
)

func main() {

	engine.InitLog("monitor")

	jsid := 0
	if len(os.Args) > 1 {
		i, err := strconv.Atoi(os.Args[1])
		if err != nil {
			engine.LogIfError(err)
			return
		}
		jsid = i
	}

	go checkEngine()

	go joystickMonitor(jsid)

	go midiMonitor("Logidy UMI3")

	select {}
}

func checkEngine() {
	tick := time.NewTicker(time.Second * 15)
	for {
		if !engine.IsRunningExecutable(engine.EngineExe) {

			engine.LogInfo("processCheck: engine is not running, killing everything and restarting engine.")
			engine.KillAll()

			// restart just the engine,  engine.autostart value will determine what else gets started
			fullexe := filepath.Join(engine.PaletteDir(), "bin", engine.PaletteExe)
			err := engine.StartExecutableLogOutput("palette", fullexe, "start", "engine")
			engine.LogIfError(err)
		}
		tm := <-tick.C
		_ = tm
	}
}

type ButtonAction func()

func joystickMonitor(jsid int) {
	js, err := joystick.Open(jsid)
	if err != nil {
		engine.LogIfError(err)
		return
	}

	if err != nil {
		engine.LogIfError(err)
		return
	}

	engine.LogInfo("joystickMonitor: listening", "name", js.Name(), "buttoncount", js.ButtonCount())

	ticker := time.NewTicker(time.Millisecond * 100)
	buttonDown := make([]bool, js.ButtonCount())
	buttonDownTime := make([]time.Time, js.ButtonCount())
	buttonAction := make([]ButtonAction, js.ButtonCount())

	buttonAction[0] = func() {
		engine.LogInfo("ButtonAction0!!")
	}
	buttonAction[1] = func() {
		engine.LogInfo("ButtonAction1!!")
	}

	for {
		jinfo, err := js.Read()
		if err != nil {
			engine.LogIfError(err)
			continue
		}

		for button := 0; button < js.ButtonCount(); button++ {
			isdown := jinfo.Buttons&(1<<uint32(button)) != 0
			if isdown != buttonDown[button] {
				// engine.LogInfo("BUTTON change", "button", button, "isdown", isdown)
				buttonDown[button] = isdown
				if isdown {
					// Button just went down.
					buttonDownTime[button] = time.Now()
				} else {
					// Button just came back up.
					dt := time.Since(buttonDownTime[button])
					// Pay attention only if the button is down for more than a second.
					if dt > time.Second {
						engine.LogInfo("BUTTON longpress", "button", button, "dt", dt)
						if buttonAction[button] != nil {
							buttonAction[button]()
						}
					}
				}
			}
		}

		tm := <-ticker.C
		_ = tm
	}
}

type NoteAction func()

func midiMonitor(port string) {

	defer midi.CloseDriver()

	in, err := midi.FindInPort(port)
	if err != nil {
		engine.LogIfError(err, "port", port)
		return
	}

	engine.LogInfo("midiMonitor: listening", "port", port)

	nnotes := 128
	noteDown := make([]bool, nnotes)
	noteDownTime := make([]time.Time, nnotes)
	noteAction := make([]NoteAction, nnotes)

	noteAction[60] = func() {
		engine.LogInfo("NoteAction60!!")
	}
	noteAction[62] = func() {
		engine.LogInfo("NoteAction62!!")
	}
	noteAction[64] = func() {
		engine.LogInfo("NoteAction64!!")
	}

	stop, err := midi.ListenTo(in, func(msg midi.Message, timestampms int32) {
		var ch, pitch, vel uint8
		switch {
		case msg.GetNoteStart(&ch, &pitch, &vel):
			engine.LogOfType("midi","NoteOn", "pitch", pitch, "chan", ch, "velocity", vel)
			if !noteDown[pitch] {
				noteDownTime[pitch] = time.Now()
				noteDown[pitch] = true
			}

		case msg.GetNoteEnd(&ch, &pitch):
			engine.LogOfType("midi","NoteOff", "pitch", pitch, "chan", ch)
			if noteDown[pitch] {
				noteDown[pitch] = false
				// Note just came back up.
				dt := time.Since(noteDownTime[pitch])
				// Pay attention only if the note is down for more than a second.
				if dt > time.Second {
					engine.LogOfType("midi","notegpress", "pitch", pitch, "dt", dt)
					if noteAction[pitch] != nil {
						noteAction[pitch]()
					}
				}
			}
		// var bt []byte
		// case msg.GetSysEx(&bt):
		// 	fmt.Printf("got sysex: % X\n", bt)
		default:
			// ignore
		}
	}, midi.UseSysEx())

	if err != nil {
		engine.LogIfError(err)
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
