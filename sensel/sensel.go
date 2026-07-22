// Package sensel is a pure-Go driver for the Sensel Morph. It speaks the
// Morph's USB CDC-ACM serial register protocol directly, so it needs no
// LibSensel native library and no cgo. It builds and runs on Windows, macOS
// and Linux/Raspberry Pi.
//
// Protocol (verified byte-for-byte against firmware 0.19 build 298):
//
//	Register read  TX (3B): [0x81, reg, size]              (0x81 = board 0x01 | read-bit 0x80)
//	               RX     : ack(1)=PT_READ_ACK reg(1) size(2 LE) payload(size) checksum(1)
//	Register write TX     : [0x01, reg, size] payload checksum(1)
//	               RX     : ack(1)=PT_WRITE_ACK reg_echo(1)
//	Var-size read  TX (3B): [0x81, reg, 0x00]
//	               RX     : ack(3) size(2 LE) payload(size) checksum(1)
//	Frame read     TX (3B): [0x81, 0x26, 0x00]
//	               RX     : ack(1)=PT_RVS_ACK reg(1) header(1) size(2 LE) payload(size) checksum(1)
//	checksum = sum(bytes) & 0xFF
package sensel

import (
	"encoding/binary"
	"fmt"
	"time"

	"go.bug.st/serial"
)

// Contact state values (match SenselContactState in the Sensel SDK).
const (
	ContactInvalid = 0
	ContactStart   = 1 // finger down
	ContactMove    = 2 // finger drag
	ContactEnd     = 3 // finger up
)

// Scan detail levels (match SenselScanDetail in the Sensel SDK).
const (
	ScanDetailHigh   = 0
	ScanDetailMedium = 1
	ScanDetailLow    = 2
)

// Contact is one parsed touch in physical units.
type Contact struct {
	ID    uint8
	State uint8   // ContactStart / ContactMove / ContactEnd
	X     float32 // mm
	Y     float32 // mm
	Force float32 // grams
	Area  float32 // sensor elements
}

// Sensel register addresses (from sensel_register_map.h).
const (
	regMagic                 = 0x00
	regFwVersionProtocol     = 0x06
	regDeviceSerialNumber    = 0x0F
	regSensorNumCols         = 0x10
	regSensorNumRows         = 0x12
	regSensorActiveWidthUM   = 0x14
	regSensorActiveHeightUM  = 0x18
	regScanFrameRate         = 0x20
	regScanDetailControl     = 0x23
	regFrameContentControl   = 0x24
	regScanEnabled           = 0x25
	regScanReadFrame         = 0x26
	regFrameContentSupported = 0x28
	regContactsMaxCount      = 0x40
	regUnitShiftDims         = 0xA0
	regUnitShiftForce        = 0xA1
	regUnitShiftArea         = 0xA2
	regUnitShiftAngle        = 0xA3
	regLedBrightness         = 0x80
	regLedBrightnessSize     = 0x81
	regLedBrightnessMax      = 0x82
	regLedCount              = 0x84
	regDeviceOpen            = 0xD0
	regSoftReset             = 0xE0
)

// Protocol packet-type (ack) values and command bits.
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

// Frame + contact content masks.
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

const scanModeSync = 1 // SenselScanMode value to enable synchronous scanning

// Device is an open Morph on a serial port.
type Device struct {
	// Public device information (populated by Open).
	SerialNum string  // firmware serial (e.g. "SM01174213529"), from reg 0x0F
	Width     float32 // active-area width in mm
	Height    float32 // active-area height in mm
	FwMajor   uint8
	FwMinor   uint8
	FwBuild   uint16
	FwRelease uint8
	DeviceID  uint16

	port        serial.Port
	name        string
	readTimeout time.Duration

	fwProtocol  uint8
	deviceRev   uint8
	maxContacts uint8
	numRows     uint16
	numCols     uint16
	dimsScale   float32
	forceScale  float32
	areaScale   float32
	angleScale  float32

	supportedContent uint8

	numLeds          uint8
	maxLedBrightness uint16
	ledRegSize       uint8
}

// Open opens the serial port at portName (e.g. "COM7", "/dev/ttyACM0",
// "/dev/cu.usbmodemXXXX"), verifies the device is a Morph, reads its serial and
// sensor info, and soft-resets it to a known state. It does not start scanning;
// call the Set*/Start methods for that.
func Open(portName string) (*Device, error) {
	mode := &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}
	port, err := serial.Open(portName, mode)
	if err != nil {
		return nil, err
	}
	d := &Device{port: port, name: portName, readTimeout: 500 * time.Millisecond, ledRegSize: 1}
	if err := port.SetReadTimeout(d.readTimeout); err != nil {
		port.Close()
		return nil, err
	}
	// The Sensel SDK opens with DTR/RTS disabled.
	_ = port.SetDTR(false)
	_ = port.SetRTS(false)
	_ = port.ResetInputBuffer()

	magic, err := d.readReg(regMagic, 6)
	if err != nil {
		port.Close()
		return nil, fmt.Errorf("reading magic on %s: %w", portName, err)
	}
	if string(magic) != "S3NS31" {
		port.Close()
		return nil, fmt.Errorf("%s is not a Sensel device (magic=%q)", portName, magic)
	}

	// The firmware serial (reg 0x0F) is the identifier LibSensel/palette report
	// and that morphs.json keys on — NOT the USB descriptor iSerial.
	if s, err := d.readDeviceSerial(); err == nil {
		d.SerialNum = s
	}

	if err := d.writeReg(regSoftReset, []byte{1}); err != nil {
		port.Close()
		return nil, fmt.Errorf("soft reset on %s: %w", portName, err)
	}
	time.Sleep(200 * time.Millisecond)
	_ = port.ResetInputBuffer()

	if err := d.initHandle(); err != nil {
		port.Close()
		return nil, fmt.Errorf("init %s: %w", portName, err)
	}
	return d, nil
}

// Close closes the serial port.
func (d *Device) Close() {
	if d.port != nil {
		d.port.Close()
	}
}

// DisableTimeouts writes the "device open" register (0xD0 = 255), which
// disables the device's internal comm timeouts.
func (d *Device) DisableTimeouts() error {
	if err := d.writeReg(regDeviceOpen, []byte{255}); err != nil {
		return fmt.Errorf("device open: %w", err)
	}
	return nil
}

// SetFrameContentContacts configures the device to report contacts only.
func (d *Device) SetFrameContentContacts() error {
	if err := d.writeReg(regFrameContentControl, []byte{frameContentContactsMask}); err != nil {
		return fmt.Errorf("set frame content: %w", err)
	}
	return nil
}

// SetScanDetail sets the scan resolution (ScanDetailHigh/Medium/Low).
func (d *Device) SetScanDetail(detail byte) error {
	if err := d.writeReg(regScanDetailControl, []byte{detail}); err != nil {
		return fmt.Errorf("set scan detail: %w", err)
	}
	return nil
}

// SetMaxFrameRate caps the device's report rate (frames per second).
func (d *Device) SetMaxFrameRate(fps uint16) error {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, fps)
	if err := d.writeReg(regScanFrameRate, b); err != nil {
		return fmt.Errorf("set max frame rate: %w", err)
	}
	return nil
}

// StartScanning enables synchronous scanning. After this, use ReadFrame.
func (d *Device) StartScanning() error {
	if err := d.writeReg(regScanEnabled, []byte{scanModeSync}); err != nil {
		return fmt.Errorf("start scanning: %w", err)
	}
	return nil
}

// TurnOffLEDs sets every LED to zero brightness (best-effort; a Morph may
// report zero LEDs, in which case this is a no-op). Returns any error so the
// caller can decide whether to treat it as fatal.
func (d *Device) TurnOffLEDs() error {
	if d.numLeds == 0 {
		return nil
	}
	rs := int(d.ledRegSize)
	if rs < 1 {
		rs = 1
	}
	buf := make([]byte, int(d.numLeds)*rs) // all zero = off
	return d.writeRegVS(regLedBrightness, buf)
}

// ReadFrame requests and parses a single frame (synchronous, unbuffered mode),
// returning its contacts in physical units.
func (d *Device) ReadFrame() ([]Contact, error) {
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
	fh := make([]byte, 4) // reg(1) header(1) payload_size(2 LE)
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

// -------------------------------------------------------------------------
// Low-level serial + register access
// -------------------------------------------------------------------------

func (d *Device) writeAll(buf []byte) error {
	for len(buf) > 0 {
		n, err := d.port.Write(buf)
		if err != nil {
			return err
		}
		buf = buf[n:]
	}
	return nil
}

func (d *Device) readFull(buf []byte) error {
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

func (d *Device) readReg(reg byte, size int) ([]byte, error) {
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

func (d *Device) writeReg(reg byte, data []byte) error {
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

func (d *Device) writeRegVS(reg byte, data []byte) error {
	hdr := make([]byte, 9)
	hdr[0] = boardAddr
	hdr[1] = reg
	hdr[2] = 0
	hdr[3] = defaultVsHeaderSize
	binary.LittleEndian.PutUint32(hdr[4:8], uint32(len(data)))
	hdr[8] = hdr[4] + hdr[5] + hdr[6] + hdr[7]
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

func (d *Device) readRegVS(reg byte, maxSize int) ([]byte, error) {
	if err := d.writeAll([]byte{boardAddr | readFlag, reg, 0}); err != nil {
		return nil, err
	}
	ack := make([]byte, 3)
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
		return nil, fmt.Errorf("readRegVS 0x%02X: checksum mismatch", reg)
	}
	return buf, nil
}

func (d *Device) readDeviceSerial() (string, error) {
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

func (d *Device) initHandle() error {
	fw, err := d.readReg(regFwVersionProtocol, 9)
	if err != nil {
		return err
	}
	d.fwProtocol = fw[0]
	d.FwMajor = fw[1]
	d.FwMinor = fw[2]
	d.FwBuild = binary.LittleEndian.Uint16(fw[3:5])
	d.FwRelease = fw[5]
	d.DeviceID = binary.LittleEndian.Uint16(fw[6:8])
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
		d.Width = float32(binary.LittleEndian.Uint32(b)) / 1000.0
	}
	if b, err := d.readReg(regSensorActiveHeightUM, 4); err != nil {
		return err
	} else {
		d.Height = float32(binary.LittleEndian.Uint32(b)) / 1000.0
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

func (d *Device) parseFrame(data []byte) ([]Contact, error) {
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

	contacts := make([]Contact, 0, n)
	for i := 0; i < n; i++ {
		if len(data) < off+contactDefaultSendSize {
			return nil, fmt.Errorf("contact %d truncated", i)
		}
		c := Contact{
			ID:    data[off+0],
			State: data[off+1],
			X:     float32(binary.LittleEndian.Uint16(data[off+2:off+4])) / d.dimsScale,
			Y:     float32(binary.LittleEndian.Uint16(data[off+4:off+6])) / d.dimsScale,
			Force: float32(binary.LittleEndian.Uint16(data[off+6:off+8])) / d.forceScale,
			Area:  float32(binary.LittleEndian.Uint16(data[off+8:off+10])) / d.areaScale,
		}
		off += contactDefaultSendSize
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
