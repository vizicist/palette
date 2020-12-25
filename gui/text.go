package gui

import (
	"image"
	"strings"

	"github.com/micaelAlastor/nanovgo"
)

// VizText xxx
type VizText struct {
	name  string
	text  string
	style Style
	rect  image.Rectangle
}

// NewText xxx
func NewText(name, text string, rect image.Rectangle, style Style) *VizText {
	if !strings.HasPrefix(name, "text.") {
		name = "text." + name
	}
	return &VizText{
		name:  name,
		text:  text,
		style: style,
		rect:  rect,
	}
}

// Name xxx
func (t *VizText) Name() string {
	return t.name
}

// HandleMouseInput xxx
func (t *VizText) HandleMouseInput(mx, my int, mdown bool) {
}

// Draw xxx
func (t *VizText) Draw(ctx *nanovgo.Context) {
	t.style.Do(ctx)
	ctx.SetFillColor(t.style.textColor)
	ctx.SetTextAlign(nanovgo.AlignLeft | nanovgo.AlignTop)
	ctx.Text(float32(t.rect.Min.X), float32(t.rect.Min.Y), t.text)
}
