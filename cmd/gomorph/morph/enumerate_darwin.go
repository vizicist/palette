//go:build darwin

package morph

// On macOS the only way to read USB VID/PID/serial for a serial port is via
// Apple's IOKit framework, which requires cgo (and thus a C toolchain, and
// breaks CGO_ENABLED=0 / cross-compilation). We avoid it entirely: Sensel
// Morphs enumerate as /dev/cu.usbmodem* CDC-ACM devices, so we glob those
// paths directly (pure Go) and let openSensel confirm each is a Morph by
// reading its magic register and firmware serial from register 0x0F.

import "path/filepath"

func listMorphPorts() ([]morphPort, error) {
	// Match the callout ("cu.") nodes; the callin ("tty.") nodes block on open.
	matches, err := filepath.Glob("/dev/cu.usbmodem*")
	if err != nil {
		return nil, err
	}
	out := make([]morphPort, 0, len(matches))
	for _, m := range matches {
		out = append(out, morphPort{Name: m})
	}
	return out, nil
}
