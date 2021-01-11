package gui

import (
	"image"
	"image/color"
	"log"

	"github.com/golang/freetype/truetype"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"

	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

var red = color.RGBA{255, 0, 0, 255}
var black = color.RGBA{0, 0, 0, 255}
var white = color.RGBA{255, 255, 255, 255}

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
	case "mono":
		f, err = truetype.Parse(gomono.TTF)
	case "regular":
		f, err = truetype.Parse(goregular.TTF)
	default:
		log.Printf("NewStyle: unrecognized fontname=%s, using regular\n", fontName)
		f, err = truetype.Parse(goregular.TTF)
	}
	if err != nil {
		log.Printf("truetype.Parse: unable to parse TTF for fontname=%s\n", fontName)
		return nil
	}
	face := truetype.NewFace(f, &truetype.Options{Size: float64(fontHeight)})
	fontRect := text.BoundString(face, "fghijklMNQ")
	realHeight := fontRect.Max.Y - fontRect.Min.Y

	return &Style{
		fontHeight:  realHeight,
		fontFace:    face,
		textColor:   black,
		strokeColor: black,
		fillColor:   white,
	}
}

// TextHeight xxx
func (style *Style) TextHeight() int {
	return style.fontHeight
}

// BoundString xxx
func (style *Style) BoundString(s string) image.Rectangle {
	return text.BoundString(style.fontFace, s)
}
