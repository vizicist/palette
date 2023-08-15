package main

import (
	"flag"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/0xcafed00d/joystick"

	midi "gitlab.com/gomidi/midi/v2"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // autoregisters driver

	"github.com/vizicist/palette/hostwin"
	"github.com/vizicist/palette/kit"
)

func main() {

	host := hostwin.NewHost("monitor")
	kit.RegisterHost(host)

	pcheck := flag.Bool("engine", true, "Check Engine")
	pjsid := flag.Int("joystick", -1, "Joystick ID")

	flag.Parse()

	if *pcheck {
		host.LogInfo("monitor is checking the engine.")
		go checkEngine()
	} else {
		host.LogInfo("monitor is NOT checking the engine.")
	}

	go joystickMonitor(*pjsid)

	go midiMonitor("Logidy UMI3")

	select {}
}

func checkEngine() {
	tick := time.NewTicker(time.Second * 15)
	for {
		if !hostwin.IsRunningExecutable(hostwin.EngineExe) {
			hostwin.LogInfo("checkEngine: engine is not running, killing everything, monitor should restart engine.")
			hostwin.KillAllExceptMonitor()
			hostwin.LogInfo("checkEngine: restarting engine")
			fullexe := filepath.Join(hostwin.PaletteDir(), "bin", hostwin.EngineExe)
			err := hostwin.StartExecutableLogOutput(hostwin.EngineExe, fullexe)
			hostwin.LogIfError(err)
		}
		tm := <-tick.C
		_ = tm
	}
}

func mmttRealign() {
	hostwin.LogInfo("Begin mmttRealign")
	_, err := hostwin.MmttApi("align_start")
	hostwin.LogIfError(err)
}

func shutdownAndReboot() {
	hostwin.LogInfo("Begin of shutdownAndReboot")
	cmd := exec.Command("shutdown", "/r", "-t", "10")
	err := cmd.Run()
	if err != nil {
		hostwin.LogInfo("err in shutdownAndReboot")
	}
	hostwin.LogIfError(err)
	hostwin.LogInfo("End of shutdownAndReboot")
}

func joystickMonitor(jsid int) {

	var monitoredJoystick joystick.Joystick

	if jsid >= 0 {
		js, err := joystick.Open(jsid)
		if err != nil {
			hostwin.LogIfError(err)
			return
		}
		monitoredJoystick = js
	} else {
		// Search for the first joystick that has 8 buttons.
		// The Sensel Morphs show up as a joystick with 16 buttons,
		// while the Ikkego footswitch (the one we want) has 8 buttons.
		for j := 0; j < 10; j++ {
			js, err := joystick.Open(j)
			if err != nil {
				break
			}
			count := js.ButtonCount()
			hostwin.LogInfo("joystick check", "j", j, "name", js.Name(), "buttoncount", count)
			if count == 8 {
				jsid = j
				monitoredJoystick = js
				break
			}
		}
		if jsid < 0 {
			hostwin.LogIfError(fmt.Errorf("joystickMonitor: disabled, unable to find joystick with 8 buttons"))
			return
		}
		hostwin.LogInfo("Found Ikkego joystick with 8 buttons", "jsid", jsid)
	}

	hostwin.LogInfo("joystickMonitor: listening", "name", monitoredJoystick.Name(), "buttoncount", monitoredJoystick.ButtonCount())

	ticker := time.NewTicker(time.Second)
	buttonDown := make([]bool, monitoredJoystick.ButtonCount())
	buttonDownTime := make([]time.Time, monitoredJoystick.ButtonCount())

	errcount := 0
	for {
		jinfo, err := monitoredJoystick.Read()
		if err == nil {
			errcount = 0
		} else {
			errcount++
			if errcount < 4 {
				hostwin.LogIfError(err)
			} else if errcount > 999 {
				hostwin.LogWarn("Too many joystick errors, aborting joystick monitoring")
				break
			}
			continue
		}

		for button := 0; button < monitoredJoystick.ButtonCount(); button++ {
			isdown := jinfo.Buttons&(1<<uint32(button)) != 0
			if isdown != buttonDown[button] {
				buttonDown[button] = isdown
				if isdown {
					// Button just went down.
					buttonDownTime[button] = time.Now()
					hostwin.LogInfo("Button went down...")
				} else {
					// Button just came back up.
					hostwin.LogInfo("Button came back up...")
					dt := time.Since(buttonDownTime[button])
					// Pay attention only if the button is down for more than a second.
					shortPress := 2 * time.Second
					longPress := 6 * time.Second
					if dt < shortPress {
						hostwin.LogInfo("BUTTON pressed, but not long enough to do anything", "button", button, "dt", dt)
					} else if dt < longPress {
						hostwin.LogInfo("BUTTON shortPress", "button", button, "dt", dt)
						hostwin.KillAllExceptMonitor()
					} else {
						hostwin.LogInfo("BUTTON longPress", "button", button, "dt", dt)
						shutdownAndReboot()
					}
				}
			}
		}

		tm := <-ticker.C
		_ = tm
	}
	hostwin.LogWarn("Joystick monitoring has terminated")
}

type NoteAction func()

func midiMonitor(port string) {

	defer midi.CloseDriver()

	in, err := midi.FindInPort(port)
	if err != nil {
		hostwin.LogIfError(err, "port", port)
		return
	}

	hostwin.LogInfo("midiMonitor: listening", "port", port)

	nnotes := 128
	noteDown := make([]bool, nnotes)
	noteDownTime := make([]time.Time, nnotes)
	noteAction := make([]NoteAction, nnotes)

	noteAction[60] = func() {
		hostwin.LogInfo("NoteAction: calling mmttRealign")
		mmttRealign()
	}
	noteAction[62] = func() {
		hostwin.LogInfo("NoteAction: calling shutdownAndReboot")
		shutdownAndReboot()
	}
	noteAction[64] = func() {
		hostwin.LogInfo("NoteAction: calling killAndRestart")
		hostwin.KillAllExceptMonitor()
	}

	stop, err := midi.ListenTo(in, func(msg midi.Message, timestampms int32) {
		var ch, pitch, vel uint8
		switch {
		case msg.GetNoteStart(&ch, &pitch, &vel):
			hostwin.LogOfType("midi", "NoteOn", "pitch", pitch, "chan", ch, "velocity", vel)
			if !noteDown[pitch] {
				noteDownTime[pitch] = time.Now()
				noteDown[pitch] = true
			}

		case msg.GetNoteEnd(&ch, &pitch):
			hostwin.LogOfType("midi", "NoteOff", "pitch", pitch, "chan", ch)
			if noteDown[pitch] {
				noteDown[pitch] = false
				// Note just came back up.
				dt := time.Since(noteDownTime[pitch])
				// Pay attention only if the note is down for more than a second.
				if dt > time.Second {
					hostwin.LogOfType("midi", "notegpress", "pitch", pitch, "dt", dt)
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
		hostwin.LogIfError(err)
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
