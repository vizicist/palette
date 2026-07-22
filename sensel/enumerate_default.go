//go:build !darwin

package sensel

// On Windows and Linux the serial-port enumerator is pure Go (SetupAPI /
// sysfs), so we use it to filter to Sensel's USB vendor ID. This also avoids
// opening non-Sensel serial devices during discovery.

import (
	"strings"

	"go.bug.st/serial/enumerator"
)

// morphVID is Sensel's USB vendor ID. The SDK matches any product ID under it.
const morphVID = "2C2F"

// ListPorts returns all serial ports that belong to a Sensel device.
func ListPorts() ([]Port, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, err
	}
	var out []Port
	for _, p := range ports {
		if p == nil || !p.IsUSB {
			continue
		}
		if !strings.EqualFold(p.VID, morphVID) {
			continue
		}
		out = append(out, Port{Name: p.Name, SerialNumber: p.SerialNumber})
	}
	return out, nil
}
