package gui

import (
	"image"
	"log"
)

// Button xxx
type Button struct {
	data      WindowDatax
	isPressed bool
	labelOrig string
	label     string
	labelX    int
	labelY    int
	noRoom    bool
}

// NewButton xxx
func NewButton(label string, style *Style) ToolData {
	b := &Button{
		isPressed: false,
		labelOrig: label,
		label:     label,
	}
	minSize := image.Point{
		X: style.TextWidth(label) + 6,
		Y: style.TextHeight() + 6,
	}
	return ToolData{b, minSize}
}

// Data xxx
func (button *Button) Data() *WindowDatax {
	return &button.data
}

// Do xxx
func (button *Button) Do(from Window, cmd string, arg interface{}) (interface{}, error) {

	wd := WinData(button)
	switch cmd {

	case "resize":
		toRect := ToRect(arg)
		if toRect.Dx() < wd.minSize.X || toRect.Dy() < wd.minSize.Y {
			nchars := toRect.Dx() / wd.style.CharWidth()
			if nchars <= 0 {
				button.label = ""
			} else if len(button.labelOrig) > nchars {
				button.label = button.labelOrig[:nchars]
			}
		} else {
			button.label = button.labelOrig
		}

	case "redraw":
		DoUpstream(button, "setcolor", ForeColor)
		DoUpstream(button, "drawrect", wd.rect)
		rect := WinRect(button)
		button.labelX = rect.Min.X + 3
		button.labelY = rect.Min.Y + wd.style.TextHeight()
		DoUpstream(button, "drawtext", DrawTextCmd{button.label, wd.style.fontFace, image.Point{button.labelX, button.labelY}})

	case "mouse":
		mouse := ToMouse(arg)
		if !mouse.Pos.In(wd.rect) {
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
