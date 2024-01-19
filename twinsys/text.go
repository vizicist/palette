package twinsys

import (
	"encoding/json"
	"fmt"
	"image"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
	"github.com/vizicist/palette/kit"
)

// TextCallback xxx
type TextCallback func(updown string)

var defaultBufferSize = 128

// ScrollingText assumes a fixed-width font
type ScrollingText struct {
	ctx       WinContext
	isPressed bool
	Buffer    []string
	nlines    int // number of lines actually displayed
	nchars    int // number of characters in a single line
}

// NewScrollingText xxx
func NewScrollingText(parent Window) WindowData {
	st := &ScrollingText{
		ctx:       NewWindowContext(parent),
		isPressed: false,
		Buffer:    make([]string, defaultBufferSize),
	}
	return NewToolData(st, "ScrollingText", image.Point{})
}

// Context xxx
func (st *ScrollingText) Context() *WinContext {
	return &st.ctx
}

func (st *ScrollingText) resize(size image.Point) {

	styleInfo := WinStyleInfo(st)
	// See how many lines and chars we can fit in the rect
	st.nlines = size.Y / styleInfo.RowHeight()
	st.nchars = size.X / styleInfo.CharWidth()

	// in case the buffer isn't big enough
	if st.nlines > len(st.Buffer) {
		newbuffer := make([]string, st.nlines)
		copy(newbuffer, st.Buffer)
		st.Buffer = newbuffer
	}
	// Adjust our size so we're exactly that height
	WinSetSize(st, image.Point{size.X, st.nlines * styleInfo.RowHeight()})
}

func (st *ScrollingText) redraw() {

	sz := WinGetSize(st)
	rect := image.Rect(0, 0, sz.X, sz.Y)
	styleInfo := WinStyleInfo(st)
	styleName := WinStyleName(st)

	WinDoUpstream(st, NewSetColorCmd(ForeColor))
	WinDoUpstream(st, NewDrawRectCmd(rect))

	if st.nchars == 0 || st.nlines == 0 {
		// window is too small
		return
	}

	textx := rect.Min.X + 3

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
			texty := rect.Max.Y - n*styleInfo.RowHeight() - 4
			WinDoUpstream(st, NewDrawTextCmd(line, styleName, image.Point{textx, texty}))
		}
	}
}

func jsonEscape(i string) string {
	b, err := json.Marshal(i)
	if err != nil {
		return "jsonEscape is unable to Marshal that string"
	}
	// Trim the beginning and trailing " character
	return string(b[1 : len(b)-1])
}

// Do xxx
func (st *ScrollingText) Do(cmd kit.Cmd) string {

	switch cmd.Subj {

	case "resize":
		size := cmd.ValuesXY("size", kit.PointZero)
		st.resize(size)

	case "redraw":
		st.redraw()

	case "restore":
		state := cmd.ValuesString("state", "")
		st.Buffer = make([]string, 0)
		st.Buffer = append(st.Buffer, strings.Split(strings.TrimSuffix(state, "\n"), "\n")...)

	case "getstate":
		state := strings.Repeat(" ", 12)
		for _, line := range st.Buffer {
			state += jsonEscape(line)
		}
		return fmt.Sprintf("\"state\":\"%s\"", state)

	case "clear":
		for n := range st.Buffer {
			st.Buffer[n] = ""
		}

	case "mouse":
		// do nothing

	case "addline":
		line := cmd.ValuesString("line", "")
		st.AddLine(line)

	default:
		kit.LogWarn("ScrollingText: didn't handle", "subj", cmd.Subj)
	}
	return kit.OkResult()
}

// AddLine xxx
func (st *ScrollingText) AddLine(line string) {
	// remove final newline if there is one, but leave interior ones
	line = strings.TrimSuffix(line, "\n")

	// prepend it to the beginning of st.lines
	st.Buffer = append([]string{line}, st.Buffer...)
}

// Clear clears all text
func (st *ScrollingText) Clear() {
	for n := 0; n < len(st.Buffer); n++ {
		st.Buffer[n] = ""
	}
}
// BoundString returns the measured size of a given string using a given font.
// This method will return the exact size in pixels that a string drawn by Draw will be.
// The bound's origin point indicates the dot (period) position.
// This means that if the text consists of one character '.', this dot is rendered at (0, 0).
//
// BoundString behaves almost exactly like golang.org/x/image/font's BoundString,
// but newline characters '\n' in the input string move the text position to the following line.
//
// face is the font for text rendering.
// text is the string that's being measured.
//
// Be careful that the passed font face is held by this package and is never released.
// This is a known issue (#498).
//
// BoundString is concurrent-safe.
func BoundString(face font.Face, text string) image.Rectangle {
	// textM.Lock()
	// defer textM.Unlock()

	m := face.Metrics()
	faceHeight := m.Height

	fx, fy := fixed.I(0), fixed.I(0)
	prevR := rune(-1)

	var bounds fixed.Rectangle26_6
	for _, r := range text {
		if prevR >= 0 {
			fx += face.Kern(prevR, r)
		}
		if r == '\n' {
			fx = fixed.I(0)
			fy += faceHeight
			prevR = rune(-1)
			continue
		}

		b := getGlyphBounds(face, r)
		b.Min.X += fx
		b.Max.X += fx
		b.Min.Y += fy
		b.Max.Y += fy
		bounds = bounds.Union(b)

		fx += glyphAdvance(face, r)
		prevR = r
	}

	return image.Rect(
		bounds.Min.X.Floor(),
		bounds.Min.Y.Floor(),
		bounds.Max.X.Ceil(),
		bounds.Max.Y.Ceil(),
	)
}

var (
	glyphBoundsCache = map[font.Face]map[rune]fixed.Rectangle26_6{}
)

func getGlyphBounds(face font.Face, r rune) fixed.Rectangle26_6 {
	if _, ok := glyphBoundsCache[face]; !ok {
		glyphBoundsCache[face] = map[rune]fixed.Rectangle26_6{}
	}
	if b, ok := glyphBoundsCache[face][r]; ok {
		return b
	}
	b, _, _ := face.GlyphBounds(r)
	glyphBoundsCache[face][r] = b
	return b
}

var glyphAdvanceCache = map[font.Face]map[rune]fixed.Int26_6{}

func glyphAdvance(face font.Face, r rune) fixed.Int26_6 {
	m, ok := glyphAdvanceCache[face]
	if !ok {
		m = map[rune]fixed.Int26_6{}
		glyphAdvanceCache[face] = m
	}

	a, ok := m[r]
	if !ok {
		a, _ = face.GlyphAdvance(r)
		m[r] = a
	}

	return a
}
