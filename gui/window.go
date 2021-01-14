package gui

import (
	"image"
	"log"
)

// Window xxx
type Window interface {
	Run()
	Draw()
	Resize(image.Rectangle) image.Rectangle
	Data() *WindowData
}

// WindowData xxx
type WindowData struct {
	mouseChan chan MouseEvent // unbuffered?
	screen    *Screen
	style     *Style
	rect      image.Rectangle // in Screen coordinates, not relative
	objects   map[string]Window
	order     []string // display order
	isMenu    bool
}

// NewWindowData xxx
func NewWindowData(parent Window) WindowData {
	return WindowData{
		screen:    parent.Data().screen,
		style:     parent.Data().style,
		rect:      image.Rectangle{},
		objects:   map[string]Window{},
		mouseChan: make(chan MouseEvent), // XXX - unbuffered?
	}
}

// A MouseEvent represents down/drag/up
type MouseEvent struct {
	pos     image.Point
	buttNum int
	ddu     DownDragUp
}

// DownDragUp xxx
type DownDragUp int

// MouseUp xxx
const (
	MouseUp DownDragUp = iota
	MouseDown
	MouseDrag
)

// ObjectUnder xxx
func ObjectUnder(o Window, pos image.Point) Window {
	windata := o.Data()
	// Check in reverse order
	for n := len(windata.order) - 1; n >= 0; n-- {
		name := windata.order[n]
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
func RemoveObject(parent Window, name string) {

	windata := parent.Data()
	_, ok := windata.objects[name]
	if !ok {
		log.Printf("RemoveObject: no object named %s\n", name)
		return
	}

	delete(windata.objects, name)
	// find and delete it in order
	for n, nm := range windata.order {
		if nm == name {
			copy(windata.order[n:], windata.order[n+1:])
			newlen := len(windata.order) - 1
			windata.order[newlen] = ""
			windata.order = windata.order[:newlen]
			break
		}
	}
}
