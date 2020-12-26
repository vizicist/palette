package gui

import (
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
