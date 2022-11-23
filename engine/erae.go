package engine

import (
	"encoding/binary"
	"fmt"
	"math"
	"sync"
	"time"

	"gitlab.com/gomidi/midi/v2/drivers"
)

var EraePlayerName = "A"
var EraePlayer *Player
var EraeEnabled = false
var MyPrefix byte = 0x55
var EraeWidth int = 0x2a
var EraeInput drivers.In
var EraeOutput drivers.Out

var EraeHeight int = 0x18
var EraeZone byte = 1
var EraeMutex sync.RWMutex

func InitErae() {
	// MIDI.openInput("Erae Touch")
	// MIDI.openDeviceOutput("Erae Touch")
	if EraeInput == nil || EraeOutput == nil {
		Warn("Unable to open Erae Touch.  Is the EraeLab application open?")
		return
	}
	EraeEnabled = true
	// engine.Debug.OSC = true
	// engine.Debug.Erae = true
	EraeZoneClearDisplay(EraeZone)
}

func HandleEraeMIDI(event MidiEvent) {
	if event.msg.Bytes != nil {
		bb := event.msg.Bytes()
		s := ""
		for _, b := range bb {
			s += fmt.Sprintf(" 0x%02x", b)
		}
		Info("HandleEraeMIDI", "bytes", s)
		if bb[1] == MyPrefix {
			handleEraeSysEx(bb)
		}
	} else {
		Warn("HandleEraeMIDI: no action for", "event", event)
	}
}

func handleEraeSysEx(bb []byte) {
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
		Warn("handleBoundary: bad reply message", "expectedbytes", expected)
		return
	}
	zone := bb[3]
	width := bb[4]
	height := bb[5]
	Info("BOUNDARY REPLY", "zone", zone, "width", width, "height", height)
}

var lastX float32
var lastY float32
var lastZ float32

func EraeFingerIndicator(zone, x, y byte) {
	DebugLogOfType("erae", "EraeFingerIndicator", "x", x, "y", y)
	rectw := byte(2)
	recth := byte(2)
	dim := 1.0
	for dim > 0 {
		// Should set color based on EraePlayer
		red := byte(0)
		green := byte(0)
		blue := byte(0)
		alpha := byte(0x7f * dim)
		switch EraePlayerName {
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
		Warn("handleFinger: chksum didn't match!  Ignoring finger message")
		return
	}
	rawx := Float32frombytes(realbytes[0:4])
	rawy := Float32frombytes(realbytes[4:8])
	rawz := Float32frombytes(realbytes[8:12])
	cid := fmt.Sprintf("%d", finger)

	x := rawx / float32(EraeWidth)
	y := rawy / float32(EraeHeight)
	z := rawz

	if IsLogging("erae") {
		dx := lastX - x
		dy := lastY - y
		dz := lastZ - z
		DebugLogOfType("erae", "FINGER",
			"finger", finger,
			"action", action,
			"zone", zone,
			"x", x, "y", y, "z", z,
			"dx", dx, "dy", dy, "dz", dz)
		lastX = x
		lastY = y
		lastZ = z
	}

	// If the position is in one of the corners,
	// we change the player to that corner.

	edge := float32(0.1)
	newplayerName := ""
	if x < edge && y < edge {
		newplayerName = "A"
	} else if x < edge && y > (1.0-edge) {
		newplayerName = "B"
	} else if x > (1.0-edge) && y > (1.0-edge) {
		newplayerName = "C"
	} else if x > (1.0-edge) && y < edge {
		newplayerName = "D"
	}

	if newplayerName != "" {
		player, err := TheRouter().PlayerManager.GetPlayer(newplayerName)
		if err != nil {
			Warn("player not found", "player", newplayerName)
			return
		}
		if newplayerName != EraePlayerName {
			// Clear cursor state from existing player
			ce := CursorEvent{
				ID:        cid,
				Source:    "erae",
				Timestamp: time.Now(),
				Ddu:       "clear",
			}
			player.HandleCursorEvent(ce)
			DebugLogOfType("erae", "Switching Erae to", "player", newplayerName)
			EraePlayerName = newplayerName
			EraePlayer = player
		}
		// We don't pass corner things through, even if we haven't changed the player
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
		Warn("handleFinger: invalid value for", "action", action)
		return
	}

	ce := CursorEvent{
		ID:        cid,
		Source:    "erae",
		Timestamp: time.Now(),
		Ddu:       ddu,
		X:         x,
		Y:         y,
		Z:         z,
		Area:      0.0,
	}
	// XXX - should Fresh be trued???

	EraePlayer.HandleCursorEvent(ce)
}

func EraeWriteSysEx(bytes []byte) {
	EraeMutex.Lock()
	defer EraeMutex.Unlock()
	err := EraeOutput.Send(bytes)
	if err != nil {
		LogError(err)
	}
}

func EraeApiModeEnable() {
	bytes := []byte{0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x01,
		MyPrefix, 0xf7}
	EraeWriteSysEx(bytes)
}

func EraeApiModeDisable() {
	bytes := []byte{0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x02, 0xf7}
	EraeWriteSysEx(bytes)
}

func EraeZoneBoundaryRequest(zone byte) {
	bytes := []byte{0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x10,
		byte(zone), 0xf7}
	EraeWriteSysEx(bytes)
}

func EraeZoneClearDisplay(zone byte) {
	bytes := []byte{0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x20,
		byte(zone), 0xf7}
	EraeWriteSysEx(bytes)
}

func EraeZoneClearPixel(zone, x, y, r, g, b byte) {
	bytes := []byte{0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x21,
		byte(zone), x, y, r, g, b, 0xf7}
	EraeWriteSysEx(bytes)
}

func EraeZoneRectangle(zone, x, y, w, h, r, g, b byte) {
	bytes := []byte{
		0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x22,
		byte(zone), x, y, w, h, r, g, b,
		0xf7,
	}
	EraeWriteSysEx(bytes)
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
		Warn("Float32frombytes: length of bytes is not 4")
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
