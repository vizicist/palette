// Package morph is a thin, gomorph-compatible wrapper around the pure-Go
// github.com/vizicist/palette/sensel driver. It enumerates Morphs, opens them,
// and emits normalized cursor events — matching the public API of the original
// github.com/vizicist/gomorph/morph package so this stays a drop-in, cgo-free
// replacement.
package morph

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/vizicist/palette/sensel"
)

// DebugMorph enables verbose per-contact logging.
var DebugMorph bool

// MaxForce is the force value that maps to Z = 1.0. Matches gomorph.
var MaxForce float32 = 1500.0

// Contact state values (kept for API compatibility with the original package).
const (
	CursorDown = sensel.ContactStart
	CursorDrag = sensel.ContactMove
	CursorUp   = sensel.ContactEnd
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

	dev *sensel.Device
}

// Init enumerates Morphs, opens each one matching serialFilter ("*" for all),
// configures it for contact scanning and returns the opened Morphs.
func Init(serialFilter string) ([]OneMorph, error) {
	ports, err := sensel.ListPorts()
	if err != nil {
		return nil, fmt.Errorf("enumerating serial ports: %w", err)
	}
	sort.Slice(ports, func(i, j int) bool { return ports[i].Name < ports[j].Name })

	morphs := make([]OneMorph, 0, len(ports))
	idx := uint8(0)
	for _, p := range ports {
		dev, err := sensel.Open(p.Name)
		if err != nil {
			log.Printf("gomorph: could not open %s: %v", p.Name, err)
			continue
		}
		serial := dev.SerialNum
		if serial == "" {
			serial = p.SerialNumber
		}
		if serial == "" {
			serial = p.Name
		}
		if serialFilter != "*" && serial != serialFilter {
			if DebugMorph {
				log.Printf("gomorph: skipping serial=%s (port %s)", serial, p.Name)
			}
			dev.Close()
			continue
		}
		if err := startDevice(dev); err != nil {
			log.Printf("gomorph: setup/start on %s failed: %v", p.Name, err)
			dev.Close()
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
// "/dev/cu.usbmodem1234"), bypassing enumeration.
func InitPort(portName string) ([]OneMorph, error) {
	dev, err := sensel.Open(portName)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", portName, err)
	}
	if err := startDevice(dev); err != nil {
		dev.Close()
		return nil, fmt.Errorf("setup/start on %s: %w", portName, err)
	}
	serial := dev.SerialNum
	if serial == "" {
		serial = portName
	}
	return []OneMorph{newOneMorph(0, serial, dev)}, nil
}

// startDevice configures a device for contact scanning (matching the original
// gomorph setup: contacts only, synchronous scanning, LEDs off).
func startDevice(dev *sensel.Device) error {
	if err := dev.DisableTimeouts(); err != nil {
		return err
	}
	if err := dev.SetFrameContentContacts(); err != nil {
		return err
	}
	if err := dev.StartScanning(); err != nil {
		return err
	}
	if err := dev.TurnOffLEDs(); err != nil {
		log.Printf("gomorph: turning off LEDs: %v", err) // non-fatal
	}
	return nil
}

func newOneMorph(idx uint8, serial string, dev *sensel.Device) OneMorph {
	return OneMorph{
		Idx:              idx,
		Opened:           true,
		SerialNum:        serial,
		Width:            dev.Width,
		Height:           dev.Height,
		FwVersionMajor:   dev.FwMajor,
		FwVersionMinor:   dev.FwMinor,
		FwVersionBuild:   dev.FwBuild,
		FwVersionRelease: dev.FwRelease,
		DeviceID:         int(dev.DeviceID),
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
	contacts, err := m.dev.ReadFrame()
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
			continue
		}

		if DebugMorph {
			log.Printf("DebugMorph: serial=%s contact_id=%d idx=%d state=%d xNorm=%f yNorm=%f zNorm=%f",
				m.SerialNum, c.ID, m.Idx, c.State, xNorm, yNorm, zNorm)
		}

		// Match OpenGL / Freeframe coordinate space.
		yNorm = 1.0 - yNorm
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
