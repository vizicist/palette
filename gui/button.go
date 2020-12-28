package gui

import (
	"image"
	"log"
	"strings"

	"github.com/micaelAlastor/nanovgo"
)

// ButtonCallback xxx
type ButtonCallback func(updown string)

// Button xxx
type Button struct {
	WindowData
	name      string
	isPressed bool
	text      string
	// Rect      image.Rectangle
	callback ButtonCallback
}

// NewButton xxx
func NewButton(name string, text string, cb ButtonCallback) *Button {
	return &Button{
		WindowData: WindowData{
			name:    name,
			style:   DefaultStyle,
			rect:    image.Rectangle{},
			objects: map[string]Window{},
		},
		name:      "",
		isPressed: false,
		text:      text,
		callback:  cb,
	}
}

// Name xxx
func (b *Button) Name() string {
	return b.name
}

// Objects xxx
func (b *Button) Objects() map[string]Window {
	return b.Objects()
}

// Rect xxx
func (b *Button) Rect() image.Rectangle {
	return b.rect
}

// Resize xxx
func (b *Button) Resize(r image.Rectangle) {
	fontHeight := r.Dy() / 24
	b.style = b.style.SetFontSizeByHeight(fontHeight)
	w := len(b.text) * b.style.charWidth
	h := b.style.lineHeight
	nr := image.Rect(r.Min.X, r.Min.Y, r.Min.X+w, r.Min.Y+h)
	b.WindowData.rect = nr
}

// HandleMouseInput xxx
func (b *Button) HandleMouseInput(pos image.Point, button int, mdown bool) bool {
	if !pos.In(b.rect) {
		log.Printf("Button.HandleMouseInput: pos not in rect!\n")
		return false
	}
	switch mdown {
	case true:
		// The mouse is inside the button
		if b.isPressed == false {
			b.isPressed = true
			b.callback("down")
		}
	case false:
		if b.isPressed == true {
			b.isPressed = false
			b.callback("up")
		}
	}
	return true
}

// Draw xxx
func (b *Button) Draw(ctx *nanovgo.Context) {
	var cornerRadius float32 = 4.0

	ctx.Save()
	defer ctx.Restore()

	/*
		bg := nanovgo.LinearGradient(b.x, b.y, b.x, b.y+b.h, nanovgo.RGBA(255, 255, 255, alpha), nanovgo.RGBA(0, 0, 0, alpha))
	*/
	b.style.Do(ctx)

	ctx.BeginPath()
	w := float32(b.rect.Max.X - b.rect.Min.X)
	h := float32(b.rect.Max.Y - b.rect.Min.Y)
	ctx.RoundedRect(float32(b.rect.Min.X+1), float32(b.rect.Min.Y+1), w-2, h-2, cornerRadius-1)
	ctx.Fill()

	ctx.BeginPath()
	ctx.RoundedRect(float32(b.rect.Min.X), float32(b.rect.Min.Y), w, h, cornerRadius-1)
	ctx.Stroke()

	ctx.SetTextAlign(nanovgo.AlignCenter | nanovgo.AlignMiddle)
	// Text uses the fill color, but we want it to be the strokeColor
	ctx.SetFillColor(b.style.textColor)
	pos := strings.Index(b.text, "\n")
	midx := float32((b.rect.Min.X + b.rect.Max.X) / 2)
	midy := float32((b.rect.Min.Y + b.rect.Max.Y) / 2)
	if pos >= 0 {
		// 2 lines
		line1 := b.text[:pos]
		line2 := b.text[pos+1:]
		halfHeight := float32(b.style.lineHeight) / 2.0
		ctx.Text(midx, midy-halfHeight, line1)
		ctx.Text(midx, midy+halfHeight, line2)
	} else {
		ctx.Text(midx, midy, b.text)
	}
}
