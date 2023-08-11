package tool

import (
	"fmt"
	"image"

	w "github.com/vizicist/palette/twinsys"
)

func init() {
	w.RegisterWindow("Console", NewConsole)
	// engine.RegisterBlock("Console", NewConsole)
}

// Console is a window that has a couple of buttons
type Console struct {
	ctx         w.WinContext
	clearButton w.Window
	testButton  w.Window
	threeButton w.Window
	TextArea    w.Window
}

// NewConsole xxx
func NewConsole(parent w.Window) w.WindowData {

	console := &Console{
		ctx: w.NewWindowContext(parent),
	}

	console.clearButton = w.WinAddChild(console, w.NewButton(console, "Clear"))

	console.testButton = w.WinAddChild(console, w.NewButton(console, "Test"))

	console.threeButton = w.WinAddChild(console, w.NewButton(console, "Three"))

	console.TextArea = w.WinAddChild(console, w.NewScrollingText(console))

	w.WinSetAttValue(console, "islogger", "true")

	return w.NewToolData(console, "Console", image.Point{})
}

// Context xxx
func (console *Console) Context() *w.WinContext {
	return &console.ctx
}

// Do xxx
func (console *Console) Do(cmd engine.Cmd) string {
	switch cmd.Subj {
	case "mouse":
		pos := cmd.ValuesPos(engine.PointZero)
		child, relpos := w.WinFindWindowUnder(console, pos)
		if child != nil {
			// Note that we update the value in cmd.Values
			cmd.ValuesSetPos(relpos)
			engine.LogOfType("mouse", "Console Do mouse", "cmd", cmd)
			child.Do(cmd)
		}

	case "resize":
		console.resize()
	case "redraw":
		console.redraw()
	case "addline":
		console.TextArea.Do(cmd)
	case "restore":
		console.TextArea.Do(cmd)
	case "getstate":
		out := console.TextArea.Do(cmd)
		return out
	case "close":
		console.addLine("console: close needs work? Maybe not\n")
	case "buttonup":
		// do nothing
	case "buttondown":
		// Clear is the only button
		lbl := cmd.ValuesString("label", "")
		switch lbl {
		case "Test":
			console.addLine("Test!\n")
		case "Three":
			console.addLine("Three!\n")
		case "Clear":
			console.TextArea.Do(engine.NewSimpleCmd("clear"))
		default:
			lbl := cmd.ValuesString("label", "")
			console.addLine(fmt.Sprintf("Unknown button: %s\n", lbl))
		}

	default:
		w.WinDoUpstream(console, cmd)
	}
	return engine.OkResult()
}

func (console *Console) addLine(s string) {
	console.TextArea.Do(w.NewAddLineCmd(s))
}

// Resize xxx
func (console *Console) resize() {

	styleInfo := w.WinStyleInfo(console)
	buttHeight := styleInfo.TextHeight() + 12
	mySize := w.WinGetSize(console)

	// handle TextArea
	y0 := buttHeight + 4
	w.WinSetChildPos(console, console.TextArea, image.Point{2, y0})

	areaSize := image.Point{mySize.X - 4, mySize.Y - y0 - 2}
	console.TextArea.Do(w.NewResizeCmd(areaSize))

	// If the TextArea has adjusted its size a bit, adjust our size as well
	currsz := w.WinGetSize(console.TextArea)
	mySize.Y += currsz.Y - areaSize.Y
	mySize.X += currsz.X - areaSize.X
	w.WinSetSize(console, mySize)

	buttWidth := mySize.X / 4
	buttSize := image.Point{buttWidth, buttHeight}

	// layout and resize all the buttons
	// XXX - this idiom should eventually be a layout utility
	pos := image.Point{2, 2}
	for _, window := range []w.Window{console.clearButton, console.testButton, console.threeButton} {
		w.WinSetChildPos(console, window, pos)
		w.WinSetChildSize(window, buttSize)
		// Advance the horizontal position of the next button
		pos = pos.Add(image.Point{buttWidth, 0})
	}
}

// Draw xxx
func (console *Console) redraw() {
	size := w.WinGetSize(console)
	rect := image.Rect(0, 0, size.X, size.Y)
	w.WinDoUpstream(console, w.NewSetColorCmd(w.BackColor))
	w.WinDoUpstream(console, w.NewDrawFilledRectCmd(rect.Inset(1)))
	w.WinDoUpstream(console, w.NewSetColorCmd(w.ForeColor))
	w.WinDoUpstream(console, w.NewDrawRectCmd(rect))
	w.WinRedrawChildren(console)
}
