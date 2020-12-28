package gui

import (
	"image"
	"log"

	"github.com/micaelAlastor/nanovgo"
)

// Console is a window that has a couple of buttons
type Console struct {
	WindowData
	b1 *Button
}

// NewConsole xxx
func NewConsole(name string) *Console {
	console := &Console{
		WindowData: WindowData{
			name:    name,
			style:   DefaultStyle,
			rect:    image.Rectangle{},
			objects: map[string]Window{},
		},
	}

	console.b1 = NewButton("clear", "Clear",
		func(updown string) {
			log.Printf("Clear button: %s\n", updown)
		})

	AddObject(console.objects, console.b1)

	return console
}

// Name xxx
func (console *Console) Name() string {
	return console.name
}

// Rect xxx
func (console *Console) Rect() image.Rectangle {
	return console.rect
}

// Objects xxx
func (console *Console) Objects() map[string]Window {
	return console.Objects()
}

// Resize xxx
func (console *Console) Resize(r image.Rectangle) {
	// 24 lines of text
	h := int(r.Dy() / 24.0)
	if h < 0 {
		h = 1
	}
	console.style = console.style.SetFontSizeByHeight(h)
	console.rect = r

	b1r := r.Inset(5)
	console.b1.Resize(b1r)
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
func (console *Console) Draw(ctx *nanovgo.Context) {

	var cornerRadius float32 = 4.0

	ctx.Save()
	defer ctx.Restore()

	console.style.Do(ctx)

	ctx.SetStrokeWidth(3.0)
	ctx.BeginPath()
	w := float32(console.rect.Max.X - console.rect.Min.X)
	h := float32(console.rect.Max.Y - console.rect.Min.Y)
	ctx.RoundedRect(float32(console.rect.Min.X+1), float32(console.rect.Min.Y+1), w-2, h-2, cornerRadius-1)
	ctx.Fill()

	ctx.BeginPath()
	ctx.RoundedRect(float32(console.rect.Min.X), float32(console.rect.Min.Y), w, h, cornerRadius-1)
	ctx.Stroke()

	console.b1.Draw(ctx)
}
