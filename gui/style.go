package gui

import (
	"image"
	"log"

	"github.com/micaelAlastor/nanovgo"
)

var red = nanovgo.RGBA(255, 0, 0, 255)
var black = nanovgo.RGBA(0, 0, 0, 255)
var white = nanovgo.RGBA(255, 255, 255, 255)

// Style xxx
type Style struct {
	fontSize    float32
	fontFace    string
	textColor   nanovgo.Color
	strokeColor nanovgo.Color
	fillColor   nanovgo.Color
	charWidth   int
	lineHeight  int
}

// DefaultStyle is for initializing Style values
var DefaultStyle Style = Style{
	fontSize:    12.0,
	fontFace:    "lucida",
	textColor:   black,
	strokeColor: black,
	fillColor:   white,
	charWidth:   0, // filled in by SetSize
	lineHeight:  0, // filled in by SetSize
}

// Do xxx
func (style Style) Do(ctx *nanovgo.Context) {
	ctx.SetFillColor(style.fillColor)
	ctx.SetStrokeColor(style.strokeColor)
	ctx.SetFontFace(style.fontFace)
	ctx.SetFontSize(style.fontSize)
}

// SetSize xxx
func (style Style) SetSize(rect image.Rectangle) Style {

	w := rect.Max.X
	h := rect.Max.Y
	if w <= 0 || h <= 0 {
		log.Printf("Style.Resize: bad dimensions? %d,%d\n", w, h)
		return style
	}
	style.lineHeight = h / 48.0
	style.charWidth = w / 80.0
	return style
}
