package gui

import "github.com/micaelAlastor/nanovgo"

// Obj xxx
type Obj interface {
	HandleInput(mx, my float32, mdown bool)
	Draw(ctx *nanovgo.Context)
	Name() string
}
