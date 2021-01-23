package gui

import (
	"image"
	"log"
	"strings"
)

// Console is a window that has a couple of buttons
type Console struct {
	WindowData
	clearButton Window
	textArea    *ScrollingText
}

// NewConsole xxx
func NewConsole(parent Window) *Console {

	console := &Console{
		WindowData: NewWindowData(parent),
	}

	console.clearButton = NewButton(console, "Clear")
	console.textArea = NewScrollingText(console)

	AddChild(console, "textArea", console.textArea)
	AddChild(console, "clearButton", console.clearButton)

	return console
}

// DoSync xxx
func (console *Console) DoSync(from Window, cmd string, arg interface{}) (result interface{}, err error) {
	return NoSyncInterface("Console")
}

// Do xxx
func (console *Console) Do(from Window, cmd string, arg interface{}) {
	switch cmd {
	case "mouse":
		mouse := ToMouse(arg)
		o, _ := WindowUnder(console, mouse.Pos)
		if o != nil {
			o.Do(console, cmd, arg)
		}
	case "resize":
		console.resize(ToRect(arg))
	case "redraw":
		console.redraw()
	case "closeyourself":
		log.Printf("console: CloseYourself needs work?\n")
	case "buttondown":
		// Clear is the only button
		log.Printf("console: buttondown of %s\n", ToString(arg))
		console.textArea.Clear()
		console.textArea.AddLine("This is after clear")
		console.textArea.AddLine("second line after clear")
	case "buttonup":
		log.Printf("console: buttonup of %s\n", ToString(arg))
	default:
		console.parent.Do(console, cmd, arg)
	}
}

// AddLine xxx
func (console *Console) AddLine(s string) {
	if strings.HasSuffix(s, "\n") {
		// XXX - remove it?
	}
	console.children["textArea"].Do(console, "addline", AddLineCmd{s})
}

// Data xxx
func (console *Console) Data() *WindowData {
	return &console.WindowData
}

// Resize xxx
func (console *Console) resize(rect image.Rectangle) {

	console.Rect = rect

	rowHeight := console.Style.TextHeight() + 2
	// See how many rows we can fit in the rect ()
	// nrows := rect.Dy() / rowHeight

	// handle Clear button
	// In Resize, the rect.Max values get recomputed to fit the button
	console.clearButton.Do(console, "resize", image.Rect(rect.Min.X+2, rect.Min.Y+2, rect.Max.X, rect.Max.Y))

	// handle ScrollingText Window
	y0 := rect.Min.Y + rowHeight + 4
	// y1 := rect.Min.Y + nrows*rowHeight
	console.textArea.Do(console, "resize", image.Rect(rect.Min.X+2, y0, rect.Max.X-2, console.Rect.Max.Y))

	// Adjust console's oveall size from the ScrollingText Window
	console.Rect.Max.Y = console.textArea.Rect.Max.Y + 2
}

// Draw xxx
func (console *Console) redraw() {
	DoUpstream(console, "drawrect", console.Rect)
	RedrawChildren(console)
}
