//go:build !darwin

package morph

// On Windows and Linux the serial-port enumerator is pure Go (SetupAPI /
// sysfs), so we use it to filter to Sensel's USB vendor ID. This also avoids
// opening non-Sensel serial devices during discovery.

import (
	"strings"

	"go.bug.st/serial/enumerator"
)

// Sensel USB vendor ID. The SDK matches any product ID under this VID.
const morphVID = "2C2F"

func listMorphPorts() ([]morphPort, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, err
	}
	var out []morphPort
	for _, p := range ports {
		if p == nil || !p.IsUSB {
			continue
		}
		if !strings.EqualFold(p.VID, morphVID) {
			continue
		}
		out = append(out, morphPort{Name: p.Name, SerialNumber: p.SerialNumber})
	}
	return out, nil
}
