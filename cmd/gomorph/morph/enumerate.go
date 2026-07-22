package morph

// morphPort identifies a candidate Morph serial port. The platform-specific
// listMorphPorts implementations live in enumerate_default.go (Windows/Linux,
// which use the USB VID) and enumerate_darwin.go (macOS, which globs /dev to
// avoid IOKit/cgo). In every case openSensel confirms the device is really a
// Morph by reading its magic register and firmware serial.
type morphPort struct {
	Name         string // OS port name (COM7, /dev/ttyACM0, /dev/cu.usbmodem…)
	SerialNumber string // USB iSerial when known; empty on the macOS glob path
}
