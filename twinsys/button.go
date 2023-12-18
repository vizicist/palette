package twinsys

import (
	"image"

	"github.com/vizicist/palette/kit"
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
	styleInfo := WinStyleInfo(b)
	minSize := image.Point{
		X: styleInfo.TextWidth(label) + 6,
		Y: styleInfo.TextHeight() + 6,
	}
	return NewToolData(b, "Button", minSize)
}

// Context xxx
func (button *Button) Context() *WinContext {
	return &button.ctx
}

// Do xxx
func (button *Button) Do(cmd kit.Cmd) string {

	switch cmd.Subj {
	case "resize":
		sz := cmd.ValuesXY("size", kit.PointZero)
		toPoint := image.Point{X: sz.X, Y: sz.Y}
		minSize := WinMinSize(button)
		if toPoint.X < minSize.X || toPoint.Y < minSize.Y {
			styleInfo := WinStyleInfo(button)
			nchars := toPoint.X / styleInfo.CharWidth()
			if nchars <= 0 {
				button.labelCurr = ""
			} else if len(button.label) > nchars {
				button.labelCurr = button.label[:nchars]
			}
		} else {
			button.labelCurr = button.label
		}

	case "redraw":
		WinDoUpstream(button, NewSetColorCmd(ForeColor))

		WinDoUpstream(button, NewDrawRectCmd(image.Rectangle{Max: WinGetSize(button)}))

		styleInfo := WinStyleInfo(button)
		labelPos := image.Point{3, styleInfo.TextHeight()}
		styleName := WinStyleName(button)
		WinDoUpstream(button, NewDrawTextCmd(button.labelCurr, styleName, labelPos))

	case "mouse":
		currSize := WinGetSize(button)
		currRect := image.Rect(0, 0, currSize.X, currSize.Y)
		pos := cmd.ValuesPos(kit.PointZero)
		if !pos.In(currRect) {
			kit.LogWarn("button: pos not in Rect?")
			break
		}
		ddu := cmd.ValuesString("ddu", "")
		switch ddu {
		case "down":
			if !button.isPressed {
				button.isPressed = true
				WinDoUpstream(button, NewButtonDownCmd(button.label))
			}
		case "up":
			if button.isPressed {
				button.isPressed = false
				WinDoUpstream(button, NewButtonUpCmd(button.label))
			}
		}

	}
	return ""
}
