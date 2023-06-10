package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

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

	jsid := 0
	if len(os.Args) > 1 {
		i, err := strconv.Atoi(os.Args[1])
		if err != nil {
			fmt.Println(err)
			return
		}
		jsid = i
	}

	go joystickMonitor(jsid)

	go midiMonitor("Logidy UMI3")

	select {}
}

func joystickMonitor(jsid int) {

	js, jserr := joystick.Open(jsid)

	fmt.Printf("Joystick: %s has %d buttons\n", js.Name(), js.ButtonCount())

	if jserr != nil {
		fmt.Println(jserr)
		return
	}

	ticker := time.NewTicker(time.Millisecond * 100)

	for doQuit := false; !doQuit; {
		tm := <-ticker.C
		_ = tm
		readJoystick(js)
	}
}

func midiMonitor(port string) {

	defer midi.CloseDriver()

	in, err := midi.FindInPort(port)
	if err != nil {
		fmt.Printf("Unable to find port %s\n", port)
		return
	}

	stop, err := midi.ListenTo(in, func(msg midi.Message, timestampms int32) {
		var bt []byte
		var ch, key, vel uint8
		switch {
		case msg.GetSysEx(&bt):
			fmt.Printf("got sysex: % X\n", bt)
		case msg.GetNoteStart(&ch, &key, &vel):
			fmt.Printf("starting note %s on channel %v with velocity %v\n", midi.Note(key), ch, vel)
		case msg.GetNoteEnd(&ch, &key):
			fmt.Printf("ending note %s on channel %v\n", midi.Note(key), ch)
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
		time.Sleep(time.Second * 10)
		stop()
	}
}