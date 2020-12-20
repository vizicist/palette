package gui

import (
	"github.com/micaelAlastor/nanovgo"
)

// VizText xxx
type VizText struct {
	name  string
	text  string
	style Style
	x, y  float32
	w, h  float32
}

// NewText xxx
func NewText(name, text string, x, y, w, h float32, style Style) *VizText {
	return &VizText{
		name:  name,
		text:  text,
		style: style,
		x:     x,
		y:     y,
		w:     w,
		h:     h,
	}
}

// Name xxx
func (t *VizText) Name() string {
	return t.name
}

// HandleInput xxx
func (t *VizText) HandleInput(mx, my float32, mdown bool) {
}

// Draw xxx
func (t *VizText) Draw(ctx *nanovgo.Context) {
	t.style.Do(ctx)
	ctx.SetFillColor(t.style.textColor)
	ctx.SetTextAlign(nanovgo.AlignLeft | nanovgo.AlignTop)
	ctx.Text(t.x, t.y, t.text)
}
