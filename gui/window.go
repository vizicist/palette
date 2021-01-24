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

// WindowData xxx
type WindowData struct {
	parent Window
	Style  *Style
	Rect   image.Rectangle // in Screen coordinates, not relative

	children   map[string]Window
	windowName map[Window]string

	order []string // display order
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
		children:   make(map[string]Window),
		windowName: make(map[Window]string),
		// childWaitGroup: &sync.WaitGroup{},
		order: make([]string, 0),
	}
}

// WindowUnder xxx
func WindowUnder(parent Window, pos image.Point) (Window, string) {

	parentData := parent.Data()

	// Check in reverse order
	for n := len(parentData.order) - 1; n >= 0; n-- {
		name := parentData.order[n]

		child := parentData.children[name]
		childRect := child.Data().Rect

		if pos.In(childRect) {
			return parentData.children[name], name
		}
	}
	return nil, ""
}

// AddChild xxx
func AddChild(parent Window, name string, child Window) Window {

	parentData := parent.Data()
	_, ok := parentData.children[name]
	if ok {
		log.Printf("AddChild: there's already a child named %s\n", name)
		return nil
	}

	// add it to the end of the display order
	parentData.order = append(parentData.order, name)

	parentData.children[name] = child
	parentData.windowName[child] = name
	return child
}

// RemoveChild xxx
func RemoveChild(parent Window, w Window) {

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
			windata.order[newlen] = "" // XXX does this do anything?
			windata.order = windata.order[:newlen]
			break
		}
	}
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

// FindChild xxx
func FindChild(parent Window, name string) Window {
	for nm, w := range parent.Data().children {
		if nm == name {
			return w
		}
	}
	return nil
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
