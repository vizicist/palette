package gui

import (
	"image"
	"log"
)

// Button xxx
type Button struct {
	WindowData
	isPressed bool
	labelOrig string
	label     string
	labelX    int
	labelY    int
	noRoom    bool
}

// NewButton xxx
func NewButton(parent Window, label string) *Button {
	b := &Button{
		WindowData: NewWindowData(parent),
		isPressed:  false,
		labelOrig:  label,
		label:      label,
	}
	b.MinRect = image.Rectangle{
		Min: image.Point{0, 0},
		Max: image.Point{
			X: b.Style.TextWidth(label) + 6,
			Y: b.Style.TextHeight() + 6,
		},
	}
	return b
}

// Data xxx
func (button *Button) Data() *WindowData {
	return &button.WindowData
}

func (button *Button) resize(rect image.Rectangle) {
	button.Rect = rect
	if button.Rect.Dx() < button.MinRect.Dx() || button.Rect.Dy() < button.MinRect.Dy() {
		nchars := button.Rect.Dx() / button.Style.CharWidth()
		if len(button.labelOrig) > nchars {
			button.label = button.labelOrig[:nchars]
		}
	} else {
		button.label = button.labelOrig
	}
}

func (button *Button) redraw() {
	DoUpstream(button, "setcolor", foreColor)
	DoUpstream(button, "drawrect", button.Rect)
	button.labelX = button.Rect.Min.X + 3
	button.labelY = button.Rect.Min.Y + button.Style.TextHeight()
	DoUpstream(button, "drawtext", DrawTextCmd{button.label, button.Style.fontFace, image.Point{button.labelX, button.labelY}})
}

// Do xxx
func (button *Button) Do(from Window, cmd string, arg interface{}) {

	switch cmd {
	case "resize":
		button.resize(ToRect(arg))
	case "redraw":
		button.redraw()
	case "mouse":
		mouse := ToMouse(arg)
		if !mouse.Pos.In(button.Rect) {
			log.Printf("button: pos not in Rect?\n")
			return
		}
		switch mouse.Ddu {
		case MouseDown:
			// The mouse is inside the button
			if button.isPressed == false {
				button.isPressed = true
				DoUpstream(button, "buttondown", button.label)
			}
		case MouseUp:
			if button.isPressed == true {
				button.isPressed = false
				DoUpstream(button, "buttonup", button.label)
			}
		}

	}
}

// DoSync xxx
func (button *Button) DoSync(w Window, cmd string, arg interface{}) (result interface{}, err error) {
	return NoSyncInterface("Button")
}
