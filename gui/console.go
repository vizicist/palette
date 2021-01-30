package gui

import (
	"image"
	"log"
)

// Console is a window that has a couple of buttons
type Console struct {
	WindowData
	clearButton *Button
	testButton  *Button
	threeButton *Button
	TextArea    *ScrollingText
}

// NewConsole xxx
func NewConsole(parent Window) *Console {

	console := &Console{
		WindowData: NewWindowData(parent),
	}

	console.clearButton = NewButton(console, "Clear")
	console.testButton = NewButton(console, "Test")
	console.threeButton = NewButton(console, "Three")
	console.TextArea = NewScrollingText(console)

	SetAttValue(console, "islogger", "true")

	AddChild(console, console.TextArea)
	AddChild(console, console.clearButton)
	AddChild(console, console.testButton)
	AddChild(console, console.threeButton)

	return console
}

// Do xxx
func (console *Console) Do(from Window, cmd string, arg interface{}) (interface{}, error) {
	switch cmd {
	case "mouse":
		mouse := ToMouse(arg)
		o := WindowUnder(console, mouse.Pos)
		if o != nil {
			o.Do(console, cmd, arg)
		}
	case "resize":
		console.resize(ToRect(arg))
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
			console.TextArea.Clear()
		}
	case "buttonup":
		//
	case "addline":
		console.TextArea.Do(console, cmd, arg)
	default:
		console.parent.Do(console, cmd, arg)
	}
	return nil, nil
}

// Data xxx
func (console *Console) data() *WindowData {
	return &console.WindowData
}

// Resize xxx
func (console *Console) resize(rect image.Rectangle) {

	console.Rect = rect

	buttWidth := rect.Dx() / 4
	// buttHeight := console.clearButton.MinRect.Max.Y

	// Clear button
	r := console.clearButton.minRect // minimum good size
	r.Max.X = buttWidth              // force width
	r = r.Add(rect.Min).Add(image.Point{2, 2})
	console.clearButton.Do(console, "resize", r)

	// Test button
	r = r.Add(image.Point{buttWidth + 2, 0})
	console.testButton.Do(console, "resize", r)

	// Three button
	r = r.Add(image.Point{buttWidth + 2, 0})
	console.threeButton.Do(console, "resize", r)

	// handle ScrollingText Window
	y0 := console.clearButton.Rect.Max.Y + 2
	console.TextArea.Do(console, "resize", image.Rect(rect.Min.X+2, y0, rect.Max.X-2, console.Rect.Max.Y))

	// Adjust console's oveall size from the ScrollingText Window
	console.Rect.Max.Y = console.TextArea.Rect.Max.Y + 2
}

// Draw xxx
func (console *Console) redraw() {
	DoUpstream(console, "setcolor", backColor)
	DoUpstream(console, "drawfilledrect", console.Rect.Inset(1))
	DoUpstream(console, "setcolor", foreColor)
	DoUpstream(console, "drawrect", console.Rect)
	RedrawChildren(console)
}
