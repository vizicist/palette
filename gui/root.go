package gui

import (
	"image"
	"log"
	// Don't be tempted to use go-gl
)

// Root xxx
type Root struct {
	WindowData
	console  *Console
	console2 *Console
}

// NewRoot xxx
func NewRoot(style *Style) (*Root, error) {
	w := &Root{
		WindowData: WindowData{
			style:   style,
			rect:    image.Rectangle{},
			objects: make(map[string]Window),
		},
		console: nil,
	}
	return w, nil
}

// Data xxx
func (root *Root) Data() WindowData {
	return root.WindowData
}

// Resize xxx
func (root *Root) Resize(r image.Rectangle) {
	root.rect = r
	midy := (r.Min.Y + r.Max.Y) / 2

	r1 := image.Rect(r.Min.X, r.Min.Y, r.Max.X, midy-5)
	root.console.Resize(r1)

	r2 := image.Rect(r.Min.X, midy+5, r.Max.X, r.Max.Y)
	root.console2.Resize(r2)
}

// HandleMouseInput xxx
func (root *Root) HandleMouseInput(pos image.Point, button int, mdown bool) bool {
	if !pos.In(root.rect) {
		log.Printf("RootWindow.HandleMouseInput: pos not in rect!\n")
		return false
	}
	log.Printf("Hey, RootWindow.HandleMousInput needs work!\n")
	return true
}
