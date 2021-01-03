package gui

import (
	"image"
	"log"
	"strings"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
)

// ButtonCallback xxx
type ButtonCallback func(updown string)

// Button xxx
type Button struct {
	WindowData
	isPressed bool
	label     string
	callback  ButtonCallback
}

// NewButton xxx
func NewButton(style *Style, text string, cb ButtonCallback) *Button {
	return &Button{
		WindowData: WindowData{
			style:   style,
			rect:    image.Rectangle{},
			objects: map[string]Window{},
		},
		isPressed: false,
		label:     text,
		callback:  cb,
	}
}

// Data xxx
func (b *Button) Data() WindowData {
	return b.WindowData
}

// Resize xxx
func (b *Button) Resize(rect image.Rectangle) {

	desiredHeight := rect.Dy() * (1 + strings.Count(b.label, "\n"))

	if desiredHeight != b.style.fontHeight {
		b.style = NewStyle("regular", desiredHeight)
	}

	// Get the real bounds needed for this label
	brect, _ := font.BoundString(b.style.fontFace, b.label)
	w := brect.Max.Sub(brect.Min).X.Round()
	h := brect.Max.Sub(brect.Min).Y.Round()

	b.rect = image.Rect(rect.Min.X, rect.Min.Y, rect.Min.X+w, rect.Min.Y+h)
}

// Draw xxx
func (b *Button) Draw(ctx *gg.Context) {

	var cornerRadius float64 = 4.0
	b.style.SetForDrawing(ctx)

	// interior
	w := float64(b.rect.Max.X - b.rect.Min.X)
	h := float64(b.rect.Max.Y - b.rect.Min.Y)
	ctx.DrawRoundedRectangle(float64(b.rect.Min.X+1), float64(b.rect.Min.Y+1), w-2, h-2, cornerRadius-1)
	ctx.Fill()

	// outline
	ctx.DrawRoundedRectangle(float64(b.rect.Min.X), float64(b.rect.Min.Y), w, h, cornerRadius-1)
	ctx.Stroke()

	// label
	b.style.SetForText(ctx)

	textx := float64(b.rect.Min.X)
	// XXX - why Max + Dy/2?  Shouldn't it be Min.Y + Dy/2?
	texty := float64(b.rect.Max.Y) + float64(b.rect.Dy())/2.0
	brect, _ := font.BoundString(b.style.fontFace, b.label)
	// XXX - hmmmmmm
	texty += float64(brect.Min.Y.Round())
	ctx.DrawString(b.label, textx, texty)
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
