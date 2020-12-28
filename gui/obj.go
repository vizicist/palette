package gui

import (
	"image"
	"log"

	"github.com/micaelAlastor/nanovgo"
)

// Window xxx
type Window interface {
	HandleMouseInput(pos image.Point, button int, down bool) bool
	Draw(ctx *nanovgo.Context)
	Rect() image.Rectangle
	// Style() Style
	Name() string
	Resize(image.Rectangle)
	Objects() map[string]Window
}

// WindowData xxx
type WindowData struct {
	name    string
	style   Style
	rect    image.Rectangle
	objects map[string]Window
}

// ObjectUnder xxx
func ObjectUnder(objects map[string]Window, pos image.Point) Window {
	for _, o := range objects {
		if pos.In(o.Rect()) {
			return o
		}
	}
	return nil
}

// AddObject xxx
func AddObject(objects map[string]Window, o Window) {
	name := o.Name()
	_, ok := objects[name]
	if ok {
		log.Printf("There's already an object named %s in that WindowData\n", name)
	} else {
		objects[name] = o
	}
}
