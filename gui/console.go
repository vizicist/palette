package gui

import (
	"image"
	"image/color"
	"log"
	"strings"
)

// Console is a window that has a couple of buttons
type Console struct {
	WindowData
	clearButton Window
	textArea    Window
}

// NewConsole xxx
func NewConsole(parent Window) Window {

	console := &Console{
		WindowData: NewWindowData(parent),
	}

	console.clearButton = AddChild(console, "clearButton", NewButton(console, "Clear"))
	console.textArea = AddChild(console, "textArea", NewScrollingText(console))

	return console
}

// DoUpstream xxx
func (console *Console) DoUpstream(w Window, cmd UpstreamCmd) {
	console.parent.DoUpstream(console, cmd)
}

// DoDownstream xxx
func (console *Console) DoDownstream(t DownstreamCmd) {
	switch cmd := t.(type) {
	case MouseCmd:
		o, _ := WindowUnder(console, cmd.Pos)
		if o != nil {
			o.DoDownstream(cmd)
		}
	case ResizeCmd:
		console.resize(cmd.Rect)
	case RedrawCmd:
		console.redraw()
	case CloseYourselfCmd:
		log.Printf("console: CloseYourself\n")
	default:
		log.Printf("Unhandled cmd=%v\n", cmd)
	}
}

// AddLine xxx
func (console *Console) AddLine(s string) {
	if strings.HasSuffix(s, "\n") {
		// XXX - remove it?
	}
	console.children["textArea"].DoDownstream(AddLineCmd{s})
}

// Data xxx
func (console *Console) Data() *WindowData {
	return &console.WindowData
}

// Resize xxx
func (console *Console) resize(rect image.Rectangle) image.Rectangle {

	console.Rect = rect

	rowHeight := console.Style.TextHeight() + 2
	// See how many rows we can fit in the rect ()
	nrows := rect.Dy() / rowHeight

	// handle Clear button
	// In Resize, the rect.Max values get recomputed to fit the button
	console.clearButton.DoDownstream(ResizeCmd{image.Rect(rect.Min.X+2, rect.Min.Y+2, rect.Max.X, rect.Max.Y)})

	// handle ScrollingText Window
	y0 := rect.Min.Y + rowHeight + 4
	y1 := rect.Min.Y + nrows*rowHeight
	console.textArea.DoDownstream(ResizeCmd{image.Rect(rect.Min.X+2, y0, rect.Max.X, y1)})

	// Adjust console's oveall size from the ScrollingText Window
	r := console.textArea.Data().Rect
	console.Rect.Max.Y = r.Max.Y

	return console.Rect
}

// Draw xxx
func (console *Console) redraw() {

	green := color.RGBA{0, 0xff, 0, 0xff}
	console.DoUpstream(console, DrawRectCmd{console.Rect, green})
	RedrawChildren(console)
}
