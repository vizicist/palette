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
	name      string
	isPressed bool
	text      string
	style     Style
	rect      image.Rectangle
	callback  VizButtonCallback
	waitForUp bool
}

// NewButton xxx
func (wind *VizWind) NewButton(name string, text string, pos image.Point, style Style, cb VizButtonCallback) *VizButton {
	if !strings.HasPrefix(name, "button.") {
		name = "button." + name
	}
	w := len(text) * wind.style.charWidth
	h := wind.style.lineHeight
	rect := image.Rect(pos.X, pos.Y, pos.X+w, pos.Y+h)
	return &VizButton{
		name:      name,
		text:      text,
		style:     style,
		isPressed: false,
		rect:      rect,
		callback:  cb,
		waitForUp: false,
	}
}

// SetWaitForUp xxx
func (b *VizButton) SetWaitForUp(waitForUp bool) {
	b.waitForUp = waitForUp
}

// Name xxx
func (b *VizButton) Name() string {
	return b.name
}

// HandleMouseInput xxx
func (b *VizButton) HandleMouseInput(mx, my int, mdown bool) {
	if false {
		log.Printf("HandleMouseInput %d,%d %v\n", mx, my, mdown)
	}
	/*
		focusObj := Wind[CurrentWindName].Focus()
		switch {
		case mdown == true:
			if b.isPressed && focusObj == b {
				// Mouse is already pressed and we have the focus
			}
			if mx >= b.rect.Min.X && mx <= b.rect.Max.X && my >= b.rect.Min.Y && my <= b.rect.Max.Y {
				// The mouse is inside the button
				if b.isPressed == false {
					Wind[CurrentWindName].SetFocus(b)
					b.isPressed = true
					if !b.waitForUp {
						b.callback("down")
					}
				}
			}
		case mdown == false:
			if b.isPressed == true {
				Wind[CurrentWindName].SetFocus(nil)
				b.isPressed = false
				if b.waitForUp {
					b.callback("up")
				}
			}
		}
	*/
}

// Draw xxx
func (b *VizButton) Draw(ctx *nanovgo.Context) {
	var cornerRadius float32 = 4.0

	/*
		bg := nanovgo.LinearGradient(b.x, b.y, b.x, b.y+b.h, nanovgo.RGBA(255, 255, 255, alpha), nanovgo.RGBA(0, 0, 0, alpha))
	*/
	b.style.Do(ctx)

	ctx.BeginPath()
	w := float32(b.rect.Max.X - b.rect.Min.X)
	h := float32(b.rect.Max.Y - b.rect.Min.Y)
	ctx.RoundedRect(float32(b.rect.Min.X+1), float32(b.rect.Min.Y+1), w-2, h-2, cornerRadius-1)
	ctx.Fill()

	ctx.BeginPath()
	ctx.RoundedRect(float32(b.rect.Min.X), float32(b.rect.Min.Y), w, h, cornerRadius-1)
	ctx.Stroke()

	ctx.SetTextAlign(nanovgo.AlignCenter | nanovgo.AlignMiddle)
	// Text uses the fill color, but we want it to be the strokeColor
	ctx.SetFillColor(b.style.textColor)
	pos := strings.Index(b.text, "\n")
	midx := float32((b.rect.Min.X + b.rect.Max.X) / 2)
	midy := float32((b.rect.Min.Y + b.rect.Max.Y) / 2)
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
