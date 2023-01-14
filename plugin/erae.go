package plugin

/*
import (
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/vizicist/palette/engine"
	"gitlab.com/gomidi/midi/v2/drivers"
)

func init() {
	RegisterPlugin("erae", &Plugin_erae{
		patchName: "A",
		enabled:   false,
		prefix:    0x55,
		width:     0x2a,
		height:    0x18,
		zone:      1,
	})
}

type Plugin_erae struct {
	ctx       *engine.EngineContext
	patchName string
	enabled   bool
	prefix    byte
	width     int
	height    int
	zone      byte
	output    drivers.In
}

func (agent *Plugin_erae) Start(ctx *engine.EngineContext) {
	agent.EraeZoneClearDisplay(agent.zone)
	agent.output = ctx.OpenMIDIOutput("erae")
	agent.ctx = ctx
}

func (agent *Plugin_erae) Api(api string, apiargs map[string]string) (string, error) {
	return "", fmt.Errorf("Api needs work")
}

func (agent *Plugin_erae) OnCursorEvent(ce engine.CursorEvent) {
	engine.Info("Plugin_erae.onCursorEvent", "ce", ce)
}

func (agent *Plugin_erae) OnMidiEvent(me engine.MidiEvent) {
	bb := me.Msg.Bytes()
	s := ""
	for _, b := range bb {
		s += fmt.Sprintf(" 0x%02x", b)
	}
	engine.Info("HandleEraeMIDI", "bytes", s)
	if bb[1] == agent.prefix {
		agent.handleEraeSysEx(bb)
	}
}

func (agent *Plugin_erae) handleEraeSysEx(bb []byte) {
	// This assumes the RECEIVER PREFIX BYTES is exactly 1 byte
	if bb[2] == 0x7f {
		agent.handleBoundary(bb)
	} else {
		agent.handleFinger(bb)
	}
}

func (agent *Plugin_erae) handleBoundary(bb []byte) {
	expected := 7
	if len(bb) != expected {
		engine.Warn("handleBoundary: bad reply message", "expectedbytes", expected)
		return
	}
	zone := bb[3]
	width := bb[4]
	height := bb[5]
	engine.Info("BOUNDARY REPLY", "zone", zone, "width", width, "height", height)
}

var lastX float32
var lastY float32
var lastZ float32

func (agent *Plugin_erae) EraeFingerIndicator(zone, x, y byte) {
	engine.LogOfType("erae", "EraeFingerIndicator", "x", x, "y", y)
	rectw := byte(2)
	recth := byte(2)
	dim := 1.0
	for dim > 0 {
		// Should set color based on EraePatch
		red := byte(0)
		green := byte(0)
		blue := byte(0)
		alpha := byte(0x7f * dim)

		switch agent.patchName {
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

		agent.EraeZoneRectangle(zone, x, y, rectw, recth, red, green, blue)

		time.Sleep(100 * time.Millisecond)
		dim -= 0.05
	}
	// Erase it
	agent.EraeZoneRectangle(zone, x, y, rectw, recth, 0x00, 0x00, 0x00)
}

func (agent *Plugin_erae) handleFinger(bb []byte) {

	finger := bb[2] & 0x0f
	action := (bb[2] & 0xf0) >> 4
	zone := bb[3]
	xyzbytes := bb[4:18]
	chksum := bb[18]
	realbytes, chk := EraeUnbitize7chksum(xyzbytes)
	if chk != chksum {
		engine.Warn("handleFinger: chksum didn't match!  Ignoring finger message")
		return
	}
	rawx := Float32frombytes(realbytes[0:4])
	rawy := Float32frombytes(realbytes[4:8])
	rawz := Float32frombytes(realbytes[8:12])
	cid := fmt.Sprintf("%d", finger)

	x := rawx / float32(agent.width)
	y := rawy / float32(agent.height)
	z := rawz

	if engine.IsLogging("erae") {
		dx := lastX - x
		dy := lastY - y
		dz := lastZ - z
		engine.LogOfType("erae", "FINGER",
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

	edge := float32(0.1)
	newPatchName := ""
	if x < edge && y < edge {
		newPatchName = "A"
	} else if x < edge && y > (1.0-edge) {
		newPatchName = "B"
	} else if x > (1.0-edge) && y > (1.0-edge) {
		newPatchName = "C"
	} else if x > (1.0-edge) && y < edge {
		newPatchName = "D"
	}

	if newPatchName != "" {
		if newPatchName != agent.patchName {
			// Clear cursor state
			ce := engine.CursorEvent{
				ID:        cid,
				Source:    "erae",
				Timestamp: time.Now(),
				Ddu:       "clear",
			}
			agent.OnCursorEvent(ce)
			engine.LogOfType("erae", "Switching Erae to", "patch", newPatchName)
			agent.patchName = newPatchName
		}
		// We don't pass corner things through, even if we haven't changed the player
		return
	}

	// Do you need to round the value of rawx/y ??
	// if action == 0 {
	go agent.EraeFingerIndicator(zone, byte(rawx), byte(rawy))
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
		engine.Warn("handleFinger: invalid value for", "action", action)
		return
	}

	ce := engine.CursorEvent{
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

	agent.OnCursorEvent(ce)
}

func (agent *Plugin_erae) EraeWriteSysEx(bytes []byte) {
	engine.Warn("Plugin_erae.EraeWriteSysex needs work")
	// err := agent.output.Send(bytes)
	// if err != nil {
	// 	engine.LogError(err)
	// }
}

func (agent *Plugin_erae) EraeApiModeEnable() {
	bytes := []byte{0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x01,
		agent.prefix, 0xf7}
	agent.EraeWriteSysEx(bytes)
}

func (agent *Plugin_erae) EraeApiModeDisable() {
	bytes := []byte{0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x02, 0xf7}
	agent.EraeWriteSysEx(bytes)
}

func (agent *Plugin_erae) EraeZoneBoundaryRequest(zone byte) {
	bytes := []byte{0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x10,
		byte(zone), 0xf7}
	agent.EraeWriteSysEx(bytes)
}

func (agent *Plugin_erae) EraeZoneClearDisplay(zone byte) {
	bytes := []byte{0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x20,
		byte(zone), 0xf7}
	agent.EraeWriteSysEx(bytes)
}

func (agent *Plugin_erae) EraeZoneClearPixel(zone, x, y, r, g, b byte) {
	bytes := []byte{0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x21,
		byte(zone), x, y, r, g, b, 0xf7}
	agent.EraeWriteSysEx(bytes)
}

func (agent *Plugin_erae) EraeZoneRectangle(zone, x, y, w, h, r, g, b byte) {
	bytes := []byte{
		0xf0, 0x00, 0x21, 0x50, 0x00, 0x01, 0x00, 0x01,
		0x01, 0x01, 0x04, 0x22,
		byte(zone), x, y, w, h, r, g, b,
		0xf7,
	}
	agent.EraeWriteSysEx(bytes)
}

// 7-bitize an array of bytes and get the resulting checksum
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

// 7-unbitize an array of bytes and get the incomming checksum
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
		engine.Warn("Float32frombytes: length of bytes is not 4")
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

func testmain() {
	bytes := Float64bytes(math.Pi)
	fmt.Println(bytes)
	float := Float64frombytes(bytes)
	fmt.Println(float)
}

*/
