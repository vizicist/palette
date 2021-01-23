package gui

import (
	"image"
	"image/color"
	"io/ioutil"
	"log"

	"github.com/golang/freetype/truetype"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"

	"github.com/vizicist/palette/engine"
	"golang.org/x/image/font/gofont/gomono"
)

var red = color.RGBA{255, 0, 0, 255}
var black = color.RGBA{0, 0, 0, 255}
var white = color.RGBA{255, 255, 255, 255}
var green = color.RGBA{0, 0xff, 0, 0xff}

// Style xxx
type Style struct {
	fontHeight  int
	textColor   color.RGBA
	strokeColor color.RGBA
	fillColor   color.RGBA
	fontFace    font.Face
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
			f, err = truetype.Parse(b)
		}

	case "regular":
		// This font sucks
		// f, err = truetype.Parse(goregular.TTF)
		fontfile := engine.ConfigFilePath("times.ttf")
		b, err := ioutil.ReadFile(fontfile)
		if err == nil {
			f, err = truetype.Parse(b)
		}

	default:
		log.Printf("NewStyle: unrecognized fontname=%s, using gomono\n", fontName)
	}

	if err != nil {
		log.Printf("NewStyle: unable to get font=%s, resorting to gomono\n", fontName)
		f, _ = truetype.Parse(gomono.TTF) // last resort
	}
	face := truetype.NewFace(f, &truetype.Options{Size: float64(fontHeight)})
	fontRect := text.BoundString(face, "fghijklMNQ")
	realHeight := fontRect.Max.Y - fontRect.Min.Y

	return &Style{
		fontHeight:  realHeight,
		fontFace:    face,
		textColor:   white,
		strokeColor: white,
		fillColor:   black,
	}
}

// TextHeight xxx
func (style *Style) TextHeight() int {
	return style.fontHeight
}

// RowHeight xxx
func (style *Style) RowHeight() int {
	return style.fontHeight + 2
}

// BoundString xxx
func (style *Style) BoundString(s string) image.Rectangle {
	return text.BoundString(style.fontFace, s)
}
