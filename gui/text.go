package gui

import (
	"fmt"
	"image"
	"log"
	"strings"
)

// TextCallback xxx
type TextCallback func(updown string)

var defaultBufferSize = 128

// ScrollingText assumes a fixed-width font
type ScrollingText struct {
	WindowData
	isPressed bool
	Buffer    []string
	nlines    int // number of lines actually displayed
	nchars    int // number of characters in a single line
}

// NewScrollingText xxx
func NewScrollingText(parent Window) *ScrollingText {
	st := &ScrollingText{
		WindowData: NewWindowData(parent),
		isPressed:  false,
		Buffer:     make([]string, defaultBufferSize),
	}
	return st
}

// Data xxx
func (st *ScrollingText) data() *WindowData {
	return &st.WindowData
}

func (st *ScrollingText) resize(rect image.Rectangle) {

	// See how many lines and chars we can fit in the rect
	st.nlines = rect.Dy() / st.style.RowHeight()
	st.nchars = rect.Dx() / st.style.CharWidth()

	// in case the buffer isn't big enough
	if st.nlines > len(st.Buffer) {
		newbuffer := make([]string, st.nlines)
		copy(newbuffer, st.Buffer)
		st.Buffer = newbuffer
	}
	// Adjust the rect so we're exactly that height
	rect.Max.Y = rect.Min.Y + st.nlines*st.style.RowHeight()

	st.Rect = rect
}

func (st *ScrollingText) redraw() {

	DoUpstream(st, "setcolor", foreColor)
	DoUpstream(st, "drawrect", st.Rect)

	if st.nchars == 0 || st.nlines == 0 {
		// window is too small
		return
	}

	textx := st.Rect.Min.X + 3

	// If a line is longer than st.nchars, we break it up.
	// The linestack we create will be at least st.nlines long,
	// but may be being longer, even though we only use st.nlines of them.
	linestack := make([]string, 0)

	for n := 0; n < st.nlines; n++ {

		// Lines in st.buffer are guaranteed to not have a trailing newline,
		// but they can have embedded newlines, so break them up.
		lines := strings.Split(st.Buffer[n], "\n")

		// go through them in reverse order
		for n := len(lines) - 1; n >= 0; n-- {
			line := lines[n]

			for len(line) > st.nchars {
				// Figure out where the last line split should be,
				// so we can pull off the last part.
				i := (len(line) / st.nchars) * st.nchars
				if i == len(line) { // it's an exact multiple of st.nchars
					i = i - st.nchars
				}
				linestack = append(linestack, line[i:])
				line = line[:i]
			}
			linestack = append(linestack, line)
		}
	}

	for n := 0; n < st.nlines; n++ {
		line := linestack[n]
		if line != "" {
			// rownum 0 is the bottom
			texty := st.Rect.Max.Y - n*st.style.RowHeight() - 4
			DoUpstream(st, "drawtext", DrawTextCmd{line, st.style.fontFace, image.Point{textx, texty}})
		}
	}
}

// Do xxx
func (st *ScrollingText) Do(from Window, cmd string, arg interface{}) (interface{}, error) {

	switch cmd {
	case "resize":
		st.resize(ToRect(arg))
	case "redraw":
		st.redraw()
	case "dumpstate":
		in := strings.Repeat(" ", 12)
		s := fmt.Sprintf("{\n%s\"buffer\": [\n", in)
		sep := ""
		for _, line := range st.Buffer {
			s += fmt.Sprintf("%s%s%s\"%s\"", sep, in, in, line)
			sep = ",\n"
		}
		s += fmt.Sprintf("\n%s]\n}", in)
		return s, nil

	case "clear":
		log.Printf("ScrollingText: Clear needs work!\n")
	case "mouse":
		// ignore
	case "addline":
		st.AddLine(ToString(arg))
	default:
		log.Printf("ScrollingText: didn't handle cmd=%s\n", cmd)
	}
	return nil, nil
}

// AddLine xxx
func (st *ScrollingText) AddLine(line string) {
	// remove final newline if there is one, but leave interior ones
	if strings.HasSuffix(line, "\n") {
		line = line[:len(line)-1]
	}
	// prepend it to the beginning of st.lines
	st.Buffer = append([]string{line}, st.Buffer...)
}

// Clear clears all text
func (st *ScrollingText) Clear() {
	for n := 0; n < len(st.Buffer); n++ {
		st.Buffer[n] = ""
	}
}
