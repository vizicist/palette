package engine

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"sync"
	"time"
)

var EraeRegion = "A"
var EraeEnabled = false
var EraeOutput *MIDIChannelOutput
var EraeInput *MIDIInput
var MyPrefix byte = 0x55
var EraeWidth int = 0x2a
var EraeHeight int = 0x18
var EraeZone byte = 1
var EraeMutex sync.RWMutex

func InitErae() {
	MIDI.openInput("Erae Touch")
	MIDI.openDeviceOutput("Erae Touch")
	EraeInput = MIDI.GetMidiInput("Erae Touch")
	EraeOutput = MIDI.GetMidiChannelOutput(PortChannel{"Erae Touch", 1})
	if EraeInput == nil || EraeOutput == nil {
		log.Printf("Unable to open Erae Touch.  Is the EraeLab application open?\n")
		return
	}
	EraeEnabled = true
	// engine.Debug.OSC = true
	// engine.Debug.Erae = true
	EraeZoneClearDisplay(EraeZone)
}

func HandleEraeMIDI(event MidiEvent) {
	if event.SysEx != nil {
		bb := event.SysEx
		if Debug.Erae {
			s := ""
			for _, b := range bb {
				s += fmt.Sprintf(" 0x%02x", b)
			}
			log.Printf("HandleEraeMIDI: bytes = %s\n", s)
		}
		if bb[1] == MyPrefix {
			handleEraeSysex(bb)
		}
	} else {
		log.Printf("HandleEraeMIDI: no action for event=%+v\n", event)
	}
}

func handleEraeSysex(bb []byte) {
	// This assumes the RECEIVER PREFIX BYTES is exactly 1 byte
	if bb[2] == 0x7f {
		handleBoundary(bb)
	} else {
		handleFinger(bb)
	}
}

func handleBoundary(bb []byte) {
	expected := 7
	if len(bb) != expected {
		log.Printf("handleBoundary: bad reply message, length isn't %d bytes\n", expected)
		return
	}
	zone := bb[3]
	width := bb[4]
	height := bb[5]
	log.Printf("BOUNDARY REPLY zone=%d width=%d height=%d\n", zone, width, height)
}

var lastX float32
var lastY float32
var lastZ float32

func EraeFingerIndicator(zone, x, y byte) {
	if Debug.Erae {
		log.Printf("EraeFingerIndicator!!   x,y=%d,%d\n", x, y)
	}
	rectw := byte(2)
	recth := byte(2)
	dim := 1.0
	for dim > 0 {
		// Should set color based on EraeRegion
		red := byte(0)
		green := byte(0)
		blue := byte(0)
		alpha := byte(0x7f * dim)
		switch EraeRegion {
		case "A":
			red = alpha
		case "B":
			green = alpha
		case "C":
			blue = alpha
		case "D":
			red = alpha
			green = alpha
			blue = alpha
		}

		EraeZoneRectangle(zone, x, y, rectw, recth, red, green, blue)

		time.Sleep(100 * time.Millisecond)
		dim -= 0.05
	}
	// Erase it
	EraeZoneRectangle(zone, x, y, rectw, recth, 0x00, 0x00, 0x00)
}

func handleFinger(bb []byte) {

	finger := bb[2] & 0x0f
	action := (bb[2] & 0xf0) >> 4
	zone := bb[3]
	xyzbytes := bb[4:18]
	chksum := bb[18]
	realbytes, chk := EraeUnbitize7chksum(xyzbytes)
	if chk != chksum {
		log.Printf("handleFinger: chksum didn't match!  Ignoring finger message\n")
		return
	}
	rawx := Float32frombytes(realbytes[0:4])
	rawy := Float32frombytes(realbytes[4:8])
	rawz := Float32frombytes(realbytes[8:12])
	cid := fmt.Sprintf("%d", finger)

	x := rawx / float32(EraeWidth)
	y := rawy / float32(EraeHeight)
	z := rawz

	if Debug.Erae {
		dx := lastX - x
		dy := lastY - y
		dz := lastZ - z
		log.Printf("FINGER finger=%d action=%d zone=%d x=%f y=%f z=%f   dx=%f dy=%f dz=%f\n", finger, action, zone, x, y, z, dx, dy, dz)
		lastX = x
		lastY = y
		lastZ = z
	}

	// If the position is in one of the corners,
	// we change the region to that corner.

	edge := float32(0.1)
	newregion := ""
	if x < edge && y < edge {
		newregion = "A"
	} else if x < edge && y > (1.0-edge) {
		newregion = "B"
	} else if x > (1.0-edge) && y > (1.0-edge) {
		newregion = "C"
	} else if x > (1.0-edge) && y < edge {
		newregion = "D"
	}

	if newregion != "" {
		if newregion != EraeRegion {
			// Clear cursor state from existing region
			ce := CursorDeviceEvent{
				ID:        cid,
				Source:    "erae",
				Timestamp: time.Now(),
				Ddu:       "clear",
			}
			TheEngine.CursorManager.handleCursorDeviceEvent(ce, true)
			if Debug.Erae {
				log.Printf("Switching Erae to region %s", newregion)
			}
			EraeRegion = newregion
		}
		// We don't pass corner things through, even if we haven't changed the region
		return
	}

	// Do you need to round the value of rawx/y ??
	// if action == 0 {
	go EraeFingerIndicator(zone, byte(rawx), byte(rawy))
	// }

	// make the coordinate space match OpenGL and Freeframe
	// yNorm = 1.0 - yNorm

	// Make sure we don't send anyting out of bounds
	if y < 0.0 {
		y = 0.0
	} else if y > 1.0 {
		y = 1.0
	}
	if x < 0.0 {
		x = 0.0
	} else if x > 1.0 {
		x = 1.0
	}

	var ddu = ""
	switch action {
	case 0:
		ddu = "down"
	case 1:
		ddu = "drag"
	case 2:
		ddu = "up"
	default:
		log.Printf("handleFinger: invalid value for action = %d\n", action)
		return
	}

	ce := CursorDeviceEvent{
		ID:        cid,
		Source:    "erae",
		Timestamp: time.Now(),
		Ddu:       ddu,
		X:         x,
		Y:         y,
		Z:         z,
		Area:      0.0,
	}
	// XXX - should Fresh be true???

	TheEngine.CursorManager.handleCursorDeviceEvent(ce, true)
}

func EraeWriteSysex(bytes []byte) {
	EraeMutex.Lock()
	defer EraeMutex.Unlock()
	EraeOutput.midiDeviceOutput.WriteSysex(bytes)
}

func EraeApiModeEnable() {
	bytes := []byte{0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x01,
		MyPrefix, 0xf7}
	EraeWriteSysex(bytes)
}

func EraeApiModeDisable() {
	bytes := []byte{0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x02, 0xf7}
	EraeWriteSysex(bytes)
}

func EraeZoneBoundaryRequest(zone byte) {
	bytes := []byte{0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x10,
		byte(zone), 0xf7}
	EraeWriteSysex(bytes)
}

func EraeZoneClearDisplay(zone byte) {
	bytes := []byte{0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x20,
		byte(zone), 0xf7}
	EraeWriteSysex(bytes)
}

func EraeZoneClearPixel(zone, x, y, r, g, b byte) {
	bytes := []byte{0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x21,
		byte(zone), x, y, r, g, b, 0xf7}
	EraeWriteSysex(bytes)
}

func EraeZoneRectangle(zone, x, y, w, h, r, g, b byte) {
	bytes := []byte{
		0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x22,
		byte(zone), x, y, w, h, r, g, b,
		0xf7,
	}
	EraeWriteSysex(bytes)
}

/**
* 7-bitize an array of bytes and get the resulting checksum
* Algorithm taken from Erae documentation.
* Return value is the output bytes and checksum
 */
func EraeBitize7chksum(in []byte) (out []byte, chksum byte) {
	chksum = 0
	i := 0
	outsize := 0
	inlen := len(in)
	out = make([]byte, 2*inlen) // too large, but the return value is sliced
	for i < inlen {
		out[outsize] = 0
		for j := 0; (j < 7) && (i+j < inlen); j++ {
			out[outsize] |= (in[i+j] & 0x80) >> (j + 1)
			out[outsize+j+1] = in[i+j] & 0x7F
			chksum ^= out[outsize+j+1]
		}
		chksum ^= out[outsize]
		i += 7
		outsize += 8
	}
	return out[0:outsize], chksum
}

/*
* 7-unbitize an array of bytes and get the incomming checksum
* Algorithm taken from Erae documentation.
* Return value is the output bytes and checksum
 */
func EraeUnbitize7chksum(in []byte) ([]byte, byte) {
	inlen := len(in)
	chksum := byte(0)
	i := 0
	outsize := 0
	out := make([]byte, 2*len(in)) // too large, but the return value is sliced
	for i < inlen {
		chksum ^= in[i]
		for j := 0; (j < 7) && (j+1+i < inlen); j++ {
			b := ((in[i] << (j + 1)) & 0x80) | in[i+j+1]
			out[outsize+j] = b
			chksum ^= in[i+j+1]
		}
		i += 8
		outsize += 7
	}
	return out[0:outsize], chksum
}

func Float32frombytes(bytes []byte) float32 {
	if len(bytes) != 4 {
		log.Printf("Float32frombytes: length of bytes is not 4\n")
		return 0
	}
	bits := binary.LittleEndian.Uint32(bytes)
	float := math.Float32frombits(bits)
	return float
}

func Float32bytes(float float32) []byte {
	bits := math.Float32bits(float)
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, bits)
	return bytes
}

/*
func testmain() {
	bytes := Float64bytes(math.Pi)
	fmt.Println(bytes)
	float := Float64frombytes(bytes)
	fmt.Println(float)
}
*/
