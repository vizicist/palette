package main

import (
	"context"
	"flag"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/0xcafed00d/joystick"
	"github.com/vizicist/palette/kit"

	"github.com/reugn/go-quartz/quartz"

	midi "gitlab.com/gomidi/midi/v2"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // autoregisters driver
)

func main() {

	kit.InitLog("monitor")

	pcheck := flag.Bool("engine", true, "Check Engine")
	pjsid := flag.Int("joystick", -1, "Joystick ID")

	flag.Parse()

	if *pcheck {
		kit.LogInfo("monitor is checking the engine.")
		go checkEngine()
	} else {
		kit.LogInfo("monitor is NOT checking the engine.")
	}

	go joystickMonitor(*pjsid)

	go midiMonitor("Logidy UMI3")

	go scheduler()

	select {}
}

func scheduler() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create scheduler
	sched := quartz.NewStdScheduler()

	// async start scheduler
	sched.Start(ctx)

	// create jobs
	// "0 15 10 * * ?"          Fire at 10:15am every day
	cronTrigger, _ := quartz.NewCronTrigger("0 0,15,30,45 * * * *")
	shellJob := quartz.NewShellJob("palette status")
	fjob := quartz.NewFunctionJob(func(_ context.Context) (int, error) {
		fmt.Printf("hi from function!\n")
		return 0, nil
	})

	// request, _ := http.NewRequest(http.MethodGet, "https://worldtimeapi.org/api/timezone/utc", nil)
	// curlJob := quartz.NewCurlJob(request)

	// functionJob := quartz.NewFunctionJob(func(_ context.Context) (int, error) { return 42, nil })

	// register jobs to scheduler
	// sched.ScheduleJob(ctx, shellJob, cronTrigger)
	sched.ScheduleJob(ctx, fjob, cronTrigger)
	sched.ScheduleJob(ctx, shellJob, cronTrigger)
	// sched.ScheduleJob(ctx, curlJob, quartz.NewSimpleTrigger(time.Second*7))
	// sched.ScheduleJob(ctx, functionJob, quartz.NewSimpleTrigger(time.Second*5))

	// stop scheduler
	// sched.Stop()

	// wait for all workers to exit
	// sched.Wait(ctx)

	select {}
}

func checkEngine() {
	tick := time.NewTicker(time.Second * 15)
	for {
		running, err := kit.IsRunningExecutable(kit.EngineExe)
		if err == nil && !running {
			kit.LogInfo("checkEngine: engine is not running, killing everything, monitor should restart engine.")
			kit.KillAllExceptMonitor()
			kit.LogInfo("checkEngine: restarting engine")
			fullexe := filepath.Join(kit.PaletteDir(), "bin", kit.EngineExe)
			_, err := kit.StartExecutableLogOutput(kit.EngineExe, fullexe)
			kit.LogIfError(err)
		}
		tm := <-tick.C
		_ = tm
	}
}

func mmttRealign() {
	kit.LogInfo("Begin mmttRealign")
	_, err := kit.MmttApi("align_start")
	kit.LogIfError(err)
}

func shutdownAndReboot() {
	kit.LogInfo("Begin of shutdownAndReboot")
	cmd := exec.Command("shutdown", "/r", "-t", "10")
	err := cmd.Run()
	if err != nil {
		kit.LogInfo("err in shutdownAndReboot")
	}
	kit.LogIfError(err)
	kit.LogInfo("End of shutdownAndReboot")
}

func joystickMonitor(jsid int) {

	var monitoredJoystick joystick.Joystick

	if jsid >= 0 {
		js, err := joystick.Open(jsid)
		if err != nil {
			kit.LogIfError(err)
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
			kit.LogInfo("joystick check", "j", j, "name", js.Name(), "buttoncount", count)
			if count == 8 {
				jsid = j
				monitoredJoystick = js
				break
			}
		}
		if jsid < 0 {
			kit.LogIfError(fmt.Errorf("joystickMonitor: disabled, unable to find joystick with 8 buttons"))
			return
		}
		kit.LogInfo("Found Ikkego joystick with 8 buttons", "jsid", jsid)
	}

	kit.LogInfo("joystickMonitor: listening", "name", monitoredJoystick.Name(), "buttoncount", monitoredJoystick.ButtonCount())

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
				kit.LogIfError(err)
			} else if errcount > 999 {
				kit.LogWarn("Too many joystick errors, aborting joystick monitoring")
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
					kit.LogInfo("Button went down...")
				} else {
					// Button just came back up.
					kit.LogInfo("Button came back up...")
					dt := time.Since(buttonDownTime[button])
					// Pay attention only if the button is down for more than a second.
					shortPress := 2 * time.Second
					longPress := 6 * time.Second
					if dt < shortPress {
						kit.LogInfo("BUTTON pressed, but not long enough to do anything", "button", button, "dt", dt)
					} else if dt < longPress {
						kit.LogInfo("BUTTON shortPress", "button", button, "dt", dt)
						kit.KillAllExceptMonitor()
					} else {
						kit.LogInfo("BUTTON longPress", "button", button, "dt", dt)
						shutdownAndReboot()
					}
				}
			}
		}

		tm := <-ticker.C
		_ = tm
	}
	kit.LogWarn("Joystick monitoring has terminated")
}

type NoteAction func()

func midiMonitor(port string) {

	defer midi.CloseDriver()

	in, err := midi.FindInPort(port)
	if err != nil {
		kit.LogIfError(err, "port", port)
		return
	}

	kit.LogInfo("midiMonitor: listening", "port", port)

	nnotes := 128
	noteDown := make([]bool, nnotes)
	noteDownTime := make([]time.Time, nnotes)
	noteAction := make([]NoteAction, nnotes)

	noteAction[60] = func() {
		kit.LogInfo("NoteAction: calling mmttRealign")
		mmttRealign()
	}
	noteAction[62] = func() {
		kit.LogInfo("NoteAction: calling shutdownAndReboot")
		shutdownAndReboot()
	}
	noteAction[64] = func() {
		kit.LogInfo("NoteAction: calling killAndRestart")
		kit.KillAllExceptMonitor()
	}

	stop, err := midi.ListenTo(in, func(msg midi.Message, timestampms int32) {
		var ch, pitch, vel uint8
		switch {
		case msg.GetNoteStart(&ch, &pitch, &vel):
			kit.LogOfType("midi", "NoteOn", "pitch", pitch, "chan", ch, "velocity", vel)
			if !noteDown[pitch] {
				noteDownTime[pitch] = time.Now()
				noteDown[pitch] = true
			}

		case msg.GetNoteEnd(&ch, &pitch):
			kit.LogOfType("midi", "NoteOff", "pitch", pitch, "chan", ch)
			if noteDown[pitch] {
				noteDown[pitch] = false
				// Note just came back up.
				dt := time.Since(noteDownTime[pitch])
				// Pay attention only if the note is down for more than a second.
				if dt > time.Second {
					kit.LogOfType("midi", "notegpress", "pitch", pitch, "dt", dt)
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
		kit.LogIfError(err)
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
