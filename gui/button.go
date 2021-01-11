package gui

import (
	"image"
	"image/color"
	"log"
	"strings"
)

// ButtonCallback xxx
type ButtonCallback func(updown string)

// Button xxx
type Button struct {
	WindowData
	isPressed bool
	callback  ButtonCallback
	label     string
	labelX    int
	labelY    int
}

// NewButton xxx
func NewButton(parent Window, text string, cb ButtonCallback) *Button {
	return &Button{
		WindowData: WindowData{
			screen:  parent.Data().screen,
			style:   parent.Data().style,
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
func (b *Button) Resize(rect image.Rectangle) image.Rectangle {

	desiredHeight := rect.Dy() * (1 + strings.Count(b.label, "\n"))

	if desiredHeight != b.style.fontHeight {
		b.style = NewStyle("regular", desiredHeight)
	}

	// Get the real bounds needed for this label
	brect := b.style.BoundString(b.label)
	w := brect.Max.Sub(brect.Min).X
	h := brect.Max.Sub(brect.Min).Y
	b.rect = image.Rect(rect.Min.X, rect.Min.Y, rect.Min.X+w, rect.Min.Y+h)

	b.labelX = b.rect.Min.X
	b.labelY = b.rect.Min.Y
	b.labelY -= brect.Min.Y

	return b.rect
}

// Draw xxx
func (b *Button) Draw() {

	color := color.RGBA{0, 0, 0xff, 0xff}

	b.screen.drawRect(b.rect, color)

	b.screen.drawText(b.label, b.style.fontFace, b.labelX, b.labelY, color)
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
