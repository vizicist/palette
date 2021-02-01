package tool

import (
	"image"
	"log"

	"github.com/vizicist/palette/gui"
)

// Console is a window that has a couple of buttons
type Console struct {
	data        *gui.WindowDatax
	clearButton gui.Window
	testButton  gui.Window
	threeButton gui.Window
	TextArea    gui.Window
}

// NewConsole xxx
func NewConsole(style *gui.Style) gui.ToolData {

	console := &Console{}

	// console.clearButton, mn = gui.NewButton("Clear", style)
	// gui.AddChild(console, console.clearButton, mn)
	console.clearButton = gui.AddChild(console, gui.NewButton("Clear", style))

	console.testButton = gui.AddChild(console, gui.NewButton("Test", style))

	console.threeButton = gui.AddChild(console, gui.NewButton("Three", style))

	console.TextArea = gui.AddChild(console, gui.NewScrollingText(style))

	gui.SetAttValue(console, "islogger", "true")

	return gui.ToolData{W: console, MinSize: image.Point{}}
}

// Data xxx
func (console *Console) Data() *gui.WindowDatax {
	return console.data
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

	pos := rect.Min.Add(image.Point{2, 2})

	// Clear button
	r := image.Rectangle{
		Min: pos,
		Max: pos.Add(gui.WinMinSize(console.clearButton)),
	}
	r.Max.X = buttWidth // force width
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
