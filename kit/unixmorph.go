//go:build unix
// +build unix

package kit

type oneMorph struct {
	idx              uint8
	opened           bool
	serialNum        string
	width            float32
	height           float32
	fwVersionMajor   uint8
	fwVersionMinor   uint8
	fwVersionBuild   uint8
	fwVersionRelease uint8
	deviceID         int
	morphtype        string // "corners", "quadrants", "A", "B", "C", "D"
	currentTag       string // "A", "B", "C", "D" - it can change dynamically
	previousTag      string // "A", "B", "C", "D" - it can change dynamically
	contactIdToGid   map[int]int
}

var morphMaxForce float32 = 1000.0

var allMorphs []*oneMorph

// StartMorph xxx
func StartMorph(callback CursorCallbackFunc, forceFactor float32) {
	LogInfo("StartMorph (unix) called")
}

// CursorDown etc match values in sensel.h
const (
	CursorDown = 1
	CursorDrag = 2
	CursorUp   = 3
)

func (m *oneMorph) readFrames(callback CursorCallbackFunc, forceFactor float32) {
	LogInfo("readFrames called")
}

