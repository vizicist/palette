package winsys

import (
	"image"
	"log"

	"github.com/vizicist/palette/engine"
)

// Button xxx
type Button struct {
	ctx       WinContext
	isPressed bool
	label     string
	labelCurr string // might be truncated because of size
}

// NewButton xxx
func NewButton(parent Window, label string) WindowData {
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
func (button *Button) Do(cmd engine.Cmd) string {

	switch cmd.Subj {
	case "resize":
		x := cmd.ValuesInt("x", 0)
		y := cmd.ValuesInt("y", 0)
		toPoint := image.Point{X: x, Y: y}
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
		DoUpstream(button, NewSetColorCmd(ForeColor))

		DoUpstream(button, NewDrawRectCmd(image.Rectangle{Max: WinCurrSize(button)}))

		style := WinStyle(button)
		labelPos := image.Point{3, style.TextHeight()}
		DoUpstream(button, NewDrawTextCmd(button.labelCurr, style.fontFace, labelPos))

	case "mouse":
		currSize := WinCurrSize(button)
		currRect := image.Rect(0, 0, currSize.X, currSize.Y)
		pos := cmd.ValuesPos(image.Point{0, 0})
		if !pos.In(currRect) {
			log.Printf("button: pos not in Rect?\n")
			break
		}
		ddu := cmd.ValuesString("ddu", "")
		switch ddu {
		case "down":
			if !button.isPressed {
				button.isPressed = true
				DoUpstream(button, NewButtonDownCmd(button.label))
			}
		case "up":
			if button.isPressed {
				button.isPressed = false
				DoUpstream(button, NewButtonUpCmd(button.label))
			}
		}

	}
	return ""
}