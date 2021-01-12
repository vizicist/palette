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
	Data() *WindowData
}

// WindowData xxx
type WindowData struct {
	screen  *Screen
	style   *Style
	rect    image.Rectangle // in Screen coordinates, not relative
	objects map[string]Window
	order   []string // display order
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
func ObjectUnder(o Window, pos image.Point) Window {
	windata := o.Data()
	for _, name := range windata.order {
		w, ok := windata.objects[name]
		if !ok {
			log.Printf("ObjectUnder: no entry in object for %s\n", name)
			return nil
		}
		if w == nil {
			log.Printf("ObjectUnder: objects entry for %s is nul?\n", name)
			return nil
		}
		if pos.In(w.Data().rect) {
			return w
		}
	}
	return nil
}

// AddObject xxx
func AddObject(parent Window, name string, o Window) {
	windata := parent.Data()
	objects := windata.objects
	_, ok := objects[name]
	if ok {
		log.Printf("There's already an object named %s in that Window\n", name)
	} else {
		objects[name] = o
		// add it to the end of the display order
		windata.order = append(windata.order, name)
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
