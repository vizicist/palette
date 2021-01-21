package gui

import (
	"image"
	"image/color"
	"strings"
)

// Console is a window that has a couple of buttons
type Console struct {
	WindowData
	b1 Window
	t1 Window
}

// NewConsole xxx
func NewConsole(parent Window) Window {

	console := &Console{
		WindowData: NewWindowData(parent),
	}

	AddChild(console, "b1", NewButton(console, "Clear"))
	AddChild(console, "t1", NewScrollingText(console))

	return console
}

// DoUpstream xxx
func (console *Console) DoUpstream(w Window, cmd UpstreamCmd) {
}

// DoDownstream xxx
func (console *Console) DoDownstream(t DownstreamCmd) {
	switch cmd := t.(type) {
	case MouseCmd:
		o, _ := WindowUnder(console, cmd.Pos)
		if o != nil {
			o.DoDownstream(cmd)
		}
	}
}

/*
func (console *Console) clear() {
	console.t1.DoUpstream(ClearCmd{})
}
*/

// AddLine xxx
func (console *Console) AddLine(s string) {
	if strings.HasSuffix(s, "\n") {
		// XXX - remove it?
	}
	console.t1.DoDownstream(AddLineCmd{s})
}

// Data xxx
func (console *Console) Data() *WindowData {
	return &console.WindowData
}

// Resize xxx
func (console *Console) Resize(rect image.Rectangle) image.Rectangle {

	console.Rect = rect

	rowHeight := console.Style.TextHeight() + 2
	// See how many rows we can fit in the rect ()
	nrows := rect.Dy() / rowHeight

	// handle Clear button
	// In Resize, the rect.Max values get recomputed to fit the button
	console.b1.DoDownstream(ResizeCmd{image.Rect(rect.Min.X+2, rect.Min.Y+2, rect.Max.X, rect.Max.Y)})

	// handle ScrollingText Window
	y0 := rect.Min.Y + rowHeight + 4
	y1 := rect.Min.Y + nrows*rowHeight
	console.t1.DoDownstream(ResizeCmd{image.Rect(rect.Min.X+2, y0, rect.Max.X, y1)})

	// Adjust console's oveall size from the ScrollingText Window
	r := WindowRect(console.t1)
	console.Rect.Max.Y = r.Max.Y

	return console.Rect
}

// Draw xxx
func (console *Console) Draw() {

	green := color.RGBA{0, 0xff, 0, 0xff}
	console.DoDownstream(DrawRectCmd{console.Rect, green})
	RedrawChildren(console)
}
