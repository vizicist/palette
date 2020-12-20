package gui

import (
	"strings"

	"github.com/micaelAlastor/nanovgo"
)

// VizButtonCallback xxx
type VizButtonCallback func(string)

// VizButton xxx
type VizButton struct {
	name      string
	isPressed bool
	text      string
	style     Style
	x, y      float32
	w, h      float32
	callback  VizButtonCallback
	waitForUp bool
}

// NewButton xxx
func NewButton(name string, text string, x, y, w, h float32, style Style, cb VizButtonCallback) *VizButton {
	return &VizButton{
		name:      name,
		text:      text,
		style:     style,
		isPressed: false,
		x:         x,
		y:         y,
		w:         w,
		h:         h,
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

// HandleInput xxx
func (b *VizButton) HandleInput(mx, my float32, mdown bool) {
	focusObj := Page[CurrentPageName].Focus()
	switch {
	case mdown == true:
		if b.isPressed && focusObj == b {
			// Mouse is already pressed and we have the focus
		}
		if mx >= b.x && mx <= (b.x+b.w) && my >= b.y && my <= (b.y+b.h) {
			// The mouse is inside the button
			if b.isPressed == false {
				Page[CurrentPageName].SetFocus(b)
				b.isPressed = true
				if !b.waitForUp {
					b.callback(b.name)
				}
			}
		}
	case mdown == false:
		if b.isPressed == true {
			Page[CurrentPageName].SetFocus(nil)
			b.isPressed = false
			if b.waitForUp {
				b.callback(b.name)
			}
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
	ctx.RoundedRect(b.x+1, b.y+1, b.w-2, b.h-2, cornerRadius-1)
	ctx.Fill()

	ctx.BeginPath()
	ctx.RoundedRect(b.x+0.5, b.y+0.5, b.w-1, b.h-1, cornerRadius-0.5)
	ctx.Stroke()

	ctx.SetTextAlign(nanovgo.AlignCenter | nanovgo.AlignMiddle)
	// Text uses the fill color, but we want it to be the strokeColor
	ctx.SetFillColor(b.style.textColor)
	pos := strings.Index(b.text, "\n")
	if pos >= 0 {
		// 2 lines
		line1 := b.text[:pos]
		line2 := b.text[pos+1:]
		halfHeight := b.style.lineHeight / 2.0
		midy := b.y + 0.5*b.h + 1
		ctx.Text(b.x+b.w*0.5, midy-halfHeight, line1)
		ctx.Text(b.x+b.w*0.5, midy+halfHeight, line2)
	} else {
		ctx.Text(b.x+b.w*0.5, b.y+0.5*b.h+1, b.text)
	}
}

func isBlack(col nanovgo.Color) bool {
	return col.R == 0.0 && col.G == 0.0 && col.B == 0.0 && col.A == 0.0
}
