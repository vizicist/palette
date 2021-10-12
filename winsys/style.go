package winsys

import (
	"image"
	"image/color"
	"io/ioutil"
	"log"

	"github.com/golang/freetype/truetype"
	"github.com/vizicist/palette/engine"
	"golang.org/x/image/font"

	"golang.org/x/image/font/gofont/gomono"
)

// RedColor xxx
var RedColor = color.RGBA{255, 0, 0, 255}

// BackColor xxx
var BackColor = color.RGBA{0, 0, 0, 255}

// ForeColor xxx
var ForeColor = color.RGBA{255, 255, 255, 255}

// GreenColor xxx
var GreenColor = color.RGBA{0, 0xff, 0, 0xff}

// Style xxx
type Style struct {
	fontHeight  int
	charWidth   int
	textColor   color.RGBA
	strokeColor color.RGBA
	fillColor   color.RGBA
	fontFace    font.Face
}

// DefaultStyle xxx
func DefaultStyle() *Style {
	return NewStyle("fixed", 16)
}

// NewStyle xxx
func NewStyle(fontName string, fontHeight int) *Style {

	if fontHeight <= 0 {
		log.Printf("NewStyle: invalid fontHeight, using 12\n")
		fontHeight = 12
	}

	var f *truetype.Font
	var err error

	switch fontName {

	case "fixed":
		fontfile := engine.ConfigFilePath("consola.ttf")
		b, err := ioutil.ReadFile(fontfile)
		if err == nil {
			f, _ = truetype.Parse(b)
		}

	case "regular":
		// This font sucks
		// f, err = truetype.Parse(goregular.TTF)
		fontfile := engine.ConfigFilePath("times.ttf")
		b, err := ioutil.ReadFile(fontfile)
		if err == nil {
			f, _ = truetype.Parse(b)
		}

	default:
		log.Printf("NewStyle: unrecognized fontname=%s, using gomono\n", fontName)
	}

	if err != nil {
		log.Printf("NewStyle: unable to get font=%s, resorting to gomono, err=%s\n", fontName, err)
		f, _ = truetype.Parse(gomono.TTF) // last resort
	} else if f == nil {
		log.Printf("NewStyle: unable to get font=%s, f is nil?\n", fontName)
		f, _ = truetype.Parse(gomono.TTF) // last resort
	}
	face := truetype.NewFace(f, &truetype.Options{Size: float64(fontHeight)})
	fontRect := TextBoundString(face, "fghijklMNQ")
	charRect := TextBoundString(face, "Q")

	return &Style{
		fontHeight:  fontRect.Max.Y - fontRect.Min.Y,
		charWidth:   charRect.Max.X - charRect.Min.X,
		fontFace:    face,
		textColor:   ForeColor,
		strokeColor: ForeColor,
		fillColor:   BackColor,
	}
}

// TextFitRect xxx
func (style *Style) TextFitRect(s string) image.Rectangle {
	width := len(s) * style.CharWidth()
	height := style.TextHeight()
	pos := image.Point{0, 0}
	return image.Rectangle{Min: pos, Max: pos.Add(image.Point{width, height})}
}

// TextHeight xxx
func (style *Style) TextHeight() int {
	return style.fontHeight
}

// TextWidth xxx
func (style *Style) TextWidth(s string) int {
	return style.charWidth * len(s)
}

// CharWidth is a single character's width
func (style *Style) CharWidth() int {
	return style.charWidth
}

// RowHeight is the height of text rows (e.g. in ScrollingTextArea)
func (style *Style) RowHeight() int {
	return style.fontHeight + 2
}

// BoundString xxx
func (style *Style) BoundString(s string) image.Rectangle {
	return TextBoundString(style.fontFace, s)
}
