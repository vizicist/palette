package kit

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"gitlab.com/gomidi/midi/v2/drivers"
)

var TheErae *Erae

func NewErae() *Erae {
	return &Erae{
		apienabled: false,
		patchName:  "A",
		prefix:     0x55,
		width:      0x2a,
		height:     0x18,
		zone:       1,
		output:     nil,
	}
}

type Erae struct {
	apienabled bool
	patchName  string
	prefix     byte
	width      int
	height     int
	zone       byte
	output     drivers.Out
}

func (erae *Erae) Start() {
	if erae.apienabled {
		erae.EraeZoneClearDisplay(erae.zone)
	}
	// erae.output = TheMidiIO.OpenMIDIOutput("erae")
}

func (erae *Erae) onMidiEvent(me MidiEvent) {
	bb := me.Msg.Bytes()
	s := ""
	for _, b := range bb {
		s += fmt.Sprintf(" 0x%02x", b)
	}
	LogInfo("HandleEraeMIDI", "bytes", s)
	if bb[1] == erae.prefix {
		erae.handleEraeSysEx(bb)
	}
}

func (erae *Erae) handleEraeSysEx(bb []byte) {
	// This assumes the RECEIVER PREFIX BYTES is exactly 1 byte
	if bb[2] == 0x7f {
		erae.handleBoundary(bb)
	} else {
		erae.handleFinger(bb)
	}
}

func (erae *Erae) handleBoundary(bb []byte) {
	expected := 7
	if len(bb) != expected {
		LogWarn("handleBoundary: bad reply message", "expectedbytes", expected)
		return
	}
	zone := bb[3]
	width := bb[4]
	height := bb[5]
	LogInfo("BOUNDARY REPLY", "zone", zone, "width", width, "height", height)
}

var lastX float64
var lastY float64
var lastZ float64

func (erae *Erae) EraeFingerIndicator(zone, x, y byte) {
	LogOfType("erae", "EraeFingerIndicator", "x", x, "y", y)
	rectw := byte(2)
	recth := byte(2)
	dim := 1.0
	for dim > 0 {
		// Should set color based on EraePatch
		red := byte(0)
		green := byte(0)
		blue := byte(0)
		alpha := byte(0x7f * dim)

		switch erae.patchName {
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

		erae.EraeZoneRectangle(zone, x, y, rectw, recth, red, green, blue)

		time.Sleep(100 * time.Millisecond)
		dim -= 0.05
	}
	// Erase it
	erae.EraeZoneRectangle(zone, x, y, rectw, recth, 0x00, 0x00, 0x00)
}

func (erae *Erae) handleFinger(bb []byte) {

	finger := bb[2] & 0x0f
	action := (bb[2] & 0xf0) >> 4
	zone := bb[3]
	xyzbytes := bb[4:18]
	chksum := bb[18]
	realbytes, chk := EraeUnbitize7chksum(xyzbytes)
	if chk != chksum {
		LogWarn("handleFinger: chksum didn't match!  Ignoring finger message")
		return
	}
	rawx := Float32frombytes(realbytes[0:4])
	rawy := Float32frombytes(realbytes[4:8])
	rawz := Float32frombytes(realbytes[8:12])
	gid := int(finger)

	x := float64(rawx) / float64(erae.width)
	y := float64(rawy) / float64(erae.height)
	z := float64(rawz)

	if IsLogging("erae") {
		dx := lastX - x
		dy := lastY - y
		dz := lastZ - z
		LogOfType("erae", "FINGER",
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
	// we change the patch to that corner.

	edge := 0.1
	tag := ""
	if x < edge && y < edge {
		tag = "A"
	} else if x < edge && y > (1.0-edge) {
		tag = "B"
	} else if x > (1.0-edge) && y > (1.0-edge) {
		tag = "C"
	} else if x > (1.0-edge) && y < edge {
		tag = "D"
	}

	if tag != "" {
		if tag != erae.patchName {
			// Clear cursor state
			ce := NewCursorEvent(gid, tag, "clear", CursorPos{})
			LogWarn("HandleFinger: needs work here")
			// erae.OnCursorEvent(ce)
			_ = ce
			LogOfType("erae", "Switching Erae to", "patch", tag)
			erae.patchName = tag
		}
		// We don't pass corner things through, even if we haven't changed the player
		return
	}

	// Do you need to round the value of rawx/y ??
	// if action == 0 {
	go erae.EraeFingerIndicator(zone, byte(rawx), byte(rawy))
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

	var ddu string
	switch action {
	case 0:
		ddu = "down"
	case 1:
		ddu = "drag"
	case 2:
		ddu = "up"
	default:
		LogWarn("handleFinger: invalid value for", "action", action)
		return
	}

	pos := CursorPos{x, y, z}

	ce := NewCursorEvent(gid, tag, ddu, pos)
	_ = ce
	// XXX - should Fresh be trued???
	LogWarn("HandleFinger: should be sending CursorEvent")
	// erae.OnCursorEvent(ce)
}

func (erae *Erae) EraeWriteSysEx(bytes []byte) {
	LogWarn("Erae.EraeWriteSysex needs work")
	// err := erae.output.Send(bytes)
	// if err != nil {
	// 	LogIfError(err)
	// }
}

func (erae *Erae) EraeAPIModeEnable() {
	bytes := []byte{0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x01,
		erae.prefix, 0xf7}
	erae.EraeWriteSysEx(bytes)
	erae.apienabled = true
}

func (erae *Erae) EraeAPIModeDisable() {
	bytes := []byte{0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x02, 0xf7}
	erae.EraeWriteSysEx(bytes)
	erae.apienabled = false
}

func (erae *Erae) EraeZoneBoundaryRequest(zone byte) {
	bytes := []byte{0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x10,
		byte(zone), 0xf7}
	erae.EraeWriteSysEx(bytes)
}

func (erae *Erae) EraeZoneClearDisplay(zone byte) {
	bytes := []byte{0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x20,
		byte(zone), 0xf7}
	erae.EraeWriteSysEx(bytes)
}

func (erae *Erae) EraeZoneClearPixel(zone, x, y, r, g, b byte) {
	bytes := []byte{0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x21,
		byte(zone), x, y, r, g, b, 0xf7}
	erae.EraeWriteSysEx(bytes)
}

func (erae *Erae) EraeZoneRectangle(zone, x, y, w, h, r, g, b byte) {
	bytes := []byte{
		0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x22,
		byte(zone), x, y, w, h, r, g, b,
		0xf7,
	}
	erae.EraeWriteSysEx(bytes)
}

// EraeBitize7chksum - 7-bitize an array of bytes and get the resulting checksum
// Algorithm taken from Erae documentation.
// Return value is the output bytes and checksum
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

// EraeUnbitize7chksum - 7-unbitize an array of bytes and get the incomming checksum
// Algorithm taken from Erae documentation.
// Return value is the output bytes and checksum
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
		LogWarn("Float32frombytes: length of bytes is not 4")
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
