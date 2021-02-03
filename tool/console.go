package tool

import (
	"image"
	"log"

	"github.com/vizicist/palette/gui"
)

// Console is a window that has a couple of buttons
type Console struct {
	ctx         *gui.WinContext
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

	return gui.ToolData{W: console, MinSize: image.Point{}}
}

// Context xxx
func (console *Console) Context() *gui.WinContext {
	return console.ctx
}

// Do xxx
func (console *Console) Do(cmd string, arg interface{}) (interface{}, error) {
	switch cmd {
	case "mouse":
		mouse := gui.ToMouse(arg)
		o := gui.WindowUnder(console, mouse.Pos)
		if o != nil {
			o.Do(cmd, arg)
		}
	case "resize":
		console.resize()
	case "redraw":
		console.redraw()
	case "restore":
		log.Printf("Console: restore arg=%v\n", arg)
		_, err := console.TextArea.Do("restore", arg)
		if err != nil {
			log.Printf("Console: restore err=%s\n", err)
			return nil, err
		}

	case "dumpstate":
		// in := string.Repeat(" ",12)
		s, err := console.TextArea.Do("dumpstate", nil)
		if err != nil {
			return nil, err
		}
		return s, nil
	case "closeyourself":
		log.Printf("console: CloseYourself needs work?\n")
	case "buttondown":
		// Clear is the only button
		bw := gui.ToWindow(arg)
		switch bw {
		case console.testButton:
			log.Printf("Test!\n")
		case console.threeButton:
			log.Printf("Three!\n")
		case console.clearButton:
			log.Printf("Clear!\n")
			console.TextArea.Do("clear", nil)
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

// Resize xxx
func (console *Console) resize() {

	size := gui.WinCurrSize(console)
	buttWidth := size.X / 4
	buttHeight := gui.WinMinSize(console.clearButton).Y
	buttSize := image.Point{buttWidth, buttHeight}

	pos := image.Point{2, 2}

	// layout and resize all the buttons
	for _, w := range []gui.Window{console.clearButton, console.testButton, console.threeButton} {
		gui.WinSetChildPos(console, w, pos)
		gui.WinSetChildSize(w, buttSize)
		// Advance the horizontal position of the next button
		pos = pos.Add(image.Point{buttWidth, 0})
	}

	// handle ScrollingText Window
	y0 := buttHeight + 4
	gui.WinSetChildPos(console, console.TextArea, image.Point{2, y0})
	areaSize := image.Point{size.X - 2, size.Y - y0 - 2}
	gui.WinSetChildSize(console.TextArea, areaSize)
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
