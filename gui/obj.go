package gui

import (
	"image"

	"github.com/micaelAlastor/nanovgo"
)

// Obj xxx
type Obj interface {
	HandleMouseInput(pos image.Point, down bool)
	Draw(ctx *nanovgo.Context)
	Name() string
	Rect() image.Rectangle
}
