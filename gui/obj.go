package gui

import (
	"image"

	"github.com/micaelAlastor/nanovgo"
)

// VizObj xxx
type VizObj interface {
	HandleMouseInput(pos image.Point, down bool) bool
	Draw(ctx *nanovgo.Context)
	Rect() image.Rectangle
	// Style() Style
	Name() string
	Resize(image.Rectangle)
	Objects() map[string]VizObj
}

// VizObjData xxx
type VizObjData struct {
	name    string
	style   Style
	rect    image.Rectangle
	objects map[string]VizObj
}

// ObjectUnder xxx
func ObjectUnder(objects map[string]VizObj, pos image.Point) VizObj {
	for _, o := range objects {
		if pos.In(o.Rect()) {
			return o
		}
	}
	return nil
}
