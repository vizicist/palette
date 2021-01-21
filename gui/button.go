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

// Resize xxx
func (button *Button) Resize(rect image.Rectangle) image.Rectangle {

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

// Draw xxx
func (button *Button) Draw() {
	log.Printf("Button.Draw\n")
	// clr := color.RGBA{0xff, 0xff, 0xff, 0xff}
	// button.toUpstream <- DrawRectCmd{button.Rect, clr}
	// button.toUpstream <- DrawTextCmd{button.label, button.Style.fontFace, image.Point{button.labelX, button.labelY}, clr}
}

// DoUpstream xxx
func (button *Button) DoUpstream(w Window, cmd UpstreamCmd) {
}

// DoDownstream xxx
func (button *Button) DoDownstream(cmd DownstreamCmd) {

	log.Printf("Button.DoDownstream: cmd = %v", cmd)

	/*
		switch t := inCmd.(type) {
		case ResizeCmd:
			button.Screen.Log("button.ResizeCmd=%v", t)
		case RedrawCmd:
			button.Screen.Log("button.RedrawCmd=%v", t)
		}
	*/

	/*
		case me := <-button.MouseChan:
			button.Screen.Log("Button.Run: me=%v", me)
			if !me.Pos.In(button.Rect) {
				continue
			}
			switch me.Ddu {
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
	*/

}
