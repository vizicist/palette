package winsys

import (
	"image"
	"image/color"
	"log"
	"os"

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

// StyleInfo xxx
type StyleInfo struct {
	fontHeight  int
	charWidth   int
	textColor   color.RGBA
	strokeColor color.RGBA
	fillColor   color.RGBA
	fontFace    font.Face
}

// DefaultStyle xxx
func DefaultStyleName() string {
	return "fixed"
}

// NewStyle xxx
func NewStyle(styleName string, fontHeight int) *StyleInfo {

	if fontHeight <= 0 {
		log.Printf("NewStyle: invalid fontHeight, using 12\n")
		fontHeight = 12
	}

	var f *truetype.Font
	var err error

	switch styleName {

	case "fixed":
		fontfile := engine.ConfigFilePath("consola.ttf")
		b, err := os.ReadFile(fontfile)
		if err == nil {
			f, _ = truetype.Parse(b)
		}

	case "regular":
		// This font sucks
		// f, err = truetype.Parse(goregular.TTF)
		fontfile := engine.ConfigFilePath("times.ttf")
		b, err := os.ReadFile(fontfile)
		if err == nil {
			f, _ = truetype.Parse(b)
		}

	default:
		log.Printf("NewStyle: unrecognized fontname=%s, using gomono\n", styleName)
	}

	if err != nil {
		log.Printf("NewStyle: unable to get font=%s, resorting to gomono, err=%s\n", styleName, err)
		f, _ = truetype.Parse(gomono.TTF) // last resort
	} else if f == nil {
		log.Printf("NewStyle: unable to get font=%s, f is nil?\n", styleName)
		f, _ = truetype.Parse(gomono.TTF) // last resort
	}
	face := truetype.NewFace(f, &truetype.Options{Size: float64(fontHeight)})
	fontRect := TextBoundString(face, "fghijklMNQ")
	charRect := TextBoundString(face, "Q")

	return &StyleInfo{
		fontHeight:  fontRect.Max.Y - fontRect.Min.Y,
		charWidth:   charRect.Max.X - charRect.Min.X,
		fontFace:    face,
		textColor:   ForeColor,
		strokeColor: ForeColor,
		fillColor:   BackColor,
	}
}

// TextFitRect xxx
func (style *StyleInfo) TextFitRect(s string) image.Rectangle {
	width := len(s) * style.CharWidth()
	height := style.TextHeight()
	pos := image.Point{0, 0}
	return image.Rectangle{Min: pos, Max: pos.Add(image.Point{width, height})}
}

// TextHeight xxx
func (style *StyleInfo) TextHeight() int {
	return style.fontHeight
}

// TextWidth xxx
func (style *StyleInfo) TextWidth(s string) int {
	return style.charWidth * len(s)
}

// CharWidth is a single character's width
func (style *StyleInfo) CharWidth() int {
	return style.charWidth
}

// RowHeight is the height of text rows (e.g. in ScrollingTextArea)
func (style *StyleInfo) RowHeight() int {
	return style.fontHeight + 2
}

// BoundString xxx
func (style *StyleInfo) BoundString(s string) image.Rectangle {
	return TextBoundString(style.fontFace, s)
}
