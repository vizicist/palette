package gui

import (
	"fmt"
	"image"
	"log"
	"reflect"
	"runtime"
	"strings"
)

// MouseHandler xxx
type MouseHandler func(MouseCmd)

// CursorDrawer xxx
type CursorDrawer func()

// Page is the top-most Window
type Page struct {
	WindowData
	lastPos       image.Point
	mouseHandler  MouseHandler
	cursorDrawer  CursorDrawer
	dragStart     image.Point // used for resize, move, etc
	targetWindow  Window      // used for resize, move, delete, etc
	currentAction string
	sweepToolName string
}

// NewPageWindow xxx
func NewPageWindow(parent Window) *Page {
	page := &Page{
		WindowData: NewWindowData(parent),
	}
	page.mouseHandler = page.defaultHandler
	return page
}

// Data xxx
func (page *Page) Data() *WindowData {
	if page.parent == nil {
		page.log("Hey, parent is nil?\n")
	}
	return &page.WindowData
}

// DoSync xxx
func (page *Page) DoSync(from Window, cmd string, arg interface{}) (result interface{}, err error) {
	return nil, fmt.Errorf("Page has no DoSync commands")
}

// Do xxx
func (page *Page) Do(from Window, cmd string, arg interface{}) {

	if page.parent == nil {
		page.log("Hey, parent is nil?\n")
	}

	// XXX - should be checking to verify that the from Window
	// is allowed to do the particular commands invoked.
	switch cmd {
	case "about":
		page.log("This is the Palette Window System\n")
		page.log("# of goroutines: %d\n", runtime.NumGoroutine())

	case "resize":
		page.resize(ToRect(arg))

	case "redraw":
		RedrawChildren(page)
		if page.cursorDrawer != nil {
			page.cursorDrawer()
		}

	case "mouse":
		mouse := ToMouse(arg)
		page.lastPos = mouse.Pos
		page.mouseHandler(mouse)

	case "closeme":
		RemoveChild(page, ToWindow(arg))

	case "toolsmenu":
		page.log("toolsmenu start")
		SetAttValue(from, "istransient", "false")
		toolsmenu := NewToolsMenu(page)
		AddChild(page, toolsmenu)
		// Place it just to the right of the window that spawned it
		pos := image.Point{from.Data().Rect.Max.X + 2, page.lastPos.Y}
		toolsmenu.Do(page, "resize", image.Rect(pos.X, pos.Y, pos.X+200, pos.Y+200))

	case "miscmenu":
		page.log("miscmenu start")

	case "windowmenu":
		page.log("windowmenu start")

	case "sweeptool":
		page.startSweep("addtool")
		page.sweepToolName = ToString(arg)
		page.showCursor(false)

	case "picktool":
		page.mouseHandler = page.pickHandler
		page.cursorDrawer = page.drawPickCursor
		page.currentAction = ToString(arg)
		page.showCursor(false)

	case "movetool":
		page.mouseHandler = page.moveHandler
		page.cursorDrawer = page.drawPickCursor
		page.showCursor(false)

	case "movemenu":
		log.Printf("movemenu start!\n")
		page.mouseHandler = page.moveHandler
		page.cursorDrawer = page.drawPickCursor
		page.dragStart = page.lastPos
		page.targetWindow = ToMenu(arg)
		page.showCursor(false)

	default:
		if page.parent == nil {
			page.log("Hey, parent is nil?\n")
		}

		DoUpstream(page, cmd, arg)
	}
}

// AddTool xxx
func (page *Page) AddTool(name string, rect image.Rectangle) Window {

	tool := page.NewTool(name)
	if tool == nil {
		page.log("Unable to create a new Tool named %s\n", name)
		return nil
	}
	w := AddChild(page, tool)
	if w != nil {
		w.Do(page, "resize", page.sweepRect())
	}
	return w
}

// NewTool xxx
func (page *Page) NewTool(name string) Window {
	// var t Page
	// methodName := "New" + name
	capName := strings.ToUpper(string(name[0])) + name[1:]
	methodName := "New" + capName + "Tool"
	toolArgs := make([]reflect.Value, 0)
	toolArgs = append(toolArgs, reflect.ValueOf("foo"))
	method := reflect.ValueOf(page).MethodByName(methodName)
	if !method.IsValid() {
		page.log("NewTool: no method in Page named %s\n", methodName)
		return nil
	}
	retval := method.Call(toolArgs)
	if len(retval) == 0 {
		page.log("NewTool: method %s didn't return anything!?\n", methodName)
		return nil
	}
	ret := retval[0].Interface()
	tool, ok := ret.(Window)
	if !ok {
		page.log("NewTool: method %s didn't return a Window!\n", methodName)
		return nil
	}
	return tool
}

// NewConsoleTool xxx
func (page *Page) NewConsoleTool(arg string) Window {
	w := NewConsole(page)
	return w
}

func (page *Page) log(format string, v ...interface{}) {
	for _, w := range page.children {
		if GetAttValue(w, "islogger") == "true" {
			w.Do(page, "addline", fmt.Sprintf(format, v...))
		}
	}
}

func (page *Page) drawSweepRect() {
	DoUpstream(page, "drawrect", page.sweepRect())
}

func (page *Page) drawSweepCursor() {
	pos := page.lastPos
	DoUpstream(page, "drawline", DrawLineCmd{
		XY0: pos, XY1: pos.Add(image.Point{20, 0}),
	})
	DoUpstream(page, "drawline", DrawLineCmd{
		XY0: pos, XY1: pos.Add(image.Point{0, 20}),
	})
	pos = pos.Add(image.Point{10, 10})
	DoUpstream(page, "drawline", DrawLineCmd{
		XY0: pos, XY1: pos.Add(image.Point{10, 0}),
	})
	DoUpstream(page, "drawline", DrawLineCmd{
		XY0: pos, XY1: pos.Add(image.Point{0, 10}),
	})
}

func (page *Page) drawPickCursor() {
	sz := 10
	pos := page.lastPos
	DoUpstream(page, "drawline", DrawLineCmd{
		XY0: pos.Add(image.Point{-sz, 0}),
		XY1: pos.Add(image.Point{sz, 0}),
	})
	DoUpstream(page, "drawline", DrawLineCmd{
		XY0: pos.Add(image.Point{0, -sz}),
		XY1: pos.Add(image.Point{0, sz}),
	})
}

func (page *Page) resize(r image.Rectangle) {
	page.Rect = r
	page.log("Page.Resize: should be doing menus (and other things)?")
}

func (page *Page) defaultHandler(cmd MouseCmd) {

	pos := cmd.Pos

	if !pos.In(page.Rect) {
		page.log("Page.Run: pos=%v not under Page!?", pos)
		return
	}

	o := WindowUnder(page, pos)
	if o != nil {
		o.Do(page, "mouse", cmd)
		return
	}

	// nothing underneath the mouse
	if cmd.Ddu == MouseDown {
		// Remove any transient windows (i.e. popup menus)
		for _, w := range page.children {
			if GetAttValue(w, "istransient") == "true" {
				RemoveChild(page, w)
			}
		}
		// pop up the PageMenu on Down,
		rect := image.Rect(pos.X, pos.Y, pos.X+200, pos.Y+200)
		AddChild(page, NewPageMenu(page)).Do(page, "resize", rect)
	}
}

func (page *Page) sweepRect() image.Rectangle {
	r := image.Rectangle{}
	r.Min = page.dragStart
	r.Max = page.lastPos
	return r
}

func (page *Page) sweepHandler(cmd MouseCmd) {

	switch cmd.Ddu {
	case MouseDown:
		page.dragStart = cmd.Pos
		page.cursorDrawer = page.drawSweepRect
	case MouseDrag:
		// do nothing
	case MouseUp:
		switch page.currentAction {
		case "resize":
			w := page.targetWindow
			w.Do(page, "resize", page.sweepRect())

		case "addtool":
			page.AddTool(page.sweepToolName, page.sweepRect())
		}
		page.resetHandlers()
	}
}

func (page *Page) resetHandlers() {
	page.mouseHandler = page.defaultHandler
	page.cursorDrawer = nil
	page.targetWindow = nil
	page.showCursor(true)
}

func (page *Page) showCursor(b bool) {
	DoUpstream(page, "showcursor", b)
}

func (page *Page) startSweep(action string) {
	page.mouseHandler = page.sweepHandler
	page.cursorDrawer = page.drawSweepCursor
	page.currentAction = action
	page.dragStart = page.lastPos
}

// This Handler waits until MouseUp
func (page *Page) pickHandler(cmd MouseCmd) {

	switch cmd.Ddu {
	case MouseDown:
		page.targetWindow = nil // to make it clear until we get MouseUp
	case MouseDrag:
	case MouseUp:
		w := WindowUnder(page, page.lastPos)
		if w == nil {
			page.resetHandlers()
			break
		}
		page.targetWindow = w
		switch page.currentAction {
		case "resize":
			page.startSweep("resize")
		case "delete":
			RemoveChild(page, w)
			page.resetHandlers()
		default:
			page.log("Unrecognized currentAction=%s\n", page.currentAction)
		}
	}
}

// This Handler starts doing things on MouseDown
func (page *Page) moveHandler(cmd MouseCmd) {

	switch cmd.Ddu {

	case MouseDrag:
		if page.targetWindow != nil {
			dpos := page.lastPos.Sub(page.dragStart)
			MoveWindow(page, page.targetWindow, dpos)
			page.dragStart = page.lastPos
		}

	case MouseUp:
		// When we move a menu, we want it to be permanent
		if page.targetWindow != nil {
			SetAttValue(page.targetWindow, "istransient", "false")
		}
		page.resetHandlers()

	case MouseDown:
		log.Printf("moveHandler cmd=%v\n", cmd)
		page.targetWindow = WindowUnder(page, page.lastPos)
		page.dragStart = page.lastPos
	}
}

// NoSyncInterface xxx
func NoSyncInterface(name string) (result interface{}, err error) {
	return nil, fmt.Errorf("DoSync() in %s has no commands", name)
}
