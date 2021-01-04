package gui

import (
	"fmt"
	"image"
	"image/color"
	"log"
)

// Console is a window that has a couple of buttons
type Console struct {
	WindowData
	b1 *Button
	t1 *ScrollingText
}

// NewConsole xxx
func NewConsole(style *Style) *Console {
	console := &Console{
		WindowData: WindowData{
			style:   style,
			rect:    image.Rectangle{},
			objects: map[string]Window{},
		},
	}
	console.b1 = NewButton(style, "ClearLongButton",
		func(updown string) {
			log.Printf("Clear button: %s\n", updown)
		})
	console.t1 = NewScrollingText(style, 10)

	AddObject(console.objects, "clear", console.b1)
	AddObject(console.objects, "text", console.t1)

	for n := 1; n < 12; n++ {
		s := fmt.Sprintf("This is a long content Line # %d", n)
		console.t1.AddLine(s)
	}

	return console
}

// Data xxx
func (console *Console) Data() WindowData {
	return console.WindowData
}

// Resize xxx
func (console *Console) Resize(r image.Rectangle) {

	// 24 lines of text
	h := int(r.Dy() / 24.0)
	if h < 0 {
		h = 1
	}
	console.style = NewStyle("mono", h)
	console.rect = r

	// Clear button positioning
	bpos := image.Point{r.Min.X + 5, r.Min.Y + 5}
	bw := 200
	bh := 20

	b1r := image.Rect(bpos.X, bpos.Y, bpos.X+bw, bpos.Y+bh)
	console.b1.Resize(b1r)

	// Text starts half-way down
	tpos := image.Point{r.Min.X, r.Min.Y + r.Dy()/2}
	t1r := image.Rect(tpos.X, tpos.Y, r.Max.X, r.Max.Y)
	t1r = t1r.Inset(5)
	console.t1.Resize(t1r)
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
func (console *Console) Draw(screen *Screen) {

	/*
		console.style.SetForDrawing(ctx)
	*/
	green := color.RGBA{0, 0xff, 0, 0xff}

	screen.DrawRect(console.rect, green)

	console.b1.Draw(screen)
	console.t1.Draw(screen)
}
