package gui

import (
	"image"
	"log"

	"github.com/micaelAlastor/nanovgo"
)

// VizContainer xxx
type VizContainer struct {
	VizObjData
	// Objects map[string]Obj
	hasBorder bool
}

// Rect xxx
func (wind *VizObjData) Rect() image.Rectangle {
	return wind.rect
}

/*
// Draw xxx
func (wind *VizObjData) Draw(ctx *nanovgo.Context) {
	for o := range wind.Objects {
		o.Draw(ctx)
	}
}
*/

// Objects xxx
func (o *VizContainer) Objects() map[string]VizObj {
	return o.objects
}

// Style xxx
func (o *VizContainer) Style() Style {
	return o.style
}

// HandleMouseInput xxx
func (o *VizContainer) HandleMouseInput(pos image.Point, down bool) {
	// log.Printf("VizObjData.HandleMouseInput: pos=%+v\n", pos)
	for _, o := range o.Objects() {
		if pos.In(o.Rect()) {
			o.HandleMouseInput(pos, down)
		}

	}
}

// AddObject xxx
func (o *VizContainer) AddObject(name string, newo VizObj) {
	objs := o.Objects()
	objs[name] = newo
}

// NewContainer xxx
func NewContainer(parent VizObj) *VizContainer {
	return &VizContainer{
		VizObjData: VizObjData{
			parent:  parent,
			style:   parent.Style(),
			rect:    image.Rectangle{},
			objects: map[string]VizObj{},
		},
		hasBorder: true,
	}
}

// Resize xxx
func (o *VizContainer) Resize(rect image.Rectangle) {
	o.style = o.style.SetSize(rect)
	o.rect = rect
	for _, oo := range o.Objects() {
		oo.Resize(rect)
	}
}

// Draw xxx
func (o *VizContainer) Draw(ctx *nanovgo.Context) {

	if o.hasBorder {
		ctx.Save()
		ctx.BeginPath()
		r := o.Rect()
		x := r.Min.X
		y := r.Min.Y
		w := r.Dx()
		h := r.Dy()
		cornerRadius := float32(4.0)
		ctx.RoundedRect(float32(x), float32(y), float32(w), float32(h), cornerRadius-0.5)
		ctx.SetStrokeWidth(3.0)
		ctx.Stroke()
		ctx.Restore()
	}

	/*
		ctx.SetTextLetterSpacing(0)
	*/

	for _, oo := range o.Objects() {
		oo.Draw(ctx)
	}
}

// AddObject xxx
func (wind *VizObjData) AddObject(name string, o VizObj) {
	_, ok := wind.objects[name]
	if ok {
		log.Printf("There's already an object named %s in that VizObjData\n", name)
	} else {
		wind.objects[name] = o
	}
}
