package tool

import (
	"image"
	"log"

	"github.com/vizicist/palette/gui"
)

// Console is a window that has a couple of buttons
type Console struct {
	clearButton gui.Window
	testButton  gui.Window
	threeButton gui.Window
	TextArea    gui.Window
}

// NewConsole xxx
func NewConsole(style *gui.Style) (w gui.Window, minRect image.Rectangle) {

	console := &Console{}

	console.clearButton, _ = gui.NewButton("Clear", style)
	console.testButton, _ = gui.NewButton("Test", style)
	console.threeButton, _ = gui.NewButton("Three", style)
	console.TextArea, _ = gui.NewScrollingText(style)

	gui.SetAttValue(console, "islogger", "true")

	gui.AddChild(console, console.TextArea)
	gui.AddChild(console, console.clearButton)
	gui.AddChild(console, console.testButton)
	gui.AddChild(console, console.threeButton)

	return console, minRect
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
		gui.DoUpstream(console, cmd, arg)
		// console.Data().Parent.Do(console, cmd, arg)
	}
	return nil, nil
}

// Resize xxx
func (console *Console) resize(rect image.Rectangle) {

	buttWidth := rect.Dx() / 4

	// Clear button
	r := gui.WinMinRect(console.clearButton) // minimum good size
	r.Max.X = buttWidth                      // force width
	r = r.Add(rect.Min).Add(image.Point{2, 2})
	console.clearButton.Do(console, "resize", r)

	// Test button
	r = r.Add(image.Point{buttWidth + 2, 0})
	console.testButton.Do(console, "resize", r)

	// Three button
	r = r.Add(image.Point{buttWidth + 2, 0})
	console.threeButton.Do(console, "resize", r)

	// handle ScrollingText Window
	y0 := gui.WinRect(console.clearButton).Max.Y + 2
	console.TextArea.Do(console, "resize", image.Rect(rect.Min.X+2, y0, rect.Max.X-2, gui.WinRect(console).Max.Y))

	// Adjust console's oveall size from the ScrollingText Window
	log.Printf("Hey, Console.resize wants to adjust Rect.Max.Y\n")
	// console.Data().Rect.Max.Y = console.TextArea.Data().Rect.Max.Y + 2
}

// Draw xxx
func (console *Console) redraw() {
	gui.DoUpstream(console, "setcolor", gui.BackColor)
	gui.DoUpstream(console, "drawfilledrect", gui.WinRect(console).Inset(1))
	gui.DoUpstream(console, "setcolor", gui.ForeColor)
	gui.DoUpstream(console, "drawrect", gui.WinRect(console))
	gui.RedrawChildren(console)
}
