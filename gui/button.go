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
	VizObjData
	name      string
	isPressed bool
	text      string
	// Rect      image.Rectangle
	callback VizButtonCallback
}

// NewButton xxx
func NewButton(name string, text string, cb VizButtonCallback) *VizButton {
	return &VizButton{
		VizObjData: VizObjData{
			name:    name,
			style:   DefaultStyle,
			rect:    image.Rectangle{},
			objects: map[string]VizObj{},
		},
		name:      "",
		isPressed: false,
		text:      text,
		callback:  cb,
	}
}

// Name xxx
func (b *VizButton) Name() string {
	return b.name
}

// Objects xxx
func (b *VizButton) Objects() map[string]VizObj {
	return b.Objects()
}

/*
// Style xxx
func (b *VizButton) Style() Style {
	return b.style
}
*/

// Resize xxx
func (b *VizButton) Resize(r image.Rectangle) {
	b.style = b.style.SetSize(r)
	w := len(b.text) * b.style.charWidth
	h := b.style.lineHeight
	nr := image.Rect(r.Min.X, r.Min.Y, r.Min.X+w, r.Min.Y+h)
	b.VizObjData.rect = nr
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

	ctx.Save()
	defer ctx.Restore()

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
