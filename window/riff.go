package window

import (
	"fmt"
	"image"

	"github.com/vizicist/palette/engine"
	"github.com/vizicist/palette/winsys"
)

func init() {
	winsys.RegisterWindow("Riff", NewRiff)
}

// Riff is a window that has a couple of buttons
type Riff struct {
	ctx         winsys.WinContext
	clearButton winsys.Window
	TextArea    winsys.Window
}

// NewRiff xxx
func NewRiff(parent winsys.Window) winsys.WindowData {

	riff := &Riff{
		ctx: winsys.NewWindowContext(parent),
	}

	riff.clearButton = winsys.WinAddChild(riff, winsys.NewButton(riff, "Clear"))

	riff.TextArea = winsys.WinAddChild(riff, winsys.NewScrollingText(riff))

	winsys.WinSetAttValue(riff, "islogger", "true")

	// engine.TheRouter().SendCursor3DDeviceEventsTo(riff.handleCursor3DDeviceInput)

	return winsys.NewToolData(riff, "Riff", image.Point{})
}

// Context xxx
func (riff *Riff) Context() *winsys.WinContext {
	return &riff.ctx
}

// Do xxx
func (riff *Riff) Do(cmd engine.Cmd) string {

	subj := cmd.Subj
	switch subj {
	case "mouse":
		winsys.WinForwardMouse(riff, cmd)
	case "resize":
		riff.resize()
	case "redraw":
		riff.redraw()
	case "restore":
		riff.TextArea.Do(cmd)
	case "clos":
		// do anything?
	case "getstate":
		ret := riff.TextArea.Do(engine.NewSimpleCmd("getstate"))
		return ret

	case "buttondown":
		// Clear is the only button
		lbl := cmd.ValuesString("label", "")
		switch lbl {
		case "Clear":
			riff.TextArea.Do(engine.NewSimpleCmd("clear"))
		default:
			riff.addLine(fmt.Sprintf("Unknown button: %s\n", lbl))
		}
	case "buttonup":
		// do nothing
	case "addline":
		riff.TextArea.Do(cmd)
	default:
		winsys.WinDoUpstream(riff, cmd)
	}
	return engine.OkResult()
}

func (riff *Riff) addLine(line string) {
	riff.TextArea.Do(winsys.NewAddLineCmd(line))
}

// Resize xxx
func (riff *Riff) resize() {

	buttHeight := winsys.WinStyleInfo(riff).TextHeight() + 12
	mySize := winsys.WinGetSize(riff)

	// handle TextArea
	y0 := buttHeight + 4
	winsys.WinSetChildPos(riff, riff.TextArea, image.Point{2, y0})

	areaSize := image.Point{mySize.X - 4, mySize.Y - y0 - 2}
	riff.TextArea.Do(winsys.NewResizeCmd(areaSize))

	// If the TextArea has adjusted its size a bit, adjust our size as well
	currsz := winsys.WinGetSize(riff.TextArea)
	mySize.Y += currsz.Y - areaSize.Y
	mySize.X += currsz.X - areaSize.X
	winsys.WinSetSize(riff, mySize)

	buttWidth := mySize.X / 4
	buttSize := image.Point{buttWidth, buttHeight}

	// layout and resize all the buttons
	// XXX - this idiom should eventually be a layout utility
	pos := image.Point{2, 2}
	for _, w := range []winsys.Window{riff.clearButton} {
		winsys.WinSetChildPos(riff, w, pos)
		winsys.WinSetChildSize(w, buttSize)
		// Advance the horizontal position for the next button
		pos = pos.Add(image.Point{buttWidth, 0})
	}
}

// Draw xxx
func (riff *Riff) redraw() {
	size := winsys.WinGetSize(riff)
	rect := image.Rect(0, 0, size.X, size.Y)
	winsys.WinDoUpstream(riff, winsys.NewSetColorCmd(winsys.BackColor))
	winsys.WinDoUpstream(riff, winsys.NewDrawFilledRectCmd(rect.Inset(1)))
	winsys.WinDoUpstream(riff, winsys.NewSetColorCmd(winsys.ForeColor))
	winsys.WinDoUpstream(riff, winsys.NewDrawRectCmd(rect))
	winsys.WinRedrawChildren(riff)
}
