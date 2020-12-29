package gui

import (
	"image"

	"github.com/fogleman/gg"
)

// Text xxx
type Text struct {
	WindowData
	name  string
	text  string
	style Style
	rect  image.Rectangle
}

// NewText xxx
func NewText(text string, rect image.Rectangle, style Style) *Text {
	return &Text{
		text:  text,
		style: style,
		rect:  rect,
	}
}

// Name xxx
func (t *Text) Name() string {
	return t.name
}

// HandleMouseInput xxx
func (t *Text) HandleMouseInput(pos image.Point, button int, down bool) bool {
	return false
}

// Draw xxx
func (t *Text) Draw(ctx *gg.Context) {
	t.style.Do(ctx)
	ctx.DrawString(t.text, float64(t.rect.Min.X), float64(t.rect.Min.Y))
}
