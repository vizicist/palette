package gui

import (
	"image/color"
	"log"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

var red = color.RGBA{255, 0, 0, 255}
var black = color.RGBA{0, 0, 0, 255}
var white = color.RGBA{255, 255, 255, 255}

// Style xxx
type Style struct {
	fontSize    float32
	fontFace    font.Face
	textColor   color.RGBA
	strokeColor color.RGBA
	fillColor   color.RGBA
	charWidth   int
	lineHeight  int
}

// DefaultStyle is for initializing Style values
var DefaultStyle Style = Style{
	fontSize:    12.0,
	fontFace:    basicfont.Face7x13,
	textColor:   black,
	strokeColor: black,
	fillColor:   white,
	charWidth:   0, // filled in by SetSize
	lineHeight:  0, // filled in by SetSize
}

// Do xxx
func (style Style) Do(ctx *gg.Context) {

	ctx.SetFillStyle(gg.NewSolidPattern(style.fillColor))
	ctx.SetStrokeStyle(gg.NewSolidPattern(style.strokeColor))

	ctx.SetFontFace(style.fontFace)
	// ctx.SetFontSize(style.fontSize)
}

// SetFontSizeByHeight xxx
func (style Style) SetFontSizeByHeight(height int) Style {

	if height <= 0 {
		log.Printf("Style.Resize: bad height = %d\n", height)
		return style
	}
	fh := float32(height)
	style.fontSize = fh
	style.lineHeight = int(fh * 1.5)
	style.charWidth = int(fh / 1.5)
	return style
}
