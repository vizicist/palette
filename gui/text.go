package gui

import (
	"image"
	"image/color"
	"log"
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
func NewScrollingText(style *Style) *ScrollingText {
	return &ScrollingText{
		WindowData: WindowData{
			style:   style,
			rect:    image.Rectangle{},
			objects: map[string]Window{},
		},
		isPressed: false,
		nlines:    0,
		lines:     make([]string, 0),
	}
}

// Data xxx
func (st *ScrollingText) Data() WindowData {
	return st.WindowData
}

// Resize xxx
func (st *ScrollingText) Resize(rect image.Rectangle) {

	textHeight := st.style.TextHeight()
	// See how many lines we can fit in the rect
	st.nlines = rect.Dy() / textHeight
	st.lines = make([]string, st.nlines)

	// Adjust the rect so we're exactly that height
	rect.Max.Y = rect.Min.Y + st.nlines*textHeight

	// desiredHeight := rect.Dy() / st.nlines
	// if textHeight != st.style.fontHeight {
	// 	st.style = NewStyle("mono", desiredHeight)
	// }

	st.rect = rect
}

// Draw xxx
func (st *ScrollingText) Draw(screen *Screen) {

	color := color.RGBA{0xff, 0xff, 0, 0xff}
	screen.DrawRect(st.rect, color)

	textHeight := st.style.TextHeight()

	textx := st.rect.Min.X
	for n, line := range st.lines {

		texty := st.rect.Min.Y + n*textHeight

		brect := st.style.BoundString(line)
		bminy := brect.Min.Y
		bmaxy := brect.Max.Y
		if bminy < 0 {
			bmaxy -= bminy
			bminy = 0
		}
		texty += bminy
		texty += bmaxy
		screen.drawText(line, st.style.fontFace, textx, texty, color)
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
