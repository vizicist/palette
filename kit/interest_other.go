//go:build !windows

package kit

import "fmt"

// CaptureMonitorLuma is only implemented on Windows, where Resolume runs.
// Elsewhere the interestingness evaluation reports the error and records
// no acceptance score.
func CaptureMonitorLuma(monitor int, w int, h int) ([]float64, error) {
	return nil, fmt.Errorf("interest capture not implemented on this platform")
}

func ListCaptureMonitors() (string, error) {
	return "", fmt.Errorf("interest capture not implemented on this platform")
}
