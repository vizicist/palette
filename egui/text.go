package egui

import (
	"image"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2/text"
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
func (st *ScrollingText) Draw(screen *Screen) {

	color := color.RGBA{0xff, 0xff, 0, 0xff}
	screen.DrawRect(st.rect, color)

	texty := st.rect.Min.Y

	for _, line := range st.lines {

		textx := st.rect.Min.X

		brect := text.BoundString(st.style.fontFace, line)
		bminy := brect.Min.Y
		bmaxy := brect.Max.Y
		if bminy < 0 {
			bmaxy -= bminy
			bminy = 0
		}
		texty += bminy
		texty += bmaxy
		text.Draw(screen.eimage, line, st.style.fontFace, textx, texty, color)
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
