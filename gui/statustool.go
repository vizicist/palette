package gui

import (
	"image"
	"log"

	"github.com/micaelAlastor/nanovgo"
)

// VizStatus is a window that has a couple of buttons
type VizStatus struct {
	VizObjData
	b1 *VizButton
}

// NewStatus xxx
func NewStatus(name string) *VizStatus {
	status := &VizStatus{
		VizObjData: VizObjData{
			name:    name,
			style:   DefaultStyle,
			rect:    image.Rectangle{},
			objects: map[string]VizObj{},
		},
	}

	status.b1 = NewButton("status", "StatusLabel",
		func(updown string) {
			log.Printf("Status button: %s\n", updown)
		})

	AddObject(status.objects, status.b1)

	return status
}

// Name xxx
func (status *VizStatus) Name() string {
	return status.name
}

// Objects xxx
func (status *VizStatus) Objects() map[string]VizObj {
	return status.Objects()
}

/*
// Style xxx
func (status *VizStatus) Style() Style {
	return status.style
}
*/

// Resize xxx
func (status *VizStatus) Resize(r image.Rectangle) {
	// 24 lines of text
	h := int(r.Dy() / 24.0)
	if h < 0 {
		h = 1
	}
	status.style = status.style.SetFontSizeByHeight(h)
	status.rect = r

	b1r := r.Inset(5)
	status.b1.Resize(b1r)
}

// HandleMouseInput xxx
func (status *VizStatus) HandleMouseInput(pos image.Point, mdown bool) bool {
	o := ObjectUnder(status.objects, pos)
	handled := false
	if o != nil {
		o.HandleMouseInput(pos, mdown)
		handled = true
	} else {
		handled = false
	}
	return handled
}

// Draw xxx
func (status *VizStatus) Draw(ctx *nanovgo.Context) {

	var cornerRadius float32 = 4.0

	ctx.Save()
	defer ctx.Restore()

	status.style.Do(ctx)

	ctx.SetStrokeWidth(3.0)
	ctx.BeginPath()
	w := float32(status.Rect().Max.X - status.Rect().Min.X)
	h := float32(status.Rect().Max.Y - status.Rect().Min.Y)
	ctx.RoundedRect(float32(status.Rect().Min.X+1), float32(status.Rect().Min.Y+1), w-2, h-2, cornerRadius-1)
	ctx.Fill()

	ctx.BeginPath()
	ctx.RoundedRect(float32(status.Rect().Min.X), float32(status.Rect().Min.Y), w, h, cornerRadius-1)
	ctx.Stroke()

	status.b1.Draw(ctx)
}
