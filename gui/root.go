package gui

import (
	"image"
	"log"

	// Don't be tempted to use go-gl
	"github.com/fogleman/gg"
)

// Root xxx
type Root struct {
	WindowData
	console *Console
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
func (b *Root) Data() WindowData {
	return b.WindowData
}

// Resize xxx
func (b *Root) Resize(r image.Rectangle) {
	b.WindowData.rect = r
	for nm, o := range b.objects {
		log.Printf("Screen: resizing wind=%s rect=%v\n", nm, b.rect)
		o.Resize(b.rect)
	}
}

// HandleMouseInput xxx
func (b *Root) HandleMouseInput(pos image.Point, button int, mdown bool) bool {
	if !pos.In(b.rect) {
		log.Printf("RootWindow.HandleMouseInput: pos not in rect!\n")
		return false
	}
	log.Printf("Hey, RootWindow.HandleMousInput needs work!\n")
	return true
}

// Draw xxx
func (b *Root) Draw(ctx *gg.Context) {

	ctx.Push()
	defer ctx.Pop()

	b.console.Draw(ctx)
}
