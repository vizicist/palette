package gui

import (
	"image"
	"log"
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

	SetAttValue(console, "islogger", "true")

	AddChild(console, console.textArea)
	AddChild(console, console.clearButton)

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
		o := WindowUnder(console, mouse.Pos)
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
		console.textArea.Clear()
	case "buttonup":
		//
	case "addline":
		console.textArea.Do(console, cmd, arg)
	default:
		console.parent.Do(console, cmd, arg)
	}
}

// Data xxx
func (console *Console) Data() *WindowData {
	return &console.WindowData
}

// Resize xxx
func (console *Console) resize(rect image.Rectangle) {

	console.Rect = rect

	rowHeight := console.Style.RowHeight()

	// handle Clear button
	// In Resize, the rect.Max values get recomputed to fit the button
	console.clearButton.Do(console, "resize", image.Rect(rect.Min.X+2, rect.Min.Y+2, rect.Max.X, rect.Max.Y))

	// handle ScrollingText Window
	y0 := rect.Min.Y + rowHeight + 4
	console.textArea.Do(console, "resize", image.Rect(rect.Min.X+2, y0, rect.Max.X-2, console.Rect.Max.Y))

	// Adjust console's oveall size from the ScrollingText Window
	console.Rect.Max.Y = console.textArea.Rect.Max.Y + 2
}

// Draw xxx
func (console *Console) redraw() {
	DoUpstream(console, "setcolor", backColor)
	DoUpstream(console, "drawfilledrect", console.Rect.Inset(1))
	DoUpstream(console, "setcolor", foreColor)
	DoUpstream(console, "drawrect", console.Rect)
	RedrawChildren(console)
}
