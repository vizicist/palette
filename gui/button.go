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
func NewButton(parent Window, text string) Window {
	w := &Button{
		WindowData: NewWindowData(parent),
		isPressed:  false,
		label:      text,
	}
	log.Printf("NewButton: go w.Run()\n")
	// go w.readFromUpstream()
	return w
}

// Data xxx
func (button *Button) Data() *WindowData {
	return &button.WindowData
}

func (button *Button) resize(rect image.Rectangle) image.Rectangle {

	// Get the real bounds needed for this label
	brect := button.Style.BoundString(button.label)
	// extra 6 for embossing of button
	w := brect.Max.Sub(brect.Min).X + 6
	h := brect.Max.Sub(brect.Min).Y + 6
	button.Rect = image.Rect(rect.Min.X, rect.Min.Y, rect.Min.X+w, rect.Min.Y+h)

	button.labelX = button.Rect.Min.X + 3
	button.labelY = button.Rect.Min.Y + 3
	button.labelY -= brect.Min.Y

	return button.Rect
}

func (button *Button) redraw() {
	button.DoUpstream(button, DrawRectCmd{button.Rect, button.Style.strokeColor})
	button.DoUpstream(button, DrawTextCmd{button.label, button.Style.fontFace, image.Point{button.labelX, button.labelY}, button.Style.textColor})
}

// DoUpstream xxx
func (button *Button) DoUpstream(w Window, cmd UpstreamCmd) {
	button.parent.DoUpstream(button, cmd)
}

// DoDownstream xxx
func (button *Button) DoDownstream(t DownstreamCmd) {

	switch cmd := t.(type) {
	case ResizeCmd:
		button.resize(cmd.Rect)
	case RedrawCmd:
		button.redraw()
	case MouseCmd:
		if !cmd.Pos.In(button.Rect) {
			log.Printf("button: pos not in Rect?\n")
			return
		}
		switch cmd.Ddu {
		case MouseDown:
			// The mouse is inside the button
			if button.isPressed == false {
				button.isPressed = true
				log.Printf("Should be calling button down\n")
				// button.callback("down")
			}
		case MouseUp:
			if button.isPressed == true {
				button.isPressed = false
				log.Printf("Should be calling button up\n")
				// button.callback("up")
			}
		}

	}
}
