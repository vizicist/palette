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
func NewScrollingText(parent Window) Window {
	st := &ScrollingText{
		WindowData: NewWindowData(parent),
		isPressed:  false,
		lines:      make([]string, 0),
	}
	return st
}

// Data xxx
func (st *ScrollingText) Data() *WindowData {
	return &st.WindowData
}

// DoUpstream xxx
func (st *ScrollingText) DoUpstream(w Window, cmd UpstreamCmd) {
	st.parent.DoUpstream(st, cmd)
}

func (st *ScrollingText) resize(rect image.Rectangle) image.Rectangle {

	st.rowHeight = st.Style.TextHeight()
	// See how many lines we can fit in the rect
	nlines := rect.Dy() / st.rowHeight
	st.lines = make([]string, nlines)

	// Adjust the rect so we're exactly that height
	rect.Max.Y = rect.Min.Y + nlines*st.rowHeight

	st.Rect = rect
	return st.Rect
}

func (st *ScrollingText) redraw() {

	clr := color.RGBA{0xff, 0xff, 0, 0xff}

	st.parent.DoUpstream(st, DrawRectCmd{st.Rect, white})

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
		st.parent.DoUpstream(st, DrawTextCmd{line, st.Style.fontFace, image.Point{textx, texty}, clr})
	}
}

// DoDownstream xxx
func (st *ScrollingText) DoDownstream(c DownstreamCmd) {

	switch cmd := c.(type) {
	case ResizeCmd:
		st.Rect = cmd.Rect
	case RedrawCmd:
		st.redraw()
	case ClearCmd:
		log.Printf("ScrollingText: Clear\n")
	case MouseCmd:
	default:
		log.Printf("ScrollingText: didn't handle cmd=%v\n", cmd)
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
