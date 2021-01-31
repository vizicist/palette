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
func NewButton(parent Window, label string) Window {
	b := &Button{
		WindowData: NewWindowData(parent),
		isPressed:  false,
		labelOrig:  label,
		label:      label,
	}
	b.MinRect = image.Rectangle{
		Min: image.Point{0, 0},
		Max: image.Point{
			X: b.style.TextWidth(label) + 6,
			Y: b.style.TextHeight() + 6,
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
		nchars := button.Rect.Dx() / button.style.CharWidth()
		if nchars <= 0 {
			button.label = ""
		} else if len(button.labelOrig) > nchars {
			button.label = button.labelOrig[:nchars]
		}
	} else {
		button.label = button.labelOrig
	}
}

func (button *Button) redraw() {
	DoUpstream(button, "setcolor", ForeColor)
	DoUpstream(button, "drawrect", button.Rect)
	button.labelX = button.Rect.Min.X + 3
	button.labelY = button.Rect.Min.Y + button.style.TextHeight()
	DoUpstream(button, "drawtext", DrawTextCmd{button.label, button.style.fontFace, image.Point{button.labelX, button.labelY}})
}

// Do xxx
func (button *Button) Do(from Window, cmd string, arg interface{}) (interface{}, error) {

	switch cmd {
	case "resize":
		button.resize(ToRect(arg))
	case "redraw":
		button.redraw()
	case "mouse":
		mouse := ToMouse(arg)
		if !mouse.Pos.In(button.Rect) {
			log.Printf("button: pos not in Rect?\n")
			break
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
	return nil, nil
}
