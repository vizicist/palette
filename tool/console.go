package tool

import (
	"image"
	"log"

	"github.com/vizicist/palette/gui"
)

// Console is a window that has a couple of buttons
type Console struct {
	gui.WindowData
	clearButton gui.Window
	testButton  gui.Window
	threeButton gui.Window
	TextArea    gui.Window
}

// NewConsole xxx
func NewConsole(parent gui.Window) gui.Window {

	console := &Console{
		WindowData: gui.NewWindowData(parent),
	}

	console.clearButton = gui.NewButton(console, "Clear")
	console.testButton = gui.NewButton(console, "Test")
	console.threeButton = gui.NewButton(console, "Three")
	console.TextArea = gui.NewScrollingText(console)

	gui.SetAttValue(console, "islogger", "true")

	gui.AddChild(console, console.TextArea)
	gui.AddChild(console, console.clearButton)
	gui.AddChild(console, console.testButton)
	gui.AddChild(console, console.threeButton)

	return console
}

// Do xxx
func (console *Console) Do(from gui.Window, cmd string, arg interface{}) (interface{}, error) {
	switch cmd {
	case "mouse":
		mouse := gui.ToMouse(arg)
		o := gui.WindowUnder(console, mouse.Pos)
		if o != nil {
			o.Do(console, cmd, arg)
		}
	case "resize":
		console.resize(gui.ToRect(arg))
	case "redraw":
		console.redraw()
	case "restore":
		log.Printf("Console: restore arg=%v\n", arg)
		_, err := console.TextArea.Do(console, "restore", arg)
		if err != nil {
			log.Printf("Console: restore err=%s\n", err)
			return nil, err
		}

	case "dumpstate":
		// in := string.Repeat(" ",12)
		s, err := console.TextArea.Do(console, "dumpstate", nil)
		if err != nil {
			return nil, err
		}
		return s, nil
	case "closeyourself":
		log.Printf("console: CloseYourself needs work?\n")
	case "buttondown":
		// Clear is the only button
		switch from {
		case console.testButton:
			log.Printf("Test!\n")
		case console.threeButton:
			log.Printf("Three!\n")
		case console.clearButton:
			log.Printf("Clear!\n")
			console.TextArea.Do(console, "clear", nil)
		}
	case "buttonup":
		//
	case "addline":
		console.TextArea.Do(console, cmd, arg)
	default:
		console.data().parent.Do(console, cmd, arg)
	}
	return nil, nil
}

// Data xxx
func (console *Console) data() *gui.WindowData {
	return &console.WindowData
}

// Resize xxx
func (console *Console) resize(rect image.Rectangle) {

	console.data().Rect = rect

	buttWidth := rect.Dx() / 4
	// buttHeight := console.clearButton.MinRect.Max.Y

	// Clear button
	r := console.clearButton.data().minRect // minimum good size
	r.Max.X = buttWidth                     // force width
	r = r.Add(rect.Min).Add(image.Point{2, 2})
	console.clearButton.Do(console, "resize", r)

	// Test button
	r = r.Add(image.Point{buttWidth + 2, 0})
	console.testButton.Do(console, "resize", r)

	// Three button
	r = r.Add(image.Point{buttWidth + 2, 0})
	console.threeButton.Do(console, "resize", r)

	// handle ScrollingText Window
	y0 := console.clearButton.data().Rect.Max.Y + 2
	console.TextArea.Do(console, "resize", image.Rect(rect.Min.X+2, y0, rect.Max.X-2, console.data().Rect.Max.Y))

	// Adjust console's oveall size from the ScrollingText Window
	console.data().Rect.Max.Y = console.TextArea.data().Rect.Max.Y + 2
}

// Draw xxx
func (console *Console) redraw() {
	gui.DoUpstream(console, "setcolor", gui.backColor)
	gui.DoUpstream(console, "drawfilledrect", console.data().Rect.Inset(1))
	gui.DoUpstream(console, "setcolor", gui.foreColor)
	gui.DoUpstream(console, "drawrect", console.data().Rect)
	gui.RedrawChildren(console)
}
