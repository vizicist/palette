package gui

import (
	"image"
	"log"
)

// Window is the external (and networkable) interface
// to a Window instance.   Context is only local (I think).
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

// ToolMakers xxx
var ToolMakers = make(map[string]ToolMaker)

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

	childWindow map[WindowID]Window
	childID     map[Window]WindowID
	childPos    map[Window]image.Point
	lastChildID WindowID          // to generate unique child window IDs
	order       []Window          // display order of child windows
	transients  map[Window]string // used for transient popup Menus

	att map[string]string
}

// NewWindowContext xxx
func NewWindowContext(parent Window) WinContext {
	return WinContext{

		parent:      parent,
		style:       nil, // uses parent's
		minSize:     image.Point{},
		currSz:      image.Point{},
		initialized: true,

		childWindow: make(map[WindowID]Window),
		childID:     make(map[Window]WindowID),
		childPos:    make(map[Window]image.Point),
		order:       make([]Window, 0),
		att:         make(map[string]string),
		transients:  make(map[Window]string),
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
			return w, WinRelativePos(parent, w, pos)
		}
	}
	return nil, image.Point{}
}

var debugChild = false

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
	pc.childPos[child] = image.Point{0, 0}
	if debugChild {
		log.Printf("AddChild: wid=%d child=%p\n", wid, child)
	}

	return child
}

// RemoveChild xxx
func RemoveChild(parent Window, child Window) {

	if child == nil {
		log.Printf("RemoveChild: child=nil?\n")
	}
	pc := parent.Context()
	wid, ok := pc.childID[child]
	if !ok {
		log.Printf("RemoveChild: no window child=%v\n", child)
		return
	}
	if debugChild {
		log.Printf("RemoveChild: removing wid=%d\n", wid)
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
		log.Printf("WinMoveWindow: w not in parent childPos?\n")
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
	return wc.att[name]
}

// SetAttValue xxx
func SetAttValue(w Window, name string, val string) {
	wc := w.Context()
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

// RegisterToolType xxx
func RegisterToolType(name string, newfunc ToolMaker) {
	log.Printf("RegisterToolType name=%s\n", name)
	ToolMakers[name] = newfunc
}

// ToolType xxx
func ToolType(w Window) string {
	return w.Context().toolType
}

func winSaveTransient(parent Window, w Window) {
	if debugChild {
		log.Printf("winSaveTransient: w=%p\n", w)
	}
	parent.Context().transients[w] = "dummy"
}

func winMakePermanent(parent Window, w Window) {
	if debugChild {
		log.Printf("winMakePermanent: w=%p\n", w)
	}
	delete(parent.Context().transients, w)
}

func winIsTransient(parent Window, w Window) bool {
	_, ok := parent.Context().transients[w]
	return ok
}

func winRemoveTransients(parent Window, exceptMenu Window) {
	wc := parent.Context()
	// Remove any transient windows (i.e. popup menus)
	for w := range wc.transients {
		if w != exceptMenu {
			if debugChild {
				log.Printf("winRemoveTransients: about to remove wid=%d\n", WinChildID(parent, w))
			}
			RemoveChild(parent, w)
			delete(wc.transients, w)
		}
	}
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
		log.Printf("WinSeetChildPos: parent is nil?\n")
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

// WinChildID xxx
func WinChildID(parent Window, child Window) WindowID {
	if parent == nil {
		log.Printf("WinChildID: parent is nil?\n")
		return 0
	}
	id, ok := parent.Context().childID[child]
	if !ok {
		log.Printf("WinChildID: w not in parent childID?\n")
		return 0
	}
	return id
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

// WinRelativePos xxx
func WinRelativePos(parent Window, w Window, pos image.Point) image.Point {
	childPos := WinChildPos(parent, w)
	relativePos := pos.Sub(childPos)
	return relativePos
}

// WinForwardMouse is a utility function for Tools that just want
// to forward all their mouse events to whatever sub-windows they have.
func WinForwardMouse(w Window, mouse MouseCmd) {
	child, relPos := WindowUnder(w, mouse.Pos)
	if child != nil {
		mouse.Pos = relPos
		child.Do("mouse", mouse)
	}
}
