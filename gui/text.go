package gui

import (
	"image"
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

// DoSync xxx
func (st *ScrollingText) DoSync(from Window, cmd string, arg interface{}) (result interface{}, err error) {
	return NoSyncInterface("ScrollingText")
}

func (st *ScrollingText) resize(rect image.Rectangle) {

	// See how many lines we can fit in the rect
	nlines := rect.Dy() / st.Style.RowHeight()
	st.lines = make([]string, nlines)

	// Adjust the rect so we're exactly that height
	rect.Max.Y = rect.Min.Y + nlines*st.Style.RowHeight()

	st.Rect = rect
}

func (st *ScrollingText) redraw() {

	DoUpstream(st, "setcolor", foreColor)
	DoUpstream(st, "drawrect", st.Rect)

	textx := st.Rect.Min.X + 2

	for n, line := range st.lines {

		// assumes the area is erased, initially
		if line == "" {
			continue
		}
		texty := st.Rect.Min.Y + (n+1)*st.Style.RowHeight() - 4

		/*
			brect := st.Style.BoundString(line)
			bminy := brect.Min.Y
			bmaxy := brect.Max.Y
			if bminy < 0 {
				bmaxy -= bminy
				bminy = 0
			}
			texty += bminy
			texty += bmaxy
		*/
		DoUpstream(st, "drawtext", DrawTextCmd{line, st.Style.fontFace, image.Point{textx, texty}})
	}
}

// Do xxx
func (st *ScrollingText) Do(from Window, cmd string, arg interface{}) {

	switch cmd {
	case "resize":
		st.resize(ToRect(arg))
	case "redraw":
		st.redraw()
	case "clear":
		log.Printf("ScrollingText: Clear needs work!\n")
	case "mouse":
		// ignore
	case "addline":
		st.AddLine(ToString(arg))
	default:
		log.Printf("ScrollingText: didn't handle cmd=%s\n", cmd)
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
