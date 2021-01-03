package egui

import (
	"image"
	"log"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
)

// TextCallback xxx
type TextCallback func(updown string)

// ScrollingText xxx
type ScrollingText struct {
	WindowData
	isPressed bool
	nlines    int
	lines     []string
}

// NewScrollingText xxx
func NewScrollingText(style *Style, nlines int) *ScrollingText {
	if nlines <= 0 {
		log.Printf("NewScrollingText: invalid nlines, assuming 1\n")
		nlines = 1
	}
	return &ScrollingText{
		WindowData: WindowData{
			style:   style,
			rect:    image.Rectangle{},
			objects: map[string]Window{},
		},
		isPressed: false,
		nlines:    nlines,
		lines:     make([]string, nlines),
	}
}

// Data xxx
func (st *ScrollingText) Data() WindowData {
	return st.WindowData
}

// Resize xxx
func (st *ScrollingText) Resize(rect image.Rectangle) {

	desiredHeight := rect.Dy() / st.nlines

	if desiredHeight != st.style.fontHeight {
		st.style = NewStyle("mono", desiredHeight)
	}

	st.rect = rect
}

// Draw xxx
func (st *ScrollingText) Draw(ctx *gg.Context) {

	var cornerRadius float64 = 4.0
	st.style.SetForDrawing(ctx)

	// interior
	w := float64(st.rect.Max.X - st.rect.Min.X)
	h := float64(st.rect.Max.Y - st.rect.Min.Y)
	ctx.DrawRoundedRectangle(float64(st.rect.Min.X+1), float64(st.rect.Min.Y+1), w-2, h-2, cornerRadius-1)
	ctx.Fill()

	// outline
	ctx.DrawRoundedRectangle(float64(st.rect.Min.X), float64(st.rect.Min.Y), w, h, cornerRadius-1)
	ctx.Stroke()

	// text lines
	st.style.SetForText(ctx)

	texty := float64(st.rect.Min.Y)

	for _, line := range st.lines {

		textx := float64(st.rect.Min.X)

		brect, _ := font.BoundString(st.style.fontFace, line)
		bminy := brect.Min.Y.Round()
		bmaxy := brect.Max.Y.Round()
		if bminy < 0 {
			bmaxy -= bminy
			bminy = 0
		}
		texty += float64(bminy)
		texty += float64(bmaxy)
		ctx.DrawString(line, textx, texty)

		// texty := float64(b.rect.Min.Y)
	}
}

// HandleMouseInput xxx
func (st *ScrollingText) HandleMouseInput(pos image.Point, button int, mdown bool) bool {
	if !pos.In(st.rect) {
		log.Printf("Text.HandleMouseInput: pos not in rect!\n")
		return false
	}
	// Should it be highlighting for copying ?
	return true
}

// AddLine xxx
func (st *ScrollingText) AddLine(newline string) {
	// XXX - this can be done better
	for n := 1; n < st.nlines; n++ {
		st.lines[n-1] = st.lines[n]
	}
	st.lines[st.nlines-1] = newline
}
