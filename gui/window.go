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
	parent  Window
	Style   *Style
	Rect    image.Rectangle // in Screen coordinates, not relative (yet)
	MinRect image.Rectangle // Min is always 0,0

	children    map[WindowID]Window
	childID     map[Window]WindowID
	lastChildID WindowID // to generate unique child window IDs
	order       []Window // display order of child windows

	att map[string]string
}

// NewWindowData xxx
func NewWindowData(parent Window) WindowData {
	if parent == nil {
		log.Printf("NewWindowData: parent is nil!?\n")
	}
	return WindowData{
		parent:   parent,
		Style:    NewStyle("fixed", 16),
		Rect:     image.Rectangle{},
		children: make(map[WindowID]Window),
		childID:  make(map[Window]WindowID),
		order:    make([]Window, 0),
		att:      make(map[string]string),
	}
}

// WindowUnder xxx
func WindowUnder(parent Window, pos image.Point) Window {

	parentData := parent.Data()
	// Check in reverse order
	for n := len(parentData.order) - 1; n >= 0; n-- {
		w := parentData.order[n]
		if pos.In(w.Data().Rect) {
			return w
		}
	}
	return nil
}

// AddChild xxx
func AddChild(parent Window, child Window) Window {

	parentData := parent.Data()
	parentData.lastChildID++
	wid := parentData.lastChildID
	_, ok := parentData.children[wid]
	if ok {
		log.Printf("AddChild: there's already a child with id=%d ??\n", wid)
		return nil
	}

	// add it to the end of the display order
	parentData.order = append(parentData.order, child)

	parentData.children[wid] = child
	parentData.childID[child] = wid
	return child
}

// RemoveChild xxx
func RemoveChild(parent Window, child Window) {

	if child == nil {
		log.Printf("RemoveChild: child=nil?\n")
	}
	windata := parent.Data()
	wid, ok := windata.childID[child]
	if !ok {
		log.Printf("RemoveWindow: no window child=%v\n", child)
		return
	}

	delete(windata.childID, child)
	delete(windata.children, wid)

	// find and delete it in the .order array
	for n, w := range windata.order {
		if w == child {
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
func RedrawChildren(parent Window) {
	if parent == nil {
		log.Printf("RedrawChildren: parent==nil?\n")
		return
	}
	for _, w := range parent.Data().order {
		w.Do(parent, "redraw", nil)
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

// WindowRect xxx
func WindowRect(w Window) image.Rectangle {
	return w.Data().Rect
}

// WindowRaise moves w to the top of the order
func WindowRaise(parent Window, raise Window) {
	data := parent.Data()
	orderLen := len(data.order)

	// Quick check for common case when it's the top Window
	if data.order[orderLen-1] == raise {
		return
	}

	shifting := false
	for n, w := range data.order {
		if w == raise {
			shifting = true
		}
		if shifting {
			if n == (orderLen - 1) {
				data.order[n] = raise
			} else {
				data.order[n] = data.order[n+1]
			}
		}
	}
}
