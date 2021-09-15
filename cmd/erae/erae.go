package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os/signal"
	"syscall"

	"github.com/vizicist/palette/engine"
	"github.com/vizicist/portmidi"
)

var midiOut *engine.MidiOutput

func main() {

	signal.Ignore(syscall.SIGHUP)
	signal.Ignore(syscall.SIGINT)

	engine.InitMIDI("Erae Touch")

	flag.Parse()

	midiOut = engine.MIDI.NewMidiOutput("Erae Touch", 1)
	// midiOut := engine.MIDI.NewMidiOutput("01. Internal MIDI", 1)
	// midiIn := engine.MIDI.NewMidiInput("01. Internal MIDI")

	r := engine.TheRouter()
	r.SetMIDIEventHandler(handleMIDI)
	go r.StartMIDI()
	go r.ListenForLocalDeviceInputsForever() // never returns

	args := flag.Args()
	nargs := len(args)

	cmd := "test" // default cmd
	if nargs != 0 {
		cmd = args[0]
	}
	switch cmd {
	case "test":
		do_test()
	default:
		log.Printf("Unrecognized: %s\n", args[0])
	}
	log.Printf("Blocking forever....\n")
	select {} // block forever
}

func handleMIDI(event portmidi.Event) {
	if event.SysEx != nil {
		s := ""
		for _, b := range event.SysEx {
			s += fmt.Sprintf(" 0x%02x", b)
		}
		log.Printf("handleMIDI: ReadEvent sysex bytes= [%s ]\n", s)
	} else {
		log.Printf("handleMIDI: ReadEvent event=%+v\n", event)
	}
}

func do_test() {
	erae_api_mode_enable()
	erae_zone_boundary_request(0)
	erae_zone_clear_display(0)
	erae_zone_rectangle(0, 0, 0, 0x2a, 0x18, 0x10, 0x10, 0x10)

	// These values should come from the boundary_request response
	boundaryWidth := 0x2a
	boundaryHeight := 0x18

	for i := 0; i < 100; i++ {
		x := byte(rand.Intn(boundaryWidth))
		y := byte(rand.Intn(boundaryHeight))
		erae_zone_clear_pixel(0, x, y, 0x37, 0x10, 0x10)
	}
	/*
		erae_zone_clear_pixel(0, 2, 3, 0x37, 0x10, 0x10)
		erae_zone_clear_pixel(0, 2, 4, 0x37, 0x10, 0x10)
		erae_zone_clear_pixel(0, 3, 2, 0x37, 0x10, 0x10)
		erae_zone_clear_pixel(0, 4, 2, 0x37, 0x10, 0x10)
		erae_zone_clear_pixel(0, 5, 2, 0x37, 0x10, 0x10)
	*/
}

func erae_api_mode_enable() {
	bytes := []byte{0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x01,
		0x55, 0xf7}
	midiOut.WriteSysex(bytes)
}

func erae_zone_boundary_request(zone int) {
	bytes := []byte{0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x10,
		byte(zone), 0xf7}
	midiOut.WriteSysex(bytes)
}

func erae_zone_clear_display(zone int) {
	bytes := []byte{0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x20,
		byte(zone), 0xf7}
	midiOut.WriteSysex(bytes)
}

func erae_zone_clear_pixel(zone int, x, y, r, g, b byte) {
	bytes := []byte{0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x21,
		byte(zone), x, y, r, g, b, 0xf7}
	midiOut.WriteSysex(bytes)
}

func erae_zone_rectangle(zone int, x, y, w, h, r, g, b byte) {
	bytes := []byte{
		0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x22,
		byte(zone), x, y, w, h, r, g, b,
		0xf7,
	}
	midiOut.WriteSysex(bytes)
}
