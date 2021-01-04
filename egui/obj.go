package egui

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
	rect    image.Rectangle
	objects map[string]Window
}

// ToWindow xxx
func ToWindow(ww interface{}) Window {
	w, ok := ww.(Window)
	if !ok {
		log.Printf("ToWindow: isn't a Window?")
		return nil
	}
	return w
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
