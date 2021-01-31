package gui

import (
	"fmt"
	"image"
	"log"
	"strings"
)

// Window is the external (and networkable) interface
// to a Window instance.
type Window interface {
	// Data() *WindowData // local only
	Do(from Window, cmd string, arg interface{}) (interface{}, error)
}

// WindowID xxx
type WindowID int

// WindowData xxx
type WindowData struct {
	rect    image.Rectangle // in Screen coordinates, not relative (yet)
	parent  Window
	minRect image.Rectangle // Min is always 0,0
	style   *Style

	children    map[WindowID]Window
	childID     map[Window]WindowID
	lastChildID WindowID // to generate unique child window IDs
	order       []Window // display order of child windows

	att map[string]string
}

// WinMap xxx
var WinMap = make(map[Window]*WindowData)

// ToolMaker xxx
type ToolMaker func(style *Style) (w Window, minRect image.Rectangle)

// Tools xxx
var Tools = make(map[string]ToolMaker)

// NewWindowData xxx
func NewWindowData(parent Window, minRect image.Rectangle, style *Style) *WindowData {
	if parent == nil {
		log.Printf("NewWindowData: parent is nil!?\n")
	}
	return &WindowData{

		parent:  parent,
		style:   style,
		minRect: minRect,

		rect:     image.Rectangle{},
		children: make(map[WindowID]Window),
		childID:  make(map[Window]WindowID),
		order:    make([]Window, 0),
		att:      make(map[string]string),
	}
}

// WindowUnder xxx
func WindowUnder(parent Window, pos image.Point) Window {

	parentData := WinData(parent)
	// Check in reverse order
	for n := len(parentData.order) - 1; n >= 0; n-- {
		w := parentData.order[n]
		if pos.In(WinRect(w)) {
			return w
		}
	}
	return nil
}

// AddChild xxx
func AddChild(parent Window, child Window) Window {

	parentData := WinData(parent)
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
	parentData := WinData(parent)
	if parentData == nil {
		log.Printf("RemoveChild: no WinData for parent=%v\n", parent)
	}
	wid, ok := parentData.childID[child]
	if !ok {
		log.Printf("RemoveWindow: no window child=%v\n", child)
		return
	}

	delete(parentData.childID, child)
	delete(parentData.children, wid)

	// find and delete it in the .order array
	for n, w := range parentData.order {
		if w == child {
			copy(parentData.order[n:], parentData.order[n+1:])
			newlen := len(parentData.order) - 1
			parentData.order = parentData.order[:newlen]
			break
		}
	}
}

// MoveWindow xxx
func MoveWindow(parent Window, child Window, delta image.Point) {
	child.Do(parent, "resize", WinRect(child).Add(delta))
}

// RedrawChildren xxx
func RedrawChildren(parent Window) {
	if parent == nil {
		log.Printf("RedrawChildren: parent==nil?\n")
		return
	}
	wd := WinData(parent)
	if wd == nil {
		log.Printf("RedrawChildren: No WinData for parent=%v\n", parent)
		return
	}
	for _, w := range wd.order {
		w.Do(parent, "redraw", nil)
	}
}

// GetAttValue xxx
func GetAttValue(w Window, name string) string {
	wd := WinData(w)
	if wd == nil {
		log.Printf("GetAttValue: no WinData for w=%v\n", w)
		return ""
	}
	return wd.att[name]
}

// SetAttValue xxx
func SetAttValue(w Window, name string, val string) {
	wd := WinData(w)
	if wd == nil {
		log.Printf("SetAttValue: no WinData for w=%v\n", w)
		return
	}
	wd.att[name] = val
}

// DoUpstream xxx
func DoUpstream(w Window, cmd string, arg interface{}) {
	wd := WinData(w)
	if wd == nil {
		log.Printf("DoUpstream: no WinData for w=%v\n", w)
	} else {
		p := wd.parent
		if p == nil {
			log.Printf("DoUpstream: no parent for w=%v\n", w)
		} else {
			p.Do(w, cmd, arg)
		}
	}
}

// WindowRect xxx
func WindowRect(w Window) image.Rectangle {
	return WinData(w).rect
}

// WindowRaise moves w to the top of the order
func WindowRaise(parent Window, raise Window) {
	pdata := WinData(parent)
	orderLen := len(pdata.order)

	// Quick check for common case when it's the top Window
	if pdata.order[orderLen-1] == raise {
		return
	}

	shifting := false
	for n, w := range pdata.order {
		if w == raise {
			shifting = true
		}
		if shifting {
			if n == (orderLen - 1) {
				pdata.order[n] = raise
			} else {
				pdata.order[n] = pdata.order[n+1]
			}
		}
	}
}

// Point - should I start using this to make code more compact?
func Point(x, y int) image.Point {
	return image.Point{X: x, Y: y}
}

// WindowType xxx
func WindowType(w Window) string {
	t := fmt.Sprintf("%T", w)
	i := strings.LastIndex(t, ".")
	if i >= 0 {
		t = t[i+1:]
	}
	return t
}

// AddToolType xxx
func AddToolType(name string, newfunc ToolMaker) {
	log.Printf("AddToolType name=%s\n", name)
	Tools[name] = newfunc
}

// WinData xxx
func WinData(w Window) *WindowData {
	wd, ok := WinMap[w]
	if !ok {
		log.Printf("WinData: no entry in WinMap for w=%v\n", w)
		return nil
	}
	return wd
}

// WinRect xxx
func WinRect(w Window) (r image.Rectangle) {
	if wd := WinData(w); wd != nil {
		r = wd.rect
	}
	return r
}

// WinMinRect xxx
func WinMinRect(w Window) (r image.Rectangle) {
	if wd := WinData(w); wd != nil {
		r = wd.minRect
	}
	return r
}

// WinParent xxx
func WinParent(w Window) Window {
	if wd := WinData(w); wd != nil {
		return wd.parent
	}
	return nil
}

// WinStyle xxx
func WinStyle(w Window) *Style {
	wd := WinData(w)
	if wd.style != nil {
		return wd.style // Window has its own style
	}
	if wd.parent == nil {
		log.Printf("WinStye: using DefaultStyle because no parent for w=%v\n", wd)
		return DefaultStyle()
	}
	return WinStyle(wd.parent) // use the parent's style
}
