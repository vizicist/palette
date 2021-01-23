package gui

import (
	"image"
	"log"
)

// Button xxx
type Button struct {
	WindowData
	isPressed bool
	label     string
	labelX    int
	labelY    int
}

// NewButton xxx
func NewButton(parent Window, text string) *Button {
	w := &Button{
		WindowData: NewWindowData(parent),
		isPressed:  false,
		label:      text,
	}
	return w
}

// Data xxx
func (button *Button) Data() *WindowData {
	return &button.WindowData
}

func (button *Button) resize(rect image.Rectangle) {

	// Get the real bounds needed for this label
	brect := button.Style.BoundString(button.label)
	// extra 6 for embossing of button
	w := brect.Max.Sub(brect.Min).X + 6
	h := brect.Max.Sub(brect.Min).Y + 6
	button.Rect = image.Rect(rect.Min.X, rect.Min.Y, rect.Min.X+w, rect.Min.Y+h)

	button.labelX = button.Rect.Min.X + 3
	button.labelY = button.Rect.Min.Y + 3
	button.labelY -= brect.Min.Y
}

func (button *Button) redraw() {
	DoUpstream(button, "drawrect", button.Rect)
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
