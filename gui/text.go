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
	rowHeight int
}

// NewScrollingText xxx
func NewScrollingText(parent Window) *ScrollingText {
	st := &ScrollingText{
		WindowData: NewWindowData(parent),
		isPressed:  false,
		lines:      make([]string, 0),
	}
	go st.Run()
	return st
}

// Data xxx
func (st *ScrollingText) Data() *WindowData {
	return &st.WindowData
}

// Resize xxx
func (st *ScrollingText) Resize(rect image.Rectangle) image.Rectangle {

	st.rowHeight = st.Style.TextHeight()
	// See how many lines we can fit in the rect
	nlines := rect.Dy() / st.rowHeight
	st.lines = make([]string, nlines)

	// Adjust the rect so we're exactly that height
	rect.Max.Y = rect.Min.Y + nlines*st.rowHeight

	st.Rect = rect
	return st.Rect
}

// Draw xxx
func (st *ScrollingText) Draw() {

	color := color.RGBA{0xff, 0xff, 0, 0xff}
	st.Screen.DrawRect(st.Rect, white)

	textHeight := st.Style.TextHeight()

	textx := st.Rect.Min.X + 2
	for n, line := range st.lines {

		texty := st.Rect.Min.Y + n*textHeight

		brect := st.Style.BoundString(line)
		bminy := brect.Min.Y
		bmaxy := brect.Max.Y
		if bminy < 0 {
			bmaxy -= bminy
			bminy = 0
		}
		texty += bminy
		texty += bmaxy
		st.Screen.drawText(line, st.Style.fontFace, textx, texty, color)
	}
}

// Run xxx
func (st *ScrollingText) Run() {
	for {
		me := <-st.MouseChan
		st.Screen.Log("ScrollingText.Run: me=%v", me)
		if !me.Pos.In(st.Rect) {
			log.Printf("Text.HandleMouseInput: pos not in rect!\n")
			continue
		}
	}
}

// AddLine xxx
func (st *ScrollingText) AddLine(newline string) {
	// XXX - this can be done better
	for n := 1; n < len(st.lines); n++ {
		st.lines[n-1] = st.lines[n]
	}
	st.lines[len(st.lines)-1] = newline
}

// Clear clears all text
func (st *ScrollingText) Clear() {
	for n := 0; n < len(st.lines); n++ {
		st.lines[n] = ""
	}
}
