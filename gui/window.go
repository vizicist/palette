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
	Context() *WinContext
	Do(cmd string, arg interface{}) (interface{}, error)
}

// ToolData fields are exported
type ToolData struct {
	w        Window
	toolType string
	minSize  image.Point
}

// NewToolData xxx
func NewToolData(w Window, toolType string, minSize image.Point) ToolData {
	return ToolData{
		w:        w,
		toolType: toolType,
		minSize:  minSize,
	}
}

// ToolMaker xxx
type ToolMaker func(parent Window) ToolData

// Tools xxx
var Tools = make(map[string]ToolMaker)

// WindowID xxx
type WindowID int

// WinContext doesn't export any of its fields
type WinContext struct {
	parent      Window
	minSize     image.Point
	toolType    string
	currSz      image.Point
	style       *Style
	initialized bool

	childData   map[Window]*WinContext
	childWindow map[WindowID]Window
	childID     map[Window]WindowID
	childPos    map[Window]image.Point
	lastChildID WindowID // to generate unique child window IDs
	order       []Window // display order of child windows
	saveMenu    *Menu    // XXX - eventually this should be a list to handle deeper nesting

	att map[string]string
}

// NewWindowContext xxx
func NewWindowContext(parent Window) *WinContext {
	return &WinContext{

		parent:      parent,
		style:       nil, // uses parent's
		minSize:     image.Point{},
		currSz:      image.Point{},
		initialized: true,

		childData:   make(map[Window]*WinContext),
		childWindow: make(map[WindowID]Window),
		childID:     make(map[Window]WindowID),
		childPos:    make(map[Window]image.Point),
		order:       make([]Window, 0),
		att:         make(map[string]string),
	}
}

// WindowUnder looks for a child window under a given point,
// and if there is one, returns both the window and a point that has
// been adjusted to make it relative to the child's coordinate space.
func WindowUnder(parent Window, pos image.Point) (Window, image.Point) {
	pc := parent.Context()
	// Check in reverse order
	for n := len(pc.order) - 1; n >= 0; n-- {
		w := pc.order[n]
		r := WinChildRect(parent, w)
		if pos.In(r) {
			return w, RelativePos(parent, w, pos)
		}
	}
	return nil, image.Point{}
}

// AddChild xxx
func AddChild(parent Window, td ToolData) Window {

	child := td.w
	cc := child.Context()
	if cc.initialized == false {
		log.Printf("AddChild: child.Context not initialized!\n")
		return nil
	}
	cc.minSize = td.minSize
	cc.currSz = td.minSize
	cc.toolType = td.toolType

	pc := parent.Context()
	if pc.initialized == false {
		log.Printf("AddChild: parent.Data not initialized!?\n")
		return nil
	}

	pc.lastChildID++
	wid := pc.lastChildID
	_, ok := pc.childWindow[wid]
	if ok {
		log.Printf("AddChild: there's already a child with id=%d ??\n", wid)
		return nil
	}

	// add it to the end of the display order
	pc.order = append(pc.order, child)

	pc.childWindow[wid] = child
	pc.childID[child] = wid
	pc.childData[child] = child.Context()
	pc.childPos[child] = image.Point{0, 0}

	return child
}

// WinChildData xxx
func WinChildData(parent Window, child Window) *WinContext {
	wd, ok := parent.Context().childData[child]
	if !ok {
		log.Printf("WinChildData: no child?\n")
		return nil
	}
	return wd
}

// RemoveChild xxx
func RemoveChild(parent Window, child Window) {

	if child == nil {
		log.Printf("RemoveChild: child=nil?\n")
	}
	pc := parent.Context()
	if pc == nil {
		log.Printf("RemoveChild: no WinData for parent=%v\n", parent)
		return
	}
	wid, ok := pc.childID[child]
	if !ok {
		log.Printf("RemoveWindow: no window child=%v\n", child)
		return
	}

	delete(pc.childID, child)
	delete(pc.childWindow, wid)

	// find and delete it in the .order array
	for n, w := range pc.order {
		if w == child {
			copy(pc.order[n:], pc.order[n+1:])
			newlen := len(pc.order) - 1
			pc.order = pc.order[:newlen]
			break
		}
	}
}

// MoveWindow xxx
func MoveWindow(parent Window, child Window, delta image.Point) {
	pc := parent.Context()
	childPos, ok := pc.childPos[child]
	if !ok {
		log.Printf("WinChildPos: w not in parent childPos?\n")
		return
	}
	pc.childPos[child] = childPos.Add(delta)
}

// RedrawChildren xxx
func RedrawChildren(parent Window) {
	if parent == nil {
		log.Printf("RedrawChildren: parent==nil?\n")
		return
	}
	pc := parent.Context()
	for _, w := range pc.order {
		w.Do("redraw", nil)
	}
}

// GetAttValue xxx
func GetAttValue(w Window, name string) string {
	wc := w.Context()
	if wc == nil {
		log.Printf("GetAttValue: no WinData for w=%v\n", w)
		return ""
	}
	return wc.att[name]
}

// SetAttValue xxx
func SetAttValue(w Window, name string, val string) {
	wc := w.Context()
	if wc == nil {
		log.Printf("SetAttValue: no WinData for w=%v\n", w)
		return
	}
	wc.att[name] = val
}

// DoUpstream xxx
func DoUpstream(w Window, cmd string, arg interface{}) {
	// log.Printf("DoUpstream: cmd=%s arg=%v\n", cmd, arg)
	parent := WinParent(w)
	if parent == nil {
		log.Printf("DoUpstream: no parent for w=%v\n", w)
		return
	}

	// Adjust coordinates to reflect child's position in the parent
	pos := WinChildPos(parent, w)
	switch cmd {
	case "drawline":
		dl := ToDrawLine(arg)
		dl.XY0 = dl.XY0.Add(pos)
		dl.XY1 = dl.XY1.Add(pos)
		arg = dl
	case "drawrect":
		r := ToRect(arg)
		arg = r.Add(pos)
	case "drawfilledrect":
		r := ToRect(arg)
		arg = r.Add(pos)
	case "drawtext":
		t := ToDrawText(arg)
		t.Pos = t.Pos.Add(pos)
		arg = t
	}

	parent.Do(cmd, arg)
}

/*
// WindowRect xxx
func WindowRect(w Window) image.Rectangle {
	return WinData(w).rect
}
*/

// WindowRaise moves w to the top of the order
func WindowRaise(parent Window, raise Window) {
	pc := parent.Context()
	orderLen := len(pc.order)

	// Quick check for common case when it's the top Window
	if pc.order[orderLen-1] == raise {
		return
	}

	shifting := false
	for n, w := range pc.order {
		if w == raise {
			shifting = true
		}
		if shifting {
			if n == (orderLen - 1) {
				pc.order[n] = raise
			} else {
				pc.order[n] = pc.order[n+1]
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

// WinToolType xxx
func WinToolType(w Window) string {
	return w.Context().toolType
}

// WinSaveMenu xxx
func WinSaveMenu(w Window, menu *Menu) {
	w.Context().saveMenu = menu
}

// WinCurrSize xxx
func WinCurrSize(w Window) (p image.Point) {
	return w.Context().currSz
}

// WinSetMySize xxx
func WinSetMySize(w Window, size image.Point) {
	w.Context().currSz = size
	// Don't do w.Do(), that would be recursive
}

// WinSetChildSize xxx
func WinSetChildSize(w Window, size image.Point) {
	if size.X == 0 || size.Y == 0 {
		log.Printf("WinSetChildSize: too small, setting to 100,100\n")
		size = image.Point{100, 100}
	}
	w.Context().currSz = size
	w.Do("resize", size)
}

// WinSetChildPos xxx
func WinSetChildPos(parent Window, child Window, pos image.Point) {
	if parent == nil {
		log.Printf("WinChildPos: parent is nil?\n")
		return
	}
	parent.Context().childPos[child] = pos
}

// WinChildPos xxx
func WinChildPos(parent Window, child Window) (p image.Point) {
	if parent == nil {
		log.Printf("WinChildPos: parent is nil?\n")
		return
	}
	childPos, ok := parent.Context().childPos[child]
	if !ok {
		log.Printf("WinChildPos: w not in parent childPos?\n")
		return
	}
	return childPos
}

// WinChildRect xxx
func WinChildRect(parent, child Window) (r image.Rectangle) {
	// A child's Rectangle is determined by two things:
	// 1) the childPos stored in the parent
	// 2) the currSize stored in the child
	childPos := WinChildPos(parent, child)
	currSize := WinCurrSize(child)
	return image.Rectangle{Min: childPos, Max: childPos.Add(currSize)}
}

// WinMinSize xxx
func WinMinSize(w Window) (r image.Point) {
	return w.Context().minSize
}

// WinParent xxx
func WinParent(w Window) Window {
	return w.Context().parent
}

// WinStyle xxx
func WinStyle(w Window) *Style {
	ctx := w.Context()
	if ctx.style != nil {
		return ctx.style // Window has its own style
	}
	if ctx.parent == nil {
		log.Printf("WinStye: using DefaultStyle because no parent for w=%v\n", w)
		return DefaultStyle()
	}
	return WinStyle(ctx.parent) // use the parent's style
}
