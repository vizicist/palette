package window

import (
	"fmt"
	"image"

	"github.com/vizicist/palette/engine"
	"github.com/vizicist/palette/winsys"
)

func init() {
	winsys.RegisterWindow("Console", NewConsole)
	// engine.RegisterBlock("Console", NewConsole)
}

// Console is a window that has a couple of buttons
type Console struct {
	ctx         winsys.WinContext
	clearButton winsys.Window
	testButton  winsys.Window
	threeButton winsys.Window
	TextArea    winsys.Window
}

// NewConsole xxx
func NewConsole(parent winsys.Window) winsys.WindowData {

	console := &Console{
		ctx: winsys.NewWindowContext(parent),
	}

	console.clearButton = winsys.AddChild(console, winsys.NewButton(console, "Clear"))

	console.testButton = winsys.AddChild(console, winsys.NewButton(console, "Test"))

	console.threeButton = winsys.AddChild(console, winsys.NewButton(console, "Three"))

	console.TextArea = winsys.AddChild(console, winsys.NewScrollingText(console))

	winsys.SetAttValue(console, "islogger", "true")

	return winsys.NewToolData(console, "Console", image.Point{})
}

// Context xxx
func (console *Console) Context() *winsys.WinContext {
	return &console.ctx
}

// Do xxx
func (console *Console) Do(cmd engine.Cmd) string {
	switch cmd.Subj {
	case "mouse":
		pos := cmd.ValuesPos(engine.PointZero)
		child, relpos := winsys.WindowUnder(console, pos)
		if child != nil {
			// Note that we update the value in cmd.Values
			cmd.ValuesSetPos(relpos)
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
		winsys.DoUpstream(console, cmd)
	}
	return engine.OkResult()
}

func (console *Console) addLine(s string) {
	console.TextArea.Do(winsys.NewAddLineCmd(s))
}

// Resize xxx
func (console *Console) resize() {

	buttHeight := winsys.WinStyle(console).TextHeight() + 12
	mySize := winsys.WinCurrSize(console)

	// handle TextArea
	y0 := buttHeight + 4
	winsys.WinSetChildPos(console, console.TextArea, image.Point{2, y0})

	areaSize := image.Point{mySize.X - 4, mySize.Y - y0 - 2}
	console.TextArea.Do(winsys.NewResizeCmd(areaSize))

	// If the TextArea has adjusted its size a bit, adjust our size as well
	currsz := winsys.WinCurrSize(console.TextArea)
	mySize.Y += currsz.Y - areaSize.Y
	mySize.X += currsz.X - areaSize.X
	winsys.WinSetMySize(console, mySize)

	buttWidth := mySize.X / 4
	buttSize := image.Point{buttWidth, buttHeight}

	// layout and resize all the buttons
	// XXX - this idiom should eventually be a layout utility
	pos := image.Point{2, 2}
	for _, w := range []winsys.Window{console.clearButton, console.testButton, console.threeButton} {
		winsys.WinSetChildPos(console, w, pos)
		winsys.WinSetChildSize(w, buttSize)
		// Advance the horizontal position of the next button
		pos = pos.Add(image.Point{buttWidth, 0})
	}
}

// Draw xxx
func (console *Console) redraw() {
	size := winsys.WinCurrSize(console)
	rect := image.Rect(0, 0, size.X, size.Y)
	winsys.DoUpstream(console, winsys.NewSetColorCmd(winsys.BackColor))
	winsys.DoUpstream(console, winsys.NewDrawFilledRectCmd(rect.Inset(1)))
	winsys.DoUpstream(console, winsys.NewSetColorCmd(winsys.ForeColor))
	winsys.DoUpstream(console, winsys.NewDrawRectCmd(rect))
	winsys.RedrawChildren(console)
}