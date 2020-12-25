package gui

import "github.com/micaelAlastor/nanovgo"

// Obj xxx
type Obj interface {
	HandleMouseInput(mx, my int, mdown bool)
	Draw(ctx *nanovgo.Context)
	Name() string
}
