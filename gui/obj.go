package gui

import (
	"image"
	"log"
)

// Window xxx
type Window interface {
	HandleMouseInput(pos image.Point, button int, down bool) bool
	Draw(*Screen)
	Resize(image.Rectangle)
	Data() WindowData
}

// WindowData xxx
type WindowData struct {
	style   *Style
	rect    image.Rectangle // relative to the parent Window
	objects map[string]Window
	isMenu  bool
}

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
