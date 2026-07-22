package morph

// Pure-Go implementation of the Sensel Morph serial (CDC-ACM) protocol.
//
// This is a from-scratch reimplementation of the register/frame protocol used
// by Sensel's MIT-licensed SDK (github.com/sensel/sensel-api). It talks to the
// Morph's virtual serial port directly, so it needs no LibSensel DLL/.so/.a and
// works identically on Windows (COMx), macOS (/dev/cu.usbmodem*) and Linux /
// Raspberry Pi (/dev/ttyACM*).
//
// Protocol summary (verified byte-for-byte against firmware 0.19 build 298):
//   Register read  TX (3B): [0x81, reg, size]              (0x81 = board 0x01 | read-bit 0x80)
//                  RX     : ack(1)=PT_READ_ACK reg(1) size(2 LE) payload(size) checksum(1)
//   Register write TX     : [0x01, reg, size] payload checksum(1)
//                  RX     : ack(1)=PT_WRITE_ACK reg_echo(1)
//   Frame read (SYNC) TX  : [0x81, 0x26, 0x00]
//                  RX     : ack(1)=PT_RVS_ACK reg(1) header(1) size(2 LE) payload(size) checksum(1)
//   checksum = sum(bytes) & 0xFF

import (
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"go.bug.st/serial"
)

// Sensel register addresses (from sensel_register_map.h).
const (
	regMagic                = 0x00
	regFwVersionProtocol    = 0x06
	regDeviceSerialNumber   = 0x0F
	regSensorNumCols        = 0x10
	regSensorNumRows        = 0x12
	regSensorActiveWidthUM  = 0x14
	regSensorActiveHeightUM = 0x18
	regFrameContentControl  = 0x24
	regScanEnabled          = 0x25
	regScanReadFrame        = 0x26
	regFrameContentSupported = 0x28
	regContactsMaxCount     = 0x40
	regUnitShiftDims        = 0xA0
	regUnitShiftForce       = 0xA1
	regUnitShiftArea        = 0xA2
	regUnitShiftAngle       = 0xA3
	regLedBrightness        = 0x80
	regLedBrightnessSize    = 0x81
	regLedBrightnessMax     = 0x82
	regLedCount             = 0x84
	regDeviceOpen           = 0xD0
	regSoftReset            = 0xE0
)

// Protocol packet-type (ack) values and command bits (from sensel_register.h).
const (
	boardAddr = 0x01
	readFlag  = 0x80

	ptReadAck  = 1
	ptRvsAck   = 3
	ptWriteAck = 5
	ptWvsAck   = 7

	defaultVsHeaderSize = 4
	maxVsPacket         = 512
)

// Frame + contact content masks (from sensel.h).
const (
	frameContentContactsMask = 0x04

	contactMaskEllipse     = 0x01
	contactMaskDeltas      = 0x02
	contactMaskBoundingBox = 0x04
	contactMaskPeak        = 0x08

	contactDefaultSendSize     = 10
	contactEllipseSendSize     = 6
	contactDeltasSendSize      = 8
	contactBoundingBoxSendSize = 8
	contactPeakSendSize        = 6
)

// SenselScanMode enum value used to enable synchronous scanning.
const scanModeSync = 1

// senselContact holds one parsed contact in physical units (mm / grams).
type senselContact struct {
	ID    uint8
	State uint8
	X     float32 // mm
	Y     float32 // mm
	Force float32 // grams
	Area  float32 // sensor elements
}

// senselDevice is an open Morph on a serial port.
type senselDevice struct {
	port        serial.Port
	name        string
	readTimeout time.Duration

	serialNum string // device firmware serial (e.g. "SM01164910472"), from reg 0x0F

	// firmware info
	fwProtocol uint8
	fwMajor    uint8
	fwMinor    uint8
	fwBuild    uint16
	fwRelease  uint8
	deviceID   uint16
	deviceRev  uint8

	// sensor info / scaling
	maxContacts uint8
	numRows     uint16
	numCols     uint16
	widthMM     float32
	heightMM    float32
	dimsScale   float32
	forceScale  float32
	areaScale   float32
	angleScale  float32

	supportedContent uint8

	// LED info
	numLeds          uint8
	maxLedBrightness uint16
	ledRegSize       uint8
}

// openSensel opens the serial port, verifies the device is a Morph, soft-resets
// it, and reads all sensor/firmware info. It mirrors senselOpenDeviceByComPort.
func openSensel(name string) (*senselDevice, error) {
	mode := &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}
	port, err := serial.Open(name, mode)
	if err != nil {
		return nil, err
	}
	d := &senselDevice{port: port, name: name, readTimeout: 500 * time.Millisecond, ledRegSize: 1}
	if err := port.SetReadTimeout(d.readTimeout); err != nil {
		port.Close()
		return nil, err
	}
	// The Sensel SDK opens with DTR/RTS disabled.
	_ = port.SetDTR(false)
	_ = port.SetRTS(false)
	_ = port.ResetInputBuffer()

	// Confirm this really is a Morph before doing anything to it.
	magic, err := d.readReg(regMagic, 6)
	if err != nil {
		port.Close()
		return nil, fmt.Errorf("reading magic on %s: %w", name, err)
	}
	if string(magic) != "S3NS31" {
		port.Close()
		return nil, fmt.Errorf("%s is not a Sensel device (magic=%q)", name, magic)
	}

	// Read the firmware serial number (e.g. "SM01164910472"). This is the same
	// identifier LibSensel/palette report (and that morphs.json keys on), NOT the
	// USB descriptor iSerial. Best-effort: on failure the caller falls back.
	if s, err := d.readDeviceSerial(); err == nil {
		d.serialNum = s
	}

	// Soft reset to a known register state, then re-read everything.
	if err := d.writeReg(regSoftReset, []byte{1}); err != nil {
		port.Close()
		return nil, fmt.Errorf("soft reset on %s: %w", name, err)
	}
	time.Sleep(200 * time.Millisecond)
	_ = port.ResetInputBuffer()

	if err := d.initHandle(); err != nil {
		port.Close()
		return nil, fmt.Errorf("init %s: %w", name, err)
	}
	return d, nil
}

func (d *senselDevice) close() {
	if d.port != nil {
		d.port.Close()
	}
}

// writeAll writes the whole buffer, looping over short writes.
func (d *senselDevice) writeAll(buf []byte) error {
	for len(buf) > 0 {
		n, err := d.port.Write(buf)
		if err != nil {
			return err
		}
		buf = buf[n:]
	}
	return nil
}

// readFull reads exactly len(buf) bytes, erroring on a read timeout.
func (d *senselDevice) readFull(buf []byte) error {
	total := 0
	for total < len(buf) {
		n, err := d.port.Read(buf[total:])
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("serial read timeout (wanted %d bytes, got %d)", len(buf), total)
		}
		total += n
	}
	return nil
}

// readReg implements the fixed-size register read (_senselReadReg, SYNC mode).
func (d *senselDevice) readReg(reg byte, size int) ([]byte, error) {
	if err := d.writeAll([]byte{boardAddr | readFlag, reg, byte(size)}); err != nil {
		return nil, err
	}
	ack := make([]byte, 1)
	if err := d.readFull(ack); err != nil {
		return nil, err
	}
	if ack[0] != ptReadAck {
		return nil, fmt.Errorf("readReg 0x%02X: unexpected ack %d", reg, ack[0])
	}
	hdr := make([]byte, 3) // reg echo (1) + response size (2, LE)
	if err := d.readFull(hdr); err != nil {
		return nil, err
	}
	respSize := int(binary.LittleEndian.Uint16(hdr[1:3]))
	if respSize != size {
		return nil, fmt.Errorf("readReg 0x%02X: size mismatch (got %d, want %d)", reg, respSize, size)
	}
	buf := make([]byte, size)
	if err := d.readFull(buf); err != nil {
		return nil, err
	}
	ck := make([]byte, 1)
	if err := d.readFull(ck); err != nil {
		return nil, err
	}
	var sum byte
	for _, b := range buf {
		sum += b
	}
	if sum != ck[0] {
		return nil, fmt.Errorf("readReg 0x%02X: checksum mismatch (got 0x%02X, computed 0x%02X)", reg, ck[0], sum)
	}
	return buf, nil
}

// writeReg implements the fixed-size register write (_senselWriteReg).
func (d *senselDevice) writeReg(reg byte, data []byte) error {
	if err := d.writeAll([]byte{boardAddr, reg, byte(len(data))}); err != nil {
		return err
	}
	if err := d.writeAll(data); err != nil {
		return err
	}
	var sum byte
	for _, b := range data {
		sum += b
	}
	if err := d.writeAll([]byte{sum}); err != nil {
		return err
	}
	resp := make([]byte, 2) // ack + reg echo
	if err := d.readFull(resp); err != nil {
		return err
	}
	if resp[0] != ptWriteAck {
		return fmt.Errorf("writeReg 0x%02X: unexpected ack %d", reg, resp[0])
	}
	return nil
}

// writeRegVS implements the variable-size register write (_senselWriteRegVS),
// used for the LED brightness array.
func (d *senselDevice) writeRegVS(reg byte, data []byte) error {
	hdr := make([]byte, 9)
	hdr[0] = boardAddr
	hdr[1] = reg
	hdr[2] = 0 // size (unused for VS)
	hdr[3] = defaultVsHeaderSize
	binary.LittleEndian.PutUint32(hdr[4:8], uint32(len(data)))
	hdr[8] = hdr[4] + hdr[5] + hdr[6] + hdr[7] // checksum over the vs_size bytes
	if err := d.writeAll(hdr); err != nil {
		return err
	}
	resp := make([]byte, 2) // ack + reg echo
	if err := d.readFull(resp); err != nil {
		return err
	}
	for idx := 0; idx < len(data); {
		ps := len(data) - idx
		if ps > maxVsPacket {
			ps = maxVsPacket
		}
		var sum byte
		for i := 0; i < ps; i++ {
			sum += data[idx+i]
		}
		pkt := make([]byte, 0, 2+ps+1)
		pkt = append(pkt, byte(ps), byte(ps>>8))
		pkt = append(pkt, data[idx:idx+ps]...)
		pkt = append(pkt, sum)
		if err := d.writeAll(pkt); err != nil {
			return err
		}
		a := make([]byte, 1)
		if err := d.readFull(a); err != nil {
			return err
		}
		if a[0] != ptWvsAck {
			return fmt.Errorf("writeRegVS 0x%02X: unexpected packet ack %d", reg, a[0])
		}
		idx += ps
	}
	return nil
}

// readRegVS implements the variable-size register read (_senselReadRegVS).
// TX: [0x81, reg, 0]. RX: ack(3) size(2 LE) payload(size) checksum(1).
func (d *senselDevice) readRegVS(reg byte, maxSize int) ([]byte, error) {
	if err := d.writeAll([]byte{boardAddr | readFlag, reg, 0}); err != nil {
		return nil, err
	}
	ack := make([]byte, 3) // ack/reg/header — not individually validated by the SDK
	if err := d.readFull(ack); err != nil {
		return nil, err
	}
	szb := make([]byte, 2)
	if err := d.readFull(szb); err != nil {
		return nil, err
	}
	n := int(binary.LittleEndian.Uint16(szb))
	if n > maxSize {
		return nil, fmt.Errorf("readRegVS 0x%02X: reported size %d exceeds max %d", reg, n, maxSize)
	}
	buf := make([]byte, n)
	if err := d.readFull(buf); err != nil {
		return nil, err
	}
	ck := make([]byte, 1)
	if err := d.readFull(ck); err != nil {
		return nil, err
	}
	var sum byte
	for _, b := range buf {
		sum += b
	}
	if sum != ck[0] {
		return nil, fmt.Errorf("readRegVS 0x%02X: checksum mismatch (got 0x%02X, computed 0x%02X)", reg, ck[0], sum)
	}
	return buf, nil
}

// readDeviceSerial reads the firmware serial string from register 0x0F.
// The firmware pads the field with 0xFF (only ~13 chars valid), matching the
// SDK which converts 0xFF->0 and null-terminates.
func (d *senselDevice) readDeviceSerial() (string, error) {
	raw, err := d.readRegVS(regDeviceSerialNumber, 64)
	if err != nil {
		return "", err
	}
	out := make([]byte, 0, len(raw))
	for _, c := range raw {
		if c == 0x00 || c == 0xFF {
			break
		}
		out = append(out, c)
	}
	return string(out), nil
}

func scaleFromShift(b byte) float32 {
	if b > 30 {
		b = 0
	}
	return float32(uint32(1) << uint(b))
}

// initHandle reads firmware, sensor and LED info (mirrors _senselInitHandle).
func (d *senselDevice) initHandle() error {
	fw, err := d.readReg(regFwVersionProtocol, 9)
	if err != nil {
		return err
	}
	d.fwProtocol = fw[0]
	d.fwMajor = fw[1]
	d.fwMinor = fw[2]
	d.fwBuild = binary.LittleEndian.Uint16(fw[3:5])
	d.fwRelease = fw[5]
	d.deviceID = binary.LittleEndian.Uint16(fw[6:8])
	d.deviceRev = fw[8]

	if b, err := d.readReg(regFrameContentSupported, 1); err != nil {
		return err
	} else {
		d.supportedContent = b[0]
	}
	if b, err := d.readReg(regContactsMaxCount, 1); err != nil {
		return err
	} else {
		d.maxContacts = b[0]
	}
	if b, err := d.readReg(regSensorNumRows, 2); err != nil {
		return err
	} else {
		d.numRows = binary.LittleEndian.Uint16(b)
	}
	if b, err := d.readReg(regSensorNumCols, 2); err != nil {
		return err
	} else {
		d.numCols = binary.LittleEndian.Uint16(b)
	}

	if b, err := d.readReg(regUnitShiftDims, 1); err != nil {
		return err
	} else {
		d.dimsScale = scaleFromShift(b[0])
	}
	if b, err := d.readReg(regUnitShiftForce, 1); err != nil {
		return err
	} else {
		d.forceScale = scaleFromShift(b[0])
	}
	if b, err := d.readReg(regUnitShiftArea, 1); err != nil {
		return err
	} else {
		d.areaScale = scaleFromShift(b[0])
	}
	if b, err := d.readReg(regUnitShiftAngle, 1); err != nil {
		return err
	} else {
		d.angleScale = scaleFromShift(b[0])
	}

	if b, err := d.readReg(regSensorActiveWidthUM, 4); err != nil {
		return err
	} else {
		d.widthMM = float32(binary.LittleEndian.Uint32(b)) / 1000.0
	}
	if b, err := d.readReg(regSensorActiveHeightUM, 4); err != nil {
		return err
	} else {
		d.heightMM = float32(binary.LittleEndian.Uint32(b)) / 1000.0
	}

	// LED info is best-effort (a Morph may report zero LEDs).
	if b, err := d.readReg(regLedCount, 1); err == nil {
		d.numLeds = b[0]
	}
	if b, err := d.readReg(regLedBrightnessMax, 2); err == nil {
		d.maxLedBrightness = binary.LittleEndian.Uint16(b)
	}
	if b, err := d.readReg(regLedBrightnessSize, 1); err == nil && b[0] > 0 {
		d.ledRegSize = b[0]
	}
	return nil
}

// setupAndStart configures the device for contact scanning and starts it.
// Mirrors the SenselSetupAndStart used by gomorph.
func (d *senselDevice) setupAndStart() error {
	// Disable device timeouts (register 0xD0).
	if err := d.writeReg(regDeviceOpen, []byte{255}); err != nil {
		return fmt.Errorf("device open: %w", err)
	}
	// Report contacts only.
	if err := d.writeReg(regFrameContentControl, []byte{frameContentContactsMask}); err != nil {
		return fmt.Errorf("set frame content: %w", err)
	}
	// Start synchronous scanning.
	if err := d.writeReg(regScanEnabled, []byte{scanModeSync}); err != nil {
		return fmt.Errorf("start scanning: %w", err)
	}
	// Turn LEDs off (best-effort; a single VS write sets the whole array).
	d.turnOffLEDs()
	return nil
}

func (d *senselDevice) turnOffLEDs() {
	if d.numLeds == 0 {
		return
	}
	rs := int(d.ledRegSize)
	if rs < 1 {
		rs = 1
	}
	buf := make([]byte, int(d.numLeds)*rs) // all zero = off
	if err := d.writeRegVS(regLedBrightness, buf); err != nil {
		log.Printf("gomorph: turnOffLEDs on %s: %v", d.name, err)
	}
}

// readOneFrame requests and parses a single frame (SYNC, unbuffered mode).
// Returns the parsed contacts (physical units).
func (d *senselDevice) readOneFrame() ([]senselContact, error) {
	// Request a frame: read SCAN_READ_FRAME with size 0.
	if err := d.writeAll([]byte{boardAddr | readFlag, regScanReadFrame, 0}); err != nil {
		return nil, err
	}
	ack := make([]byte, 1)
	if err := d.readFull(ack); err != nil {
		return nil, err
	}
	if ack[0] != ptRvsAck {
		return nil, fmt.Errorf("readFrame: unexpected ack %d", ack[0])
	}
	// Frame header: reg(1) header(1) payload_size(2 LE).
	fh := make([]byte, 4)
	if err := d.readFull(fh); err != nil {
		return nil, err
	}
	payloadSize := int(binary.LittleEndian.Uint16(fh[2:4]))
	buf := make([]byte, payloadSize+1) // payload + checksum
	if err := d.readFull(buf); err != nil {
		return nil, err
	}
	data := buf[:payloadSize]
	var sum byte
	for _, b := range data {
		sum += b
	}
	if sum != buf[payloadSize] {
		return nil, fmt.Errorf("readFrame: checksum mismatch")
	}
	return d.parseFrame(data)
}

// parseFrame decodes one frame payload (mirrors _senselParseFrame +
// _senselParseContactFrame for contact content only).
func (d *senselDevice) parseFrame(data []byte) ([]senselContact, error) {
	// Frame header: content_bit_mask(1) rolling_counter(1) timestamp(4).
	if len(data) < 6 {
		return nil, fmt.Errorf("frame too short (%d bytes)", len(data))
	}
	content := data[0]
	off := 6

	if content&frameContentContactsMask == 0 {
		return nil, nil
	}
	if len(data) < off+2 {
		return nil, fmt.Errorf("contact frame header truncated")
	}
	contactMask := data[off]
	n := int(data[off+1])
	off += 2

	contacts := make([]senselContact, 0, n)
	for i := 0; i < n; i++ {
		if len(data) < off+contactDefaultSendSize {
			return nil, fmt.Errorf("contact %d truncated", i)
		}
		c := senselContact{
			ID:    data[off+0],
			State: data[off+1],
			X:     float32(binary.LittleEndian.Uint16(data[off+2:off+4])) / d.dimsScale,
			Y:     float32(binary.LittleEndian.Uint16(data[off+4:off+6])) / d.dimsScale,
			Force: float32(binary.LittleEndian.Uint16(data[off+6:off+8])) / d.forceScale,
			Area:  float32(binary.LittleEndian.Uint16(data[off+8:off+10])) / d.areaScale,
		}
		off += contactDefaultSendSize
		// Skip any optional per-contact fields the device included.
		if contactMask&contactMaskEllipse != 0 {
			off += contactEllipseSendSize
		}
		if contactMask&contactMaskDeltas != 0 {
			off += contactDeltasSendSize
		}
		if contactMask&contactMaskBoundingBox != 0 {
			off += contactBoundingBoxSendSize
		}
		if contactMask&contactMaskPeak != 0 {
			off += contactPeakSendSize
		}
		contacts = append(contacts, c)
	}
	return contacts, nil
}
