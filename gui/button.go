package gui

import (
	"image"
	"log"
	"strings"

	"github.com/micaelAlastor/nanovgo"
)

// VizButtonCallback xxx
type VizButtonCallback func(updown string)

// VizButton xxx
type VizButton struct {
	VizWind
	name      string
	isPressed bool
	text      string
	style     Style
	// Rect      image.Rectangle
	callback VizButtonCallback
}

// Name xxx
func (b *VizButton) Name() string {
	return b.name
}

// HandleMouseInput xxx
func (b *VizButton) HandleMouseInput(pos image.Point, mdown bool) {
	log.Printf("VizButton.HandleMouseInput, b.isPressed=%v\n", b.isPressed)
	switch mdown {
	case true:
		// The mouse is inside the button
		if b.isPressed == false {
			b.isPressed = true
			b.callback("down")
		}
	case false:
		if b.isPressed == true {
			b.isPressed = false
			b.callback("up")
		}
	}
}

// Draw xxx
func (b *VizButton) Draw(ctx *nanovgo.Context) {
	var cornerRadius float32 = 4.0

	/*
		bg := nanovgo.LinearGradient(b.x, b.y, b.x, b.y+b.h, nanovgo.RGBA(255, 255, 255, alpha), nanovgo.RGBA(0, 0, 0, alpha))
	*/
	b.style.Do(ctx)

	ctx.BeginPath()
	w := float32(b.Rect().Max.X - b.Rect().Min.X)
	h := float32(b.Rect().Max.Y - b.Rect().Min.Y)
	ctx.RoundedRect(float32(b.Rect().Min.X+1), float32(b.Rect().Min.Y+1), w-2, h-2, cornerRadius-1)
	ctx.Fill()

	ctx.BeginPath()
	ctx.RoundedRect(float32(b.Rect().Min.X), float32(b.Rect().Min.Y), w, h, cornerRadius-1)
	ctx.Stroke()

	ctx.SetTextAlign(nanovgo.AlignCenter | nanovgo.AlignMiddle)
	// Text uses the fill color, but we want it to be the strokeColor
	ctx.SetFillColor(b.style.textColor)
	pos := strings.Index(b.text, "\n")
	midx := float32((b.Rect().Min.X + b.Rect().Max.X) / 2)
	midy := float32((b.Rect().Min.Y + b.Rect().Max.Y) / 2)
	if pos >= 0 {
		// 2 lines
		line1 := b.text[:pos]
		line2 := b.text[pos+1:]
		halfHeight := float32(b.style.lineHeight) / 2.0
		ctx.Text(midx, midy-halfHeight, line1)
		ctx.Text(midx, midy+halfHeight, line2)
	} else {
		ctx.Text(midx, midy, b.text)
	}
}

func isBlack(col nanovgo.Color) bool {
	return col.R == 0.0 && col.G == 0.0 && col.B == 0.0 && col.A == 0.0
}
