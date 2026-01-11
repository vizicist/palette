//go:build unix

package kit

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
