package gui

import (
	"image"
	"log"
)

// Button xxx
type Button struct {
	ctx       WinContext
	isPressed bool
	label     string
	labelCurr string // might be truncated because of size
}

// NewButton xxx
func NewButton(parent Window, label string) ToolData {
	b := &Button{
		ctx:       NewWindowContext(parent),
		isPressed: false,
		label:     label,
		labelCurr: label,
	}
	style := WinStyle(b)
	minSize := image.Point{
		X: style.TextWidth(label) + 6,
		Y: style.TextHeight() + 6,
	}
	return NewToolData(b, "Button", minSize)
}

// Context xxx
func (button *Button) Context() *WinContext {
	return &button.ctx
}

// Do xxx
func (button *Button) Do(cmd string, arg interface{}) (interface{}, error) {

	switch cmd {

	case "resize":
		toPoint := ToPoint(arg)
		minSize := WinMinSize(button)
		if toPoint.X < minSize.X || toPoint.Y < minSize.Y {
			style := WinStyle(button)
			nchars := toPoint.X / style.CharWidth()
			if nchars <= 0 {
				button.labelCurr = ""
			} else if len(button.label) > nchars {
				button.labelCurr = button.label[:nchars]
			}
		} else {
			button.labelCurr = button.label
		}

	case "redraw":
		DoUpstream(button, "setcolor", ForeColor)
		rect := image.Rectangle{Max: WinCurrSize(button)}
		DoUpstream(button, "drawrect", rect)
		style := WinStyle(button)
		labelPos := image.Point{3, style.TextHeight()}
		DoUpstream(button, "drawtext", DrawTextCmd{button.labelCurr, style.fontFace, labelPos})

	case "mouse":
		mouse := ToMouse(arg)
		currSize := WinCurrSize(button)
		currRect := image.Rect(0, 0, currSize.X, currSize.Y)
		if !mouse.Pos.In(currRect) {
			log.Printf("button: pos not in Rect?\n")
			break
		}
		switch mouse.Ddu {
		case MouseDown:
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
