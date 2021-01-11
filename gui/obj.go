package gui

import (
	"image"
	"log"
)

// Window xxx
type Window interface {
	HandleMouseInput(pos image.Point, button int, me MouseEvent) bool
	Draw()
	Resize(image.Rectangle) image.Rectangle
	Data() WindowData
}

// WindowData xxx
type WindowData struct {
	screen  *Screen
	style   *Style
	rect    image.Rectangle // in Screen coordinates, not relative
	objects map[string]Window
	isMenu  bool
}

// A MouseEvent represents down/drag/up
type MouseEvent int

// MouseEvent
const (
	MouseUp   MouseEvent = 0
	MouseDown MouseEvent = 1
	MouseDrag MouseEvent = 2
)

// ObjectUnder xxx
func ObjectUnder(objects map[string]Window, pos image.Point) Window {
	for _, o := range objects {
		if pos.In(o.Data().rect) {
			return o
		}
	}
	return nil
}

// AddObject xxx
func AddObject(objects map[string]Window, name string, o Window) {
	_, ok := objects[name]
	if ok {
		log.Printf("There's already an object named %s in that WindowData\n", name)
	} else {
		objects[name] = o
	}
}

// RemoveObject xxx
func RemoveObject(objects map[string]Window, name string, o Window) {
	oo, ok := objects[name]
	if ok {
		if oo != o {
			log.Printf("RemoveObject: Unexpected menu object for %s\n", name)
		}
		delete(objects, name)
	} else {
		log.Printf("RemoveObject: no object named %s\n", name)
	}
}
