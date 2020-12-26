package gui

import (
	"image"

	"github.com/micaelAlastor/nanovgo"
)

// VizObj xxx
type VizObj interface {
	HandleMouseInput(pos image.Point, down bool)
	Draw(ctx *nanovgo.Context)
	Rect() image.Rectangle
	Style() Style
	Resize(image.Rectangle)
	Objects() map[string]VizObj
	AddObject(name string, o VizObj)
}

// VizObjData xxx
type VizObjData struct {
	parent  VizObj
	style   Style
	rect    image.Rectangle
	objects map[string]VizObj
}
