package gui

import (
	"image"
	"image/color"
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
	b := &Button{
		WindowData: NewWindowData(parent),
		isPressed:  false,
		label:      text,
		callback:   cb,
	}
	go b.Run()
	return b
}

// Data xxx
func (button *Button) Data() *WindowData {
	return &button.WindowData
}

// Resize xxx
func (button *Button) Resize(rect image.Rectangle) image.Rectangle {

	// Get the real bounds needed for this label
	brect := button.style.BoundString(button.label)
	// extra 6 for embossing of button
	w := brect.Max.Sub(brect.Min).X + 6
	h := brect.Max.Sub(brect.Min).Y + 6
	button.rect = image.Rect(rect.Min.X, rect.Min.Y, rect.Min.X+w, rect.Min.Y+h)

	button.labelX = button.rect.Min.X + 3
	button.labelY = button.rect.Min.Y + 3
	button.labelY -= brect.Min.Y

	return button.rect
}

// Draw xxx
func (button *Button) Draw() {

	color := color.RGBA{0xff, 0xff, 0xff, 0xff}

	button.screen.drawRect(button.rect, color)

	button.screen.drawText(button.label, button.style.fontFace, button.labelX, button.labelY, color)
}

// Run xxx
func (button *Button) Run() {
	for {
		me := <-button.mouseChan
		button.screen.log("Button.Run: me=%v", me)
		if !me.pos.In(button.rect) {
			continue
		}
		switch me.ddu {
		case MouseDown:
			// The mouse is inside the button
			if button.isPressed == false {
				button.isPressed = true
				button.callback("down")
			}
		case MouseUp:
			if button.isPressed == true {
				button.isPressed = false
				button.callback("up")
			}
		}
	}
}
