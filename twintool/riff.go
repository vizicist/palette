package tool

import (
	"fmt"
	"image"

	"github.com/vizicist/palette/kit"
	w "github.com/vizicist/palette/twinsys"
)

func init() {
	w.RegisterWindow("Riff", NewRiff)
}

// Riff is a window that has a couple of buttons
type Riff struct {
	ctx         w.WinContext
	clearButton w.Window
	TextArea    w.Window
}

// NewRiff xxx
func NewRiff(parent w.Window) w.WindowData {

	riff := &Riff{
		ctx: w.NewWindowContext(parent),
	}

	riff.clearButton = w.WinAddChild(riff, w.NewButton(riff, "Clear"))

	riff.TextArea = w.WinAddChild(riff, w.NewScrollingText(riff))

	w.WinSetAttValue(riff, "islogger", "true")

	return w.NewToolData(riff, "Riff", image.Point{})
}

// Context xxx
func (riff *Riff) Context() *w.WinContext {
	return &riff.ctx
}

// Do xxx
func (riff *Riff) Do(cmd kit.Cmd) string {

	// Things that should be handled the same for all tools?
	// "mouse" forwarding downstream
	// "draw*" forwarding upstream

	subj := cmd.Subj
	switch subj {
	case "mouse":
		w.WinForwardMouse(riff, cmd)
	case "resize":
		riff.resize()
	case "redraw":
		riff.redraw()
	case "restore":
		riff.TextArea.Do(cmd)
	case "close":
		// do anything?
	case "getstate":
		ret := riff.TextArea.Do(kit.NewSimpleCmd("getstate"))
		return ret

	case "buttondown":
		// Clear is the only button
		lbl := cmd.ValuesString("label", "")
		switch lbl {
		case "Clear":
			riff.TextArea.Do(kit.NewSimpleCmd("clear"))
		default:
			riff.addLine(fmt.Sprintf("Unknown button: %s\n", lbl))
		}
	case "buttonup":
		// do nothing
	case "addline":
		riff.TextArea.Do(cmd)
	default:
		w.WinDoUpstream(riff, cmd)
	}
	return kit.OkResult()
}

func (riff *Riff) addLine(line string) {
	riff.TextArea.Do(w.NewAddLineCmd(line))
}

// Resize xxx
func (riff *Riff) resize() {

	buttHeight := w.WinStyleInfo(riff).TextHeight() + 12
	mySize := w.WinGetSize(riff)

	// handle TextArea
	y0 := buttHeight + 4
	w.WinSetChildPos(riff, riff.TextArea, image.Point{2, y0})

	areaSize := image.Point{mySize.X - 4, mySize.Y - y0 - 2}
	riff.TextArea.Do(w.NewResizeCmd(areaSize))

	// If the TextArea has adjusted its size a bit, adjust our size as well
	currsz := w.WinGetSize(riff.TextArea)
	mySize.Y += currsz.Y - areaSize.Y
	mySize.X += currsz.X - areaSize.X
	w.WinSetSize(riff, mySize)

	buttWidth := mySize.X / 4
	buttSize := image.Point{buttWidth, buttHeight}

	// layout and resize all the buttons
	// XXX - this idiom should eventually be a layout utility
	pos := image.Point{2, 2}
	for _, window := range []w.Window{riff.clearButton} {
		w.WinSetChildPos(riff, window, pos)
		w.WinSetChildSize(window, buttSize)
		// Advance the horizontal position for the next button
		pos = pos.Add(image.Point{buttWidth, 0})
	}
}

// Draw xxx
func (riff *Riff) redraw() {
	size := w.WinGetSize(riff)
	rect := image.Rect(0, 0, size.X, size.Y)
	w.WinDoUpstream(riff, w.NewSetColorCmd(w.BackColor))
	w.WinDoUpstream(riff, w.NewDrawFilledRectCmd(rect.Inset(1)))
	w.WinDoUpstream(riff, w.NewSetColorCmd(w.ForeColor))
	w.WinDoUpstream(riff, w.NewDrawRectCmd(rect))
	w.WinRedrawChildren(riff)
}
