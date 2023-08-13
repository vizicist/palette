package twinsys

import (
	"fmt"
	"image"

	"github.com/vizicist/palette/hostwin"
)

// Window is the external (and networkable) interface
// to a Window instance.   Context is only local (I think).

// All of the funcs that we want to work on this interface are
// named Win*

type Window interface {
	Context() *WinContext
	Do(cmd hostwin.Cmd) string
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

// NewWindowContextNoParent xxx
func newWindowContextNoParent() WinContext {
	return realNewWindowContext(nil, DefaultStyleName())
}

// NewWindowContext xxx
func NewWindowContext(parent Window) WinContext {
	var style string
	if parent == nil {
		hostwin.LogWarn("NewWindowContext: unexpected parent == nil?")
		style = parent.Context().styleName
	}
	return realNewWindowContext(parent, style)
}

func realNewWindowContext(parent Window, style string) WinContext {

	if style == "" {
		style = DefaultStyleName()
	}
	return WinContext{

		parent:      parent,
		styleName:   style,
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

// WinFindWindowUnder looks for a child window under a given point,
// and if there is one, returns both the window and a point that has
// been adjusted to make it relative to the child's coordinate space.
func WinFindWindowUnder(parent Window, pos image.Point) (Window, image.Point) {
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

// WinAddChild xxx
func WinAddChild(parent Window, td WindowData) Window {

	child := td.w
	cc := child.Context()
	if !cc.initialized {
		hostwin.LogWarn("AddChild: child.Context not initialized!")
		return nil
	}
	cc.minSize = td.minSize
	cc.currSz = td.minSize
	cc.toolType = td.toolType

	pc := parent.Context()
	if !pc.initialized {
		hostwin.LogWarn("AddChild: parent.Data not initialized!?")
		return nil
	}

	pc.lastChildID++
	wname := fmt.Sprintf("%s.%d", td.toolType, pc.lastChildID)
	_, ok := pc.childWindow[wname]
	if ok {
		hostwin.LogWarn("AddChild: there's already a child with", "name", wname)
		return nil
	}

	// add it to the end of the display order
	pc.order = append(pc.order, child)

	pc.childWindow[wname] = child
	pc.childName[child] = wname
	pc.childPos[child] = image.Point{0, 0}

	return child
}

// WinRemoveChild xxx
func WinRemoveChild(parent Window, child Window) {

	if child == nil {
		hostwin.LogWarn("RemoveChild: child=nil?")
	}
	pc := parent.Context()
	childName, ok := pc.childName[child]
	if !ok {
		// XXX - this happens when you're restoring, not sure why,
		// XXX - but for the moment I'll just ignore it silently
		return
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

// winMoveWindow xxx
func winMoveWindow(parent Window, child Window, delta image.Point) {
	pc := parent.Context()
	childPos, ok := pc.childPos[child]
	if !ok {
		hostwin.LogWarn("WinMoveWindow: w not in parent childPos?")
		return
	}
	pc.childPos[child] = childPos.Add(delta)
}

// WinRedrawChildren xxx
func WinRedrawChildren(parent Window) {
	if parent == nil {
		hostwin.LogWarn("RedrawChildren: parent==nil?")
		return
	}
	pc := parent.Context()
	for _, w := range pc.order {
		w.Do(hostwin.NewSimpleCmd("redraw"))
	}
}

// WinGetAttValue xxx
func WinGetAttValue(w Window, name string) string {
	wc := w.Context()
	return wc.att[name]
}

// WinSetAttValue xxx
func WinSetAttValue(w Window, name string, val string) {
	wc := w.Context()
	wc.att[name] = val
}

func getAndAdjustXY01(cmd hostwin.Cmd, adjust image.Point) hostwin.Cmd {
	xy0 := cmd.ValuesXY0(image.Point{})
	xy1 := cmd.ValuesXY1(image.Point{})
	newxy0 := xy0.Add(adjust)
	newxy1 := xy1.Add(adjust)
	cmd.ValuesSetXY0(newxy0)
	cmd.ValuesSetXY1(newxy1)
	return cmd
}

// WinDoUpstream xxx
func WinDoUpstream(w Window, cmd hostwin.Cmd) {
	subj := cmd.Subj
	// hostwin.Info("DoUpstream","cmd",cmd,"arg",arg)
	parent := WinParent(w)
	if parent == nil {
		hostwin.LogWarn("DoUpstream: no parent", "w", w)
		return
	}

	// Adjust coordinates to reflect child's position in the parent
	adjust := WinChildPos(parent, w)

	var forwarded hostwin.Cmd

	switch subj {

	case "drawline":
		cmd := getAndAdjustXY01(cmd, adjust)
		forwarded = cmd

	case "drawrect":
		cmd := getAndAdjustXY01(cmd, adjust)
		forwarded = cmd

	case "drawfilledrect":
		cmd := getAndAdjustXY01(cmd, adjust)
		forwarded = cmd

	case "drawtext":
		pos := cmd.ValuesXY("pos", hostwin.PointZero)
		newpos := pos.Add(adjust)
		cmd.ValuesSetPos(newpos)
		forwarded = cmd

	default:
		forwarded = cmd
	}

	parent.Do(forwarded)
}

// winRaise moves w to the top of the order
func winRaise(parent Window, raise Window) {
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

// winToolType xxx
func winToolType(w Window) string {
	return w.Context().toolType
}

func winSaveTransient(parent Window, w Window) {
	parent.Context().transients[w] = "dummy"
}

func winMakePermanent(parent Window, w Window) {
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
			WinRemoveChild(parent, w)
			delete(wc.transients, w)
		}
	}
}

// WinGetSize xxx
func WinGetSize(w Window) (p image.Point) {
	return w.Context().currSz
}

// WinSetSize xxx
func WinSetSize(w Window, size image.Point) {
	w.Context().currSz = size
	// Don't do w.Do(), that would be recursive
}

// WinSetChildSize xxx
func WinSetChildSize(w Window, size image.Point) {
	if size.X == 0 || size.Y == 0 {
		hostwin.LogWarn("WinSetChildSize: too small, setting to 100,100")
		size = image.Point{100, 100}
	}
	w.Context().currSz = size
	w.Do(NewResizeCmd(size))
}

// WinSetChildPos xxx
func WinSetChildPos(parent Window, child Window, pos image.Point) {
	if parent == nil {
		hostwin.LogWarn("WinSeetChildPos: parent is nil?")
		return
	}
	parent.Context().childPos[child] = pos
}

// WinChildPos xxx
func WinChildPos(parent Window, child Window) (p image.Point) {
	if parent == nil {
		hostwin.LogWarn("WinChildPos: parent is nil?")
		return
	}
	childPos, ok := parent.Context().childPos[child]
	if !ok {
		hostwin.LogWarn("WinChildPos: w not in parent childPos?")
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
	currSize := WinGetSize(child)
	return image.Rectangle{Min: childPos, Max: childPos.Add(currSize)}
}

// WinChildName xxx
func WinChildName(parent Window, child Window) string {
	if parent == nil {
		hostwin.LogWarn("WinChildID: parent is nil?")
		return ""
	}
	id, ok := parent.Context().childName[child]
	if !ok {
		// hostwin.Warn("WinChildID: w not in parent childName?")
		return ""
	}
	return id
}

// WinChildNamed xxx
func WinChildNamed(parent Window, name string) Window {
	if parent == nil {
		hostwin.LogWarn("WinChildNamed: parent is nil?")
		return nil
	}
	for w, nm := range parent.Context().childName {
		if nm == name {
			return w
		}
	}
	hostwin.LogWarn("WinChildNamed: no child with name", "name", name)
	return nil
}

// WinMinSize xxx
func WinMinSize(w Window) (r image.Point) {
	return w.Context().minSize
}

// WinParent xxx
func WinParent(w Window) Window {
	parent := w.Context().parent
	if parent == nil {
		hostwin.LogWarn("Hey, why is WinParent being called for WorldWindow")
	}
	return parent
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
		hostwin.LogWarn("WinStye: using DefaultStyle because no parent", "w", w)
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
func WinForwardMouse(w Window, cmd hostwin.Cmd) {
	ddu := cmd.ValuesString("ddu", "")
	bnum := cmd.ValuesInt("buttonnum", 0)
	pos := cmd.ValuesPos(hostwin.PointZero)
	child, relPos := WinFindWindowUnder(w, pos)
	if child != nil {
		relcmd := NewMouseCmd(ddu, relPos, bnum)
		child.Do(relcmd)
	}
}
