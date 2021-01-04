package egui

import (
	"image"
	"image/color"
	"log"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/text"
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
	brect := text.BoundString(b.style.fontFace, b.label)
	w := brect.Max.Sub(brect.Min).X
	h := brect.Max.Sub(brect.Min).Y
	b.rect = image.Rect(rect.Min.X, rect.Min.Y, rect.Min.X+w, rect.Min.Y+h)
}

// Draw xxx
func (b *Button) Draw(screen *Screen) {

	color := color.RGBA{0, 0, 0xff, 0xff}

	screen.DrawRect(b.rect, color)

	// This is bogus stuff
	textx := b.rect.Min.X
	// XXX - why Max + Dy/2?  Shouldn't it be Min.Y + Dy/2?
	texty := b.rect.Max.Y + b.rect.Dy()/2.0
	brect := text.BoundString(b.style.fontFace, b.label)
	texty += brect.Min.Y

	// XXX - this should call something in Screen
	text.Draw(screen.eimage, b.label, b.style.fontFace, textx, texty, color)
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
