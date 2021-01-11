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
	b1 *Button
	t1 *ScrollingText
}

// NewConsole xxx
func NewConsole(parent Window) *Console {

	console := &Console{
		WindowData: WindowData{
			screen:  parent.Data().screen,
			style:   parent.Data().style,
			rect:    image.Rectangle{},
			objects: map[string]Window{},
		},
	}
	console.b1 = NewButton(console, "Clear",
		func(updown string) {
			log.Printf("Clear button: %s\n", updown)
			console.clear()
		})
	console.t1 = NewScrollingText(console)

	AddObject(console.objects, "clear", console.b1)
	AddObject(console.objects, "text", console.t1)

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
func (console *Console) Data() WindowData {
	return console.WindowData
}

// Resize xxx
func (console *Console) Resize(rect image.Rectangle) image.Rectangle {

	console.rect = rect

	textHeight := console.style.TextHeight()
	// See how many lines we can fit in the rect
	nlines := rect.Dy() / textHeight

	// position of Clear button
	buttonWidth := 200
	buttonHeight := textHeight
	b1r := image.Rect(rect.Min.X, rect.Min.Y, rect.Min.X+buttonWidth, rect.Min.Y+buttonHeight)
	console.b1.Resize(b1r)

	// position of ScrollableTextx
	y0 := rect.Min.Y + textHeight
	// Round the height down so exactly that many lines will fit
	y1 := rect.Min.Y + nlines*textHeight
	t1r := image.Rect(rect.Min.X, y0, rect.Max.X, y1)
	// t1r = t1r.Inset(5)
	console.t1.Resize(t1r)

	return console.rect
}

// HandleMouseInput xxx
func (console *Console) HandleMouseInput(pos image.Point, button int, mdown bool) bool {
	o := ObjectUnder(console.objects, pos)
	handled := false
	if o != nil {
		o.HandleMouseInput(pos, button, mdown)
		handled = true
	} else {
		handled = false
	}
	return handled
}

// Draw xxx
func (console *Console) Draw() {

	green := color.RGBA{0, 0xff, 0, 0xff}

	console.screen.drawRect(console.rect, green)

	console.b1.Draw()
	console.t1.Draw()
}
