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
	lines     []string
}

// NewScrollingText xxx
func NewScrollingText(parent Window) *ScrollingText {
	return &ScrollingText{
		WindowData: WindowData{
			screen:  parent.Data().screen,
			style:   parent.Data().style,
			rect:    image.Rectangle{},
			objects: map[string]Window{},
		},
		isPressed: false,
		lines:     make([]string, 0),
	}
}

// Data xxx
func (st *ScrollingText) Data() WindowData {
	return st.WindowData
}

// Resize xxx
func (st *ScrollingText) Resize(rect image.Rectangle) image.Rectangle {

	textHeight := st.style.TextHeight()
	// See how many lines we can fit in the rect
	nlines := rect.Dy() / textHeight
	st.lines = make([]string, nlines)

	// Adjust the rect so we're exactly that height
	rect.Max.Y = rect.Min.Y + nlines*textHeight

	st.rect = rect
	return st.rect
}

// Draw xxx
func (st *ScrollingText) Draw() {

	color := color.RGBA{0xff, 0xff, 0, 0xff}
	st.screen.drawRect(st.rect, color)

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
		st.screen.drawText(line, st.style.fontFace, textx, texty, color)
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
	for n := 1; n < len(st.lines); n++ {
		st.lines[n-1] = st.lines[n]
	}
	st.lines[len(st.lines)-1] = newline
}

func (st *ScrollingText) clear() {
	for n := 0; n < len(st.lines); n++ {
		st.lines[n] = ""
	}
}
