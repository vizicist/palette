package gui

import (
	"image"
	"log"
)

// Window xxx
type Window interface {
	Data() *WindowData
	Do(from Window, cmd string, arg interface{})
	DoSync(from Window, cmd string, arg interface{}) (result interface{}, err error)
}

// WindowID xxx
type WindowID int

// WindowData xxx
type WindowData struct {
	parent Window
	Style  *Style
	Rect   image.Rectangle // in Screen coordinates, not relative

	children   map[WindowID]Window
	windowName map[Window]WindowID
	lastID     WindowID // sequence number for window names

	order []WindowID // display order
	att   map[string]string
}

// NewWindowData xxx
func NewWindowData(parent Window) WindowData {
	if parent == nil {
		log.Printf("NewWindowData: parent is nil!?\n")
	}
	return WindowData{
		parent: parent,
		// fromUpstream:    fromUpstream,
		// toUpstream:      toUpstream,
		Style:      NewStyle("fixed", 16),
		Rect:       image.Rectangle{},
		children:   make(map[WindowID]Window),
		windowName: make(map[Window]WindowID),
		// childWaitGroup: &sync.WaitGroup{},
		order: make([]WindowID, 0),
		att:   make(map[string]string),
	}
}

// WindowUnder xxx
func WindowUnder(parent Window, pos image.Point) Window {

	parentData := parent.Data()

	// Check in reverse order
	for n := len(parentData.order) - 1; n >= 0; n-- {
		name := parentData.order[n]

		child := parentData.children[name]
		childRect := child.Data().Rect

		if pos.In(childRect) {
			return parentData.children[name]
		}
	}
	return nil
}

// AddChild xxx
func AddChild(w Window, child Window) Window {

	wdata := w.Data()
	wdata.lastID++
	wid := wdata.lastID
	_, ok := wdata.children[wid]
	if ok {
		log.Printf("AddChild: there's already a child with id=%d ??\n", wid)
		return nil
	}

	// add it to the end of the display order
	wdata.order = append(wdata.order, wid)

	wdata.children[wid] = child
	wdata.windowName[child] = wid
	return child
}

// RemoveChild xxx
func RemoveChild(parent Window, w Window) {

	if w == nil {
		log.Printf("RemoveChild: w==nil?\n")
	}
	windata := parent.Data()
	name, ok := windata.windowName[w]
	if !ok {
		log.Printf("RemoveWindow: no window w=%v\n", w)
		return
	}

	delete(windata.windowName, w)
	delete(windata.children, name)

	// find and delete it in the .order array
	for n, win := range windata.order {
		if name == win {
			copy(windata.order[n:], windata.order[n+1:])
			newlen := len(windata.order) - 1
			windata.order = windata.order[:newlen]
			break
		}
	}
}

// MoveWindow xxx
func MoveWindow(parent Window, child Window, delta image.Point) {
	child.Do(parent, "resize", child.Data().Rect.Add(delta))
}

// RedrawChildren xxx
func RedrawChildren(w Window) {
	if w == nil {
		log.Printf("RedrawChildren: w==nil?\n")
		return
	}
	for _, name := range w.Data().order {
		child := w.Data().children[name]
		child.Do(w, "redraw", nil)
	}
}

// GetAttValue xxx
func GetAttValue(w Window, name string) string {
	return w.Data().att[name]
}

// SetAttValue xxx
func SetAttValue(w Window, name string, val string) {
	w.Data().att[name] = val
}

// DoUpstream xxx
func DoUpstream(w Window, cmd string, arg interface{}) {
	p := w.Data().parent
	if p == nil {
		log.Printf("Hey, no parent?\n")
	} else {
		p.Do(w, cmd, arg)
	}
}
