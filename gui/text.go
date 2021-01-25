package gui

import (
	"image"
	"log"
)

// TextCallback xxx
type TextCallback func(updown string)

var defaultBufferSize = 4

// ScrollingText xxx
type ScrollingText struct {
	WindowData
	isPressed bool
	buffer    []string
	nlines    int
}

// NewScrollingText xxx
func NewScrollingText(parent Window) *ScrollingText {
	st := &ScrollingText{
		WindowData: NewWindowData(parent),
		isPressed:  false,
		buffer:     make([]string, defaultBufferSize),
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
	st.nlines = rect.Dy() / st.Style.RowHeight()
	// in case the buffer isn't big enough
	if st.nlines > len(st.buffer) {
		newbuffer := make([]string, st.nlines)
		copy(newbuffer, st.buffer)
		st.buffer = newbuffer
	}

	// Adjust the rect so we're exactly that height
	rect.Max.Y = rect.Min.Y + st.nlines*st.Style.RowHeight()

	st.Rect = rect
}

func (st *ScrollingText) redraw() {

	DoUpstream(st, "setcolor", foreColor)
	DoUpstream(st, "drawrect", st.Rect)

	textx := st.Rect.Min.X + 2

	for n := 0; n < st.nlines; n++ {
		// assumes the area is erased, so we don't erase blank lines
		line := st.buffer[n]
		if line == "" {
			continue
		}
		// rownum 0 is the bottom
		texty := st.Rect.Max.Y - n*st.Style.RowHeight() - 4
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
func (st *ScrollingText) AddLine(line string) {
	// prepend it to the beginning of st.lines
	st.buffer = append([]string{line}, st.buffer...)
}

// Clear clears all text
func (st *ScrollingText) Clear() {
	for n := 0; n < len(st.buffer); n++ {
		st.buffer[n] = ""
	}
}
