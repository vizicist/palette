//go:build darwin

package sensel

// On macOS the only way to read USB VID/PID/serial for a serial port is via
// Apple's IOKit framework, which requires cgo (and breaks CGO_ENABLED=0 /
// cross-compilation). We avoid it entirely: Sensel Morphs enumerate as
// /dev/cu.usbmodem* CDC-ACM devices, so we glob those paths directly (pure Go)
// and let Open confirm each is a Morph via its magic register and serial.

import "path/filepath"

// ListPorts returns candidate Morph serial ports on macOS.
func ListPorts() ([]Port, error) {
	// Match the callout ("cu.") nodes; the callin ("tty.") nodes block on open.
	matches, err := filepath.Glob("/dev/cu.usbmodem*")
	if err != nil {
		return nil, err
	}
	out := make([]Port, 0, len(matches))
	for _, m := range matches {
		out = append(out, Port{Name: m})
	}
	return out, nil
}
