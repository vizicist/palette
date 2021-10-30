package winsys

import (
	"fmt"
	"image"
	"log"

	"github.com/vizicist/palette/engine"
)

// Window is the external (and networkable) interface
// to a Window instance.   Context is only local (I think).
type Window interface {
	Context() *WinContext
	Do(cmd engine.Cmd) string
}

// WinContext doesn't export any of its fields
type WinContext struct {
	parent      Window
	minSize     image.Point
	toolType    string
	currSz      image.Point
	styleName   string
	initialized bool

	childWindow map[string]Window // map window name to Window
	childName   map[Window]string
	childPos    map[Window]image.Point
	lastChildID int               // to generate unique child window IDs
	order       []Window          // display order of child windows
	transients  map[Window]string // used for transient popup Menus

	att map[string]string
}

// NewWindowContext xxx
func NewWindowContext(parent Window) WinContext {
	return WinContext{

		parent:      parent,
		styleName:   "", // uses parent's
		minSize:     image.Point{},
		currSz:      image.Point{},
		initialized: true,

		childWindow: make(map[string]Window),
		childName:   make(map[Window]string),
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
func AddChild(parent Window, td WindowData) Window {

	child := td.w
	cc := child.Context()
	if !cc.initialized {
		log.Printf("AddChild: child.Context not initialized!\n")
		return nil
	}
	cc.minSize = td.minSize
	cc.currSz = td.minSize
	cc.toolType = td.toolType

	pc := parent.Context()
	if !pc.initialized {
		log.Printf("AddChild: parent.Data not initialized!?\n")
		return nil
	}

	pc.lastChildID++
	wname := fmt.Sprintf("%s.%d", td.toolType, pc.lastChildID)
	_, ok := pc.childWindow[wname]
	if ok {
		log.Printf("AddChild: there's already a child with name=%s ??\n", wname)
		return nil
	}

	// add it to the end of the display order
	pc.order = append(pc.order, child)

	pc.childWindow[wname] = child
	pc.childName[child] = wname
	pc.childPos[child] = image.Point{0, 0}
	if debugChild {
		log.Printf("AddChild: wname=%s child=%p\n", wname, child)
	}

	return child
}

// RemoveChild xxx
func RemoveChild(parent Window, child Window) {

	if child == nil {
		log.Printf("RemoveChild: child=nil?\n")
	}
	pc := parent.Context()
	childName, ok := pc.childName[child]
	if !ok {
		// XXX - this happens when you're restoring, not sure why,
		// XXX - but for the moment I'll just ignore it silently
		// log.Printf("RemoveChild: no window child=%v\n", child)
		return
	}
	if debugChild {
		log.Printf("RemoveChild: removing wid=%s\n", childName)
	}

	delete(pc.childName, child)
	delete(pc.childWindow, childName)

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
		w.Do(engine.NewSimpleCmd("redraw"))
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

func GetAndAdjustXY01(cmd engine.Cmd, adjust image.Point) engine.Cmd {
	xy0 := cmd.ValuesXY0(image.Point{})
	xy1 := cmd.ValuesXY1(image.Point{})
	newxy0 := xy0.Add(adjust)
	newxy1 := xy1.Add(adjust)
	cmd.ValuesSetXY0(newxy0)
	cmd.ValuesSetXY1(newxy1)
	return cmd
}

// DoUpstream xxx
func DoUpstream(w Window, cmd engine.Cmd) {
	subj := cmd.Subj
	// log.Printf("DoUpstream: cmd=%s arg=%v\n", cmd, arg)
	parent := WinParent(w)
	if parent == nil {
		log.Printf("DoUpstream: no parent for w=%v\n", w)
		return
	}

	// Adjust coordinates to reflect child's position in the parent
	adjust := WinChildPos(parent, w)

	var forwarded engine.Cmd

	switch subj {

	case "drawline":
		cmd := GetAndAdjustXY01(cmd, adjust)
		forwarded = cmd

	case "drawrect":
		cmd := GetAndAdjustXY01(cmd, adjust)
		forwarded = cmd

	case "drawfilledrect":
		cmd := GetAndAdjustXY01(cmd, adjust)
		forwarded = cmd

	case "drawtext":
		pos := cmd.ValuesXY("pos", engine.PointZero)
		newpos := pos.Add(adjust)
		cmd.ValuesSetPos(newpos)
		forwarded = cmd

	default:
		forwarded = cmd
	}

	parent.Do(forwarded)
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
				log.Printf("winRemoveTransients: about to remove wid=%s\n", WinChildName(parent, w))
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
	w.Do(NewResizeCmd(size))
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

// WinChildName xxx
func WinChildName(parent Window, child Window) string {
	if parent == nil {
		log.Printf("WinChildID: parent is nil?\n")
		return ""
	}
	id, ok := parent.Context().childName[child]
	if !ok {
		log.Printf("WinChildID: w not in parent childName?\n")
		return ""
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

// WinStyleName xxx
func WinStyleInfo(w Window) *StyleInfo {
	styleName := WinStyleName(w)
	return Styles[styleName]
}

// WinStyleName xxx
func WinStyleName(w Window) string {
	ctx := w.Context()
	if ctx.styleName != "" {
		return ctx.styleName // Window has its own style
	}
	if ctx.parent == nil {
		log.Printf("WinStye: using DefaultStyle because no parent for w=%v\n", w)
		return DefaultStyleName()
	}
	return WinStyleName(ctx.parent) // use the parent's style
}

// WinRelativePos xxx
func WinRelativePos(parent Window, w Window, pos image.Point) image.Point {
	childPos := WinChildPos(parent, w)
	relativePos := pos.Sub(childPos)
	return relativePos
}

// WinForwardMouse is a utility function for Tools that just want
// to forward all their mouse events to whatever sub-windows they have.
func WinForwardMouse(w Window, cmd engine.Cmd) {
	ddu := cmd.ValuesString("ddu", "")
	bnum := cmd.ValuesInt("buttonnum", 0)
	pos := cmd.ValuesPos(engine.PointZero)
	child, relPos := WindowUnder(w, pos)
	if child != nil {
		relcmd := NewMouseCmd(ddu, relPos, bnum)
		child.Do(relcmd)
	}
}
