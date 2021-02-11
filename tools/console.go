package tools

import (
	"fmt"
	"image"

	"github.com/vizicist/palette/gui"
)

// Console is a window that has a couple of buttons
type Console struct {
	ctx         gui.WinContext
	clearButton gui.Window
	testButton  gui.Window
	threeButton gui.Window
	TextArea    gui.Window
}

// NewConsole xxx
func NewConsole(parent gui.Window) gui.ToolData {

	console := &Console{
		ctx: gui.NewWindowContext(parent),
	}

	console.clearButton = gui.AddChild(console, gui.NewButton(console, "Clear"))

	console.testButton = gui.AddChild(console, gui.NewButton(console, "Test"))

	console.threeButton = gui.AddChild(console, gui.NewButton(console, "Three"))

	console.TextArea = gui.AddChild(console, gui.NewScrollingText(console))

	gui.SetAttValue(console, "islogger", "true")

	return gui.NewToolData(console, "Console", image.Point{})
}

// Context xxx
func (console *Console) Context() *gui.WinContext {
	return &console.ctx
}

// Do xxx
func (console *Console) Do(cmd string, arg interface{}) (interface{}, error) {
	switch cmd {
	case "mouse":
		mouse := gui.ToMouse(arg)
		child, relPos := gui.WindowUnder(console, mouse.Pos)
		if child != nil {
			mouse.Pos = relPos
			child.Do("mouse", mouse)
		}
	case "resize":
		console.resize()
	case "redraw":
		console.redraw()
	case "restore":
		_, err := console.TextArea.Do("restore", arg)
		if err != nil {
			return nil, err
		}

	case "dumpstate":
		s, err := console.TextArea.Do("dumpstate", nil)
		if err != nil {
			return nil, err
		}
		return s, nil

	case "close":
		console.addLine("console: lose needs work? Maybe not\n")

	case "buttondown":
		// Clear is the only button
		b := gui.ToString(arg)
		switch b {
		case "Test":
			console.addLine("Test!\n")
		case "Three":
			console.addLine("Three!\n")
		case "Clear":
			console.TextArea.Do("clear", nil)
		default:
			console.addLine(fmt.Sprintf("Unknown button: %s\n", b))
		}
	case "buttonup":
		//
	case "addline":
		console.TextArea.Do(cmd, arg)
	default:
		gui.DoUpstream(console, cmd, arg)
	}
	return nil, nil
}

func (console *Console) addLine(s string) {
	console.TextArea.Do("addline", s)
}

// Resize xxx
func (console *Console) resize() {

	buttHeight := gui.WinStyle(console).TextHeight() + 12
	mySize := gui.WinCurrSize(console)

	// handle TextArea
	y0 := buttHeight + 4
	gui.WinSetChildPos(console, console.TextArea, image.Point{2, y0})

	areaSize := image.Point{mySize.X - 4, mySize.Y - y0 - 2}
	console.TextArea.Do("resize", areaSize)

	// If the TextArea has adjusted its size a bit, adjust our size as well
	currsz := gui.WinCurrSize(console.TextArea)
	mySize.Y += currsz.Y - areaSize.Y
	mySize.X += currsz.X - areaSize.X
	gui.WinSetMySize(console, mySize)

	buttWidth := mySize.X / 4
	buttSize := image.Point{buttWidth, buttHeight}

	// layout and resize all the buttons
	// XXX - this idiom should eventually be a layout utility
	pos := image.Point{2, 2}
	for _, w := range []gui.Window{console.clearButton, console.testButton, console.threeButton} {
		gui.WinSetChildPos(console, w, pos)
		gui.WinSetChildSize(w, buttSize)
		// Advance the horizontal position of the next button
		pos = pos.Add(image.Point{buttWidth, 0})
	}
}

// Draw xxx
func (console *Console) redraw() {
	size := gui.WinCurrSize(console)
	rect := image.Rect(0, 0, size.X, size.Y)
	gui.DoUpstream(console, "setcolor", gui.BackColor)
	gui.DoUpstream(console, "drawfilledrect", rect.Inset(1))
	gui.DoUpstream(console, "setcolor", gui.ForeColor)
	gui.DoUpstream(console, "drawrect", rect)
	gui.RedrawChildren(console)
}
