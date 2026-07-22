// Package morph is a pure-Go driver for the Sensel Morph. It speaks the Morph's
// USB CDC-ACM serial protocol directly (no LibSensel native library), so it
// builds and runs unchanged on Windows, macOS and Linux/Raspberry Pi.
//
// The public API mirrors github.com/vizicist/gomorph/morph so this is a
// drop-in, cgo-free replacement.
package morph

import (
	"fmt"
	"log"
	"sort"
	"time"
)

// DebugMorph enables verbose per-contact logging.
var DebugMorph bool

// MaxForce is the force value that maps to Z = 1.0. Matches gomorph.
var MaxForce float32 = 1500.0

// Contact state values (match SenselContactState in sensel.h).
const (
	CursorDown = 1 // CONTACT_START
	CursorDrag = 2 // CONTACT_MOVE
	CursorUp   = 3 // CONTACT_END
)

// CursorDeviceEvent is a single cursor event emitted by a Morph.
type CursorDeviceEvent struct {
	CID       string
	Timestamp int64 // milliseconds
	Ddu       string // "down", "drag", "up"
	X         float32
	Y         float32
	Z         float32
	Area      float32
}

// CursorDeviceCallbackFunc receives cursor events.
type CursorDeviceCallbackFunc func(e CursorDeviceEvent)

// OneMorph is a single opened Morph.
type OneMorph struct {
	Idx              uint8
	Opened           bool
	SerialNum        string
	Width            float32
	Height           float32
	FwVersionMajor   uint8
	FwVersionMinor   uint8
	FwVersionBuild   uint16
	FwVersionRelease uint8
	DeviceID         int

	dev *senselDevice
}

// Init enumerates Morphs, opens each one matching serialFilter ("*" for all),
// configures it for contact scanning and returns the opened Morphs.
func Init(serialFilter string) ([]OneMorph, error) {
	ports, err := listMorphPorts()
	if err != nil {
		return nil, fmt.Errorf("enumerating serial ports: %w", err)
	}
	// Stable ordering so idx values are deterministic across runs.
	sort.Slice(ports, func(i, j int) bool { return ports[i].Name < ports[j].Name })

	morphs := make([]OneMorph, 0, len(ports))
	idx := uint8(0)
	for _, p := range ports {
		dev, err := openSensel(p.Name)
		if err != nil {
			log.Printf("gomorph: could not open %s: %v", p.Name, err)
			continue
		}
		// Prefer the firmware serial (e.g. "SM01164910472"), matching palette /
		// morphs.json. Fall back to the USB iSerial, then the port name.
		serial := dev.serialNum
		if serial == "" {
			serial = p.SerialNumber
		}
		if serial == "" {
			serial = p.Name
		}
		// The -serial filter matches the firmware serial, so it can only be
		// applied after opening the device.
		if serialFilter != "*" && serial != serialFilter {
			if DebugMorph {
				log.Printf("gomorph: skipping serial=%s (port %s)", serial, p.Name)
			}
			dev.close()
			continue
		}
		if err := dev.setupAndStart(); err != nil {
			log.Printf("gomorph: setup/start on %s failed: %v", p.Name, err)
			dev.close()
			continue
		}
		morphs = append(morphs, newOneMorph(idx, serial, dev))
		idx++
	}
	if len(morphs) == 0 {
		return nil, fmt.Errorf("could not open any Morph matching serial=%s", serialFilter)
	}
	return morphs, nil
}

// InitPort opens a single Morph at an explicit device path (e.g. "COM7" or
// "/dev/cu.usbmodem1234"), bypassing enumeration. Useful as a fallback when
// automatic discovery misbehaves, or to target one specific device.
func InitPort(portName string) ([]OneMorph, error) {
	dev, err := openSensel(portName)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", portName, err)
	}
	if err := dev.setupAndStart(); err != nil {
		dev.close()
		return nil, fmt.Errorf("setup/start on %s: %w", portName, err)
	}
	serial := dev.serialNum
	if serial == "" {
		serial = portName
	}
	return []OneMorph{newOneMorph(0, serial, dev)}, nil
}

// newOneMorph builds a OneMorph from an opened device.
func newOneMorph(idx uint8, serial string, dev *senselDevice) OneMorph {
	return OneMorph{
		Idx:              idx,
		Opened:           true,
		SerialNum:        serial,
		Width:            dev.widthMM,
		Height:           dev.heightMM,
		FwVersionMajor:   dev.fwMajor,
		FwVersionMinor:   dev.fwMinor,
		FwVersionBuild:   dev.fwBuild,
		FwVersionRelease: dev.fwRelease,
		DeviceID:         int(dev.deviceID),
		dev:              dev,
	}
}

// Start loops forever, reading frames from all Morphs and invoking callback for
// each contact. It never returns unless the morphs slice is empty.
func Start(morphs []OneMorph, callback CursorDeviceCallbackFunc, forceFactor float32) error {
	if len(morphs) == 0 {
		return fmt.Errorf("morph.Start: morphs array is empty")
	}
	for {
		for i := range morphs {
			if err := morphs[i].readFrames(callback, forceFactor); err != nil {
				log.Printf("readFrames: err=%s", err)
			}
		}
		time.Sleep(time.Millisecond)
	}
}

func (m *OneMorph) readFrames(callback CursorDeviceCallbackFunc, forceFactor float32) error {
	contacts, err := m.dev.readOneFrame()
	if err != nil {
		return err
	}
	for _, c := range contacts {
		var xNorm, yNorm float32
		if m.Width > 0 {
			xNorm = c.X / m.Width
		}
		if m.Height > 0 {
			yNorm = c.Y / m.Height
		}
		zNorm := (c.Force / MaxForce) * forceFactor

		var ddu string
		switch c.State {
		case CursorDown:
			ddu = "down"
		case CursorDrag:
			ddu = "drag"
		case CursorUp:
			ddu = "up"
		default:
			// CONTACT_INVALID or unknown: ignore.
			continue
		}

		if DebugMorph {
			log.Printf("DebugMorph: serial=%s contact_id=%d idx=%d state=%d xNorm=%f yNorm=%f zNorm=%f",
				m.SerialNum, c.ID, m.Idx, c.State, xNorm, yNorm, zNorm)
		}

		// Match OpenGL / Freeframe coordinate space.
		yNorm = 1.0 - yNorm

		// Clamp to [0,1].
		if yNorm < 0.0 {
			yNorm = 0.0
		} else if yNorm > 1.0 {
			yNorm = 1.0
		}
		if xNorm < 0.0 {
			xNorm = 0.0
		} else if xNorm > 1.0 {
			xNorm = 1.0
		}

		callback(CursorDeviceEvent{
			CID:  fmt.Sprintf("%d", c.ID),
			Ddu:  ddu,
			X:    xNorm,
			Y:    yNorm,
			Z:    zNorm,
			Area: c.Area,
		})
	}
	return nil
}
