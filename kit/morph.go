package kit

// Pure-Go Sensel Morph input, cross-platform (Windows/macOS/Linux/Raspberry Pi).
// This replaces the previous cgo/LibSensel implementation with the pure-Go
// github.com/vizicist/palette/sensel driver, so palette needs no LibSensel DLL
// and runs on the Raspberry Pi with no native library.

import (
	"fmt"
	"runtime/debug"
	"sort"
	"time"

	"github.com/vizicist/palette/sensel"
)

// CursorDown etc match the Sensel contact-state values.
const (
	CursorDown = 1
	CursorDrag = 2
	CursorUp   = 3
)

var morphMaxForce float64 = 1000.0

var allMorphs []*oneMorph

type oneMorph struct {
	idx              uint8
	opened           bool
	serialNum        string
	width            float64
	height           float64
	fwVersionMajor   uint8
	fwVersionMinor   uint8
	fwVersionBuild   uint16
	fwVersionRelease uint8
	deviceID         int
	morphtype        string // "corners", "quadrants", "A", "B", "C", "D"
	currentTag       string // "A", "B", "C", "D" - it can change dynamically
	previousTag      string // "A", "B", "C", "D" - it can change dynamically
	contactIDToGID   map[int]int
	pressureStats    morphPressureStats
	dev              *sensel.Device
}

type morphPressureStats struct {
	seen     bool
	samples  int
	forceMin float64
	forceMax float64
	zMin     float64
	zMax     float64
}

func (m *oneMorph) updatePressureStats(force, z float64) morphPressureStats {
	stats := m.pressureStats
	if !stats.seen {
		stats.seen = true
		stats.forceMin = force
		stats.forceMax = force
		stats.zMin = z
		stats.zMax = z
	} else {
		if force < stats.forceMin {
			stats.forceMin = force
		}
		if force > stats.forceMax {
			stats.forceMax = force
		}
		if z < stats.zMin {
			stats.zMin = z
		}
		if z > stats.zMax {
			stats.zMax = z
		}
	}
	stats.samples++
	m.pressureStats = stats
	return stats
}

// morphRetryInterval is how long to wait before looking for Morphs again after
// there are none left to read from. Long enough that a machine with no Morph
// attached isn't enumerating serial ports constantly, short enough that
// replugging one during a show brings the pad back while someone is still
// standing at it.
const morphRetryInterval = 3 * time.Second

// StartMorph opens all connected Morphs and streams their contacts as cursor
// events until the process exits.
//
// It keeps going rather than giving up. This runs unattended at venues, where
// the failure it used to have - one USB hiccup, or one panic, and the pads were
// dead until somebody noticed and restarted the engine - costs the whole
// installation. A Morph that stops reading is closed and re-opened on the next
// pass, and Morphs plugged in after startup are picked up the same way.
func StartMorph(callback CursorCallbackFunc, forceFactor float64) {
	for {
		// Backing off only when there was nothing to read keeps a real drop-out
		// short: on a four-pad installation, one Morph hiccupping re-opens all
		// four within a frame or two rather than leaving every pad dead for the
		// retry interval. A machine with no Morph attached backs off instead.
		if !readMorphsUntilOneDrops(callback, forceFactor) {
			time.Sleep(morphRetryInterval)
		}
	}
}

// readMorphsUntilOneDrops opens whatever Morphs are attached and reads them
// until one stops responding, then returns so its caller can start over with a
// fresh enumeration - which is what picks up a replugged device, since it comes
// back as a new port. Reports whether any Morph was opened at all. A panic in
// here is logged and ends this pass only.
func readMorphsUntilOneDrops(callback CursorCallbackFunc, forceFactor float64) (opened bool) {

	defer closeAllMorphs()

	defer func() {
		if r := recover(); r != nil {
			// Print stack trace in the error messages
			stacktrace := string(debug.Stack())
			// First to stdout, then to log file
			fmt.Printf("PANIC: recover in StartMorph called, r=%+v stack=%v", r, stacktrace)
			err := fmt.Errorf("PANIC: recover in StartMorph has been called")
			LogError(err, "r", r, "stack", stacktrace)
		}
	}()

	err := morphInitialize()
	if err != nil {
		LogIfError(err)
		return
	}
	if len(allMorphs) == 0 {
		// Logged only when it changes, since this is now retried forever and a
		// machine with no Morph attached would otherwise say so every few
		// seconds for as long as it runs.
		if !noMorphsReported {
			LogInfo("No Morphs were found, will keep looking", "every", morphRetryInterval)
			noMorphsReported = true
		}
		return false
	}
	noMorphsReported = false

	for {
		for _, m := range allMorphs {
			m.readFrames(callback, forceFactor)
			if !m.opened {
				return true // readFrames closed it; re-enumerate everything
			}
		}
		time.Sleep(time.Millisecond)
	}
}

// noMorphsReported keeps the "no Morphs" message down to one per dry spell.
var noMorphsReported bool

// closeAllMorphs releases the devices so the next morphInitialize can re-open
// them. Without this a re-enumeration would find ports it already holds.
func closeAllMorphs() {
	for _, m := range allMorphs {
		if m.opened {
			m.opened = false
			m.dev.Close()
		}
	}
	allMorphs = nil
}

func (m *oneMorph) readFrames(callback CursorCallbackFunc, forceFactor float64) {
	contacts, err := m.dev.ReadFrame()
	if err != nil {
		LogWarn("Morph stopped reading, will look for it again", "serialnum", m.serialNum, "err", err)
		m.opened = false
		m.dev.Close()
		return
	}
	for n, contact := range contacts {
		xNorm := float64(contact.X) / m.width
		yNorm := float64(contact.Y) / m.height
		rawForce := float64(contact.Force)
		zNorm := rawForce / morphMaxForce
		zNorm *= forceFactor
		area := float64(contact.Area)
		pressureStats := m.updatePressureStats(rawForce, zNorm)
		var ddu string
		switch contact.State {
		case CursorDown:
			ddu = "down"
		case CursorDrag:
			ddu = "drag"
		case CursorUp:
			ddu = "up"
		default:
			LogWarn("Invalid value", "state", contact.State)
			continue
		}

		switch m.morphtype {

		case "corners":

			// If the position is in one of the corners,
			// we change the source to that corner.
			edge := 0.075
			cornerSource := ""
			if xNorm < edge && yNorm < edge {
				cornerSource = "A"
			} else if xNorm < edge && yNorm > (1.0-edge) {
				cornerSource = "B"
			} else if xNorm > (1.0-edge) && yNorm > (1.0-edge) {
				cornerSource = "C"
			} else if xNorm > (1.0-edge) && yNorm < edge {
				cornerSource = "D"
			}
			if cornerSource != m.currentTag {
				LogInfo("Switching corners pad", "source", cornerSource)
				ce := NewCursorClearEvent()
				callback(ce)
				m.currentTag = cornerSource
				continue // loop, doesn't send a cursor event
			}

		case "quadrants":

			// This method splits a single pad into quadrants.
			// Adjust the xNorm and yNorm values to provide
			// full range 0-1 within each quadrant.
			quadSource := ""
			switch {
			case xNorm < 0.5 && yNorm < 0.5:
				quadSource = "A"
			case xNorm < 0.5 && yNorm >= 0.5:
				quadSource = "B"
				yNorm = yNorm - 0.5
			case xNorm >= 0.5 && yNorm >= 0.5:
				quadSource = "C"
				xNorm = xNorm - 0.5
				yNorm = yNorm - 0.5
			case xNorm >= 0.5 && yNorm < 0.5:
				quadSource = "D"
				xNorm = xNorm - 0.5
			default:
				LogWarn("unable to find QUAD source", "x", xNorm, "y", yNorm)
				continue
			}
			xNorm *= 2.0
			yNorm *= 2.0
			m.currentTag = quadSource

		case "A", "B", "C", "D":
			m.currentTag = m.morphtype

		default:
			LogWarn("Unknown morphtype", "morphtype", m.morphtype)
			m.currentTag = ""
		}

		if m.currentTag == "" {
			LogWarn("Hey! currentTag not set, assuming A")
			m.currentTag = "A"
		}

		contactid := int(contact.ID)
		gid, ok := m.contactIDToGID[contactid]
		if !ok {
			// If we've never seen this contact before, create a new cid...
			gid = theCursorManager.UniqueGID()
			m.contactIDToGID[contactid] = gid
		} else if m.currentTag != m.previousTag {
			// If we're switching to a new source, clear existing cursors...
			ce := NewCursorClearEvent()
			callback(ce)
			// and create a new cid...
			gid = theCursorManager.UniqueGID()
		}

		m.previousTag = m.currentTag

		LogOfType("morph,pressure", "Morph",
			"idx", m.idx,
			"contactid", contactid,
			"gid", gid,
			"n", n,
			"contactstate", contact.State,
			"xNorm", xNorm,
			"yNorm", yNorm,
			"zNorm", zNorm,
			"totalForce", rawForce,
			"area", area,
			"morphMaxForce", morphMaxForce,
			"forceFactor", forceFactor,
			"forceMin", pressureStats.forceMin,
			"forceMax", pressureStats.forceMax,
			"zMin", pressureStats.zMin,
			"zMax", pressureStats.zMax,
			"pressureSamples", pressureStats.samples)

		// make the coordinate space match OpenGL and Freeframe
		yNorm = 1.0 - yNorm

		// Make sure we don't send anyting out of bounds
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

		pos := CursorPos{xNorm, yNorm, zNorm}
		ce := NewCursorEvent(gid, m.currentTag, ddu, pos)
		ce.Area = area
		callback(ce)

		// A contact that has lifted must not leave its GID behind. Sensel
		// contact IDs are slot indices the firmware reuses right away, so the
		// next finger down usually gets the same one - and with looping on,
		// DeleteActiveCursorIfZLessThan deliberately keeps the ActiveCursor
		// alive past the up when maxZ reached the loop threshold. A stale entry
		// therefore hands the next touch the finished gesture's GID, and
		// ExecuteCursorEvent takes its "down" as a continuation of that still
		// looping cursor instead of a new gesture: it inherits the old patch
		// and maxZ, and the older gesture's loop cleanup later deletes the
		// newer one's events. maxZ never decreases, so that slot stayed
		// poisoned for the rest of a run - days, on an installation.
		if ddu == "up" {
			delete(m.contactIDToGID, contactid)
		}
	}
}

// setupMorph configures a freshly opened device for contact scanning, matching
// the previous LibSensel-based setup (contacts only, medium scan detail,
// 250 fps cap, synchronous scanning, LEDs off).
func setupMorph(dev *sensel.Device) error {
	if err := dev.DisableTimeouts(); err != nil {
		return err
	}
	if err := dev.SetFrameContentContacts(); err != nil {
		return err
	}
	if err := dev.SetScanDetail(sensel.ScanDetailMedium); err != nil {
		return err
	}
	if err := dev.SetMaxFrameRate(250); err != nil {
		return err
	}
	if err := dev.StartScanning(); err != nil {
		return err
	}
	if err := dev.TurnOffLEDs(); err != nil {
		LogWarn("Morph: turning off LEDs", "err", err) // non-fatal
	}
	return nil
}

func morphInitialize() error {

	ports, err := sensel.ListPorts()
	if err != nil {
		return fmt.Errorf("morphInitialize: listing ports: %w", err)
	}
	// Stable ordering so idx values are deterministic across runs.
	sort.Slice(ports, func(i, j int) bool { return ports[i].Name < ports[j].Name })

	allMorphs = make([]*oneMorph, 0, len(ports))

	defaultMorphName := []string{"A", "B", "C", "D"}
	nextMorphIndex := 0
	idx := uint8(0)

	for _, p := range ports {

		dev, err := sensel.Open(p.Name)
		if err != nil {
			// Skip this port and keep the other Morphs working.
			LogWarn("Unable to open Morph", "port", p.Name, "err", err)
			continue
		}

		if err := setupMorph(dev); err != nil {
			LogWarn("Unable to set up Morph", "port", p.Name, "err", err)
			dev.Close()
			continue
		}

		m := &oneMorph{
			contactIDToGID: map[int]int{},
			dev:            dev,
		}
		m.idx = idx
		m.serialNum = dev.SerialNum
		m.opened = true
		m.width = float64(dev.Width)
		m.height = float64(dev.Height)
		m.fwVersionMajor = dev.FwMajor
		m.fwVersionMinor = dev.FwMinor
		m.fwVersionBuild = dev.FwBuild
		m.fwVersionRelease = dev.FwRelease
		m.deviceID = int(dev.DeviceID)

		morphtype, ok := MorphDefs[m.serialNum]
		if !ok {
			// It's not explicitly present in morphs.json
			morphtype = defaultMorphName[nextMorphIndex]
			if nextMorphIndex < len(defaultMorphName)-1 {
				nextMorphIndex++
			}
			LogInfo("Morph serial# isn't in morphs.json", "serialnum", m.serialNum, "morphtype", morphtype)
		}

		m.morphtype = morphtype
		switch m.morphtype {
		case "corners", "quadrants":
			m.currentTag = "A"
		case "A", "B", "C", "D":
			m.currentTag = morphtype
		default:
			LogWarn("Unexpected morphtype", "morphtype", morphtype)
		}

		// Don't use Debug.Morph, this should always gets logged
		firmware := fmt.Sprintf("%d.%d.%d", m.fwVersionMajor, m.fwVersionMinor, m.fwVersionBuild)
		LogInfo("Morph Opened and Started", "idx", m.idx, "serial", m.serialNum, "firmware", firmware, "morphtype", m.morphtype)

		allMorphs = append(allMorphs, m)
		idx++
	}
	return nil
}
