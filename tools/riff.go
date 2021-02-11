package tools

import (
	"fmt"
	"image"

	"github.com/vizicist/palette/engine"
	"github.com/vizicist/palette/gui"
)

// Riff is a window that has a couple of buttons
type Riff struct {
	ctx         gui.WinContext
	clearButton gui.Window
	testButton  gui.Window
	threeButton gui.Window
	TextArea    gui.Window
}

// NewRiff xxx
func NewRiff(parent gui.Window) gui.ToolData {

	riff := &Riff{
		ctx: gui.NewWindowContext(parent),
	}

	riff.clearButton = gui.AddChild(riff, gui.NewButton(riff, "Clear"))

	riff.testButton = gui.AddChild(riff, gui.NewButton(riff, "Test"))

	riff.threeButton = gui.AddChild(riff, gui.NewButton(riff, "Three"))

	riff.TextArea = gui.AddChild(riff, gui.NewScrollingText(riff))

	gui.SetAttValue(riff, "islogger", "true")

	engine.TheRouter().AddGestureDeviceCallback(riff.handleGestureDeviceInput)

	return gui.NewToolData(riff, "Riff", image.Point{})
}

func (riff *Riff) handleGestureDeviceInput(e engine.GestureDeviceEvent) {
	riff.addLine(fmt.Sprintf("GestureInput: e=%v\n", e))
}

// Context xxx
func (riff *Riff) Context() *gui.WinContext {
	return &riff.ctx
}

// Do xxx
func (riff *Riff) Do(cmd string, arg interface{}) (interface{}, error) {

	switch cmd {
	case "mouse":
		gui.WinForwardMouse(riff, gui.ToMouse(arg))
	case "resize":
		riff.resize()
	case "redraw":
		riff.redraw()
	case "restore":
		_, err := riff.TextArea.Do("restore", arg)
		if err != nil {
			return nil, err
		}

	case "dumpstate":
		s, err := riff.TextArea.Do("dumpstate", nil)
		if err != nil {
			return nil, err
		}
		return s, nil

	case "close":
		// do nothing?

	case "buttondown":
		// Clear is the only button
		b := gui.ToString(arg)
		switch b {
		case "Test":
			riff.addLine("Test!\n")
		case "Three":
			riff.addLine("Three!\n")
		case "Clear":
			riff.TextArea.Do("clear", nil)
		default:
			riff.addLine(fmt.Sprintf("Unknown button: %s\n", b))
		}
	case "buttonup":
		//
	case "addline":
		riff.TextArea.Do(cmd, arg)
	default:
		gui.DoUpstream(riff, cmd, arg)
	}
	return nil, nil
}

func (riff *Riff) addLine(s string) {
	riff.TextArea.Do("addline", s)
}

// Resize xxx
func (riff *Riff) resize() {

	buttHeight := gui.WinStyle(riff).TextHeight() + 12
	mySize := gui.WinCurrSize(riff)

	// handle TextArea
	y0 := buttHeight + 4
	gui.WinSetChildPos(riff, riff.TextArea, image.Point{2, y0})

	areaSize := image.Point{mySize.X - 4, mySize.Y - y0 - 2}
	riff.TextArea.Do("resize", areaSize)

	// If the TextArea has adjusted its size a bit, adjust our size as well
	currsz := gui.WinCurrSize(riff.TextArea)
	mySize.Y += currsz.Y - areaSize.Y
	mySize.X += currsz.X - areaSize.X
	gui.WinSetMySize(riff, mySize)

	buttWidth := mySize.X / 4
	buttSize := image.Point{buttWidth, buttHeight}

	// layout and resize all the buttons
	// XXX - this idiom should eventually be a layout utility
	pos := image.Point{2, 2}
	for _, w := range []gui.Window{riff.clearButton, riff.testButton, riff.threeButton} {
		gui.WinSetChildPos(riff, w, pos)
		gui.WinSetChildSize(w, buttSize)
		// Advance the horizontal position of the next button
		pos = pos.Add(image.Point{buttWidth, 0})
	}
}

// Draw xxx
func (riff *Riff) redraw() {
	size := gui.WinCurrSize(riff)
	rect := image.Rect(0, 0, size.X, size.Y)
	gui.DoUpstream(riff, "setcolor", gui.BackColor)
	gui.DoUpstream(riff, "drawfilledrect", rect.Inset(1))
	gui.DoUpstream(riff, "setcolor", gui.ForeColor)
	gui.DoUpstream(riff, "drawrect", rect)
	gui.RedrawChildren(riff)
}
