package gui

import (
	"image"
	"log"
	"strings"
)

// Console is a window that has a couple of buttons
type Console struct {
	WindowData
	b1 *Button
	t1 *ScrollingText
}

// NewConsole xxx
func NewConsole(parent Window) *Console {

	console := &Console{
		WindowData: NewWindowData(parent),
	}
	console.b1 = NewButton(console, "Clear",
		func(updown string) {
			log.Printf("Clear button: %s\n", updown)
			console.clear()
		})
	console.t1 = NewScrollingText(console)

	AddObject(console, "clear", console.b1)
	AddObject(console, "text", console.t1)

	go console.Run()

	return console
}

func (console *Console) clear() {
	console.t1.clear()
}

// AddLine xxx
func (console *Console) AddLine(s string) {
	if strings.HasSuffix(s, "\n") {
		// XXX - remove it?
	}
	console.t1.AddLine(s)
}

// Data xxx
func (console *Console) Data() *WindowData {
	return &console.WindowData
}

// Resize xxx
func (console *Console) Resize(rect image.Rectangle) image.Rectangle {

	console.rect = rect

	rowHeight := console.style.TextHeight() + 2
	// See how many rows we can fit in the rect ()
	nrows := rect.Dy() / rowHeight

	// handle Clear button
	// In Resize, the rect.Max values get recomputed to fit the button
	r := image.Rect(rect.Min.X+2, rect.Min.Y+2, rect.Max.X, rect.Max.Y)
	console.b1.Resize(r)

	// handle ScrollingText Window
	y0 := rect.Min.Y + rowHeight + 4
	y1 := rect.Min.Y + nrows*rowHeight
	tr := image.Rect(rect.Min.X+2, y0, rect.Max.X, y1)
	console.t1.Resize(tr)

	// Adjust console's oveall size from the ScrollingText Window
	console.rect.Max.Y = console.t1.rect.Max.Y

	return console.rect
}

// Run xxx
func (console *Console) Run() {
	for {
		me := <-console.mouseChan
		console.screen.log("Console.Run: me=%v", me)
		o := ObjectUnder(console, me.pos)
		if o != nil {
			o.Data().mouseChan <- me
		}
	}
}

// Draw xxx
func (console *Console) Draw() {

	console.screen.drawRect(console.rect, green)

	console.b1.Draw()
	console.t1.Draw()
}
