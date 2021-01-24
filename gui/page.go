package gui

import (
	"fmt"
	"image"
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
	lastPos      image.Point
	mouseHandler MouseHandler
	cursorDrawer CursorDrawer
	sweepStart   image.Point
	sweepAction  string
	sweepArg     string
	sweepTool    string
	pickAction   string
	pickWindow   Window
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
		toolsmenu := NewToolsMenu(page)
		AddChild(page, "toolsmenu", toolsmenu)
		pos := page.lastPos
		toolsmenu.Do(page, "resize", image.Rect(pos.X, pos.Y, pos.X+200, pos.Y+200))

	case "miscmenu":
		page.log("miscmenu start")

	case "windowmenu":
		page.log("windowmenu start")

	case "sweeptool":
		page.startSweep("addtool")
		page.sweepArg = ToString(arg)
		DoUpstream(page, "showcursor", false)

	case "picktool":
		page.mouseHandler = page.pickHandler
		page.cursorDrawer = page.drawPickCursor
		page.pickAction = ToString(arg)
		switch page.pickAction {
		case "resize":
		default:
			page.log("Unrecognized pickAction=%s\n", page.pickAction)
		}
		DoUpstream(page, "showcursor", false)

	default:
		if page.parent == nil {
			page.log("Hey, parent is nil?\n")
		}

		DoUpstream(page, cmd, arg)
	}
}

// AddTool xxx
func (page *Page) AddTool(toolName string, instanceName string, rect image.Rectangle) Window {

	tool := page.NewTool(toolName)
	if tool == nil {
		page.log("Unable to create a new Tool named %s\n", toolName)
		return nil
	}
	w := AddChild(page, instanceName, tool)
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
	w := FindChild(page, "console")
	if w != nil {
		s := fmt.Sprintf(format, v...)
		w.Do(page, "addline", s)
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
	// midy := (r.Min.Y + r.Max.Y) / 2

	/*
		r1 := r.Inset(20)
		r1.Min.Y = (r.Min.Y + r.Max.Y) / 2
		page.console.Resize(r1)
	*/

	page.log("Page.Resize: should be doing menus (and other things)?")
	/*
		if page.menu != nil {
			r1 := r.Inset(40)
			r1.Max.Y = (r.Min.Y + r.Max.Y) / 2
			page.menu.Resize(r1)
		}
	*/
}

func (page *Page) defaultHandler(cmd MouseCmd) {

	pos := cmd.Pos

	if !pos.In(page.Rect) {
		page.log("Page.Run: pos=%v not under Page!?", pos)
		return
	}

	o, _ := WindowUnder(page, pos)
	if o != nil {
		o.Do(page, "mouse", cmd)
		return
	}

	// nothing underneath the mouse
	if cmd.Ddu == MouseDown {
		// pop up the PageMenu on Down
		menuName := "pagemenu"
		child, ok := page.children[menuName]
		if ok {
			// We only want one pagemenu on the screen at a time.
			RemoveChild(page, child)
		} else {
			pagemenu := NewPageMenu(page)
			// page.pagemenu = pagemenu
			AddChild(page, menuName, pagemenu)
			pagemenu.Do(page, "resize", image.Rect(pos.X, pos.Y, pos.X+200, pos.Y+200))
		}
	}
}

func (page *Page) sweepRect() image.Rectangle {
	r := image.Rectangle{}
	r.Min = page.sweepStart
	r.Max = page.lastPos
	return r
}

func (page *Page) sweepHandler(cmd MouseCmd) {

	switch cmd.Ddu {
	case MouseDown:
		page.sweepStart = cmd.Pos
		page.cursorDrawer = page.drawSweepRect
	case MouseDrag:
		// do nothing
	case MouseUp:
		page.resetHandlers()

		switch page.sweepAction {
		case "resize":
			w := page.pickWindow
			w.Do(page, "resize", page.sweepRect())

		case "addtool":
			if page.sweepArg == "console" {
				consoleInstance := "console"
				// If there's already a console window, close it
				w := FindChild(page, consoleInstance)
				if w != nil {
					w.Do(page, "closeyourself", nil)
					RemoveChild(page, w)
				}
			}

			page.AddTool("console", page.sweepArg, page.sweepRect())
		}
	}
}

func (page *Page) resetHandlers() {
	page.mouseHandler = page.defaultHandler
	page.cursorDrawer = nil
	DoUpstream(page, "showcursor", true)
}

func (page *Page) startSweep(action string) {
	page.mouseHandler = page.sweepHandler
	page.cursorDrawer = page.drawSweepCursor
	page.sweepAction = action
	page.sweepStart = page.lastPos
}

func (page *Page) pickHandler(cmd MouseCmd) {

	switch cmd.Ddu {
	case MouseDown:
	case MouseDrag:
	case MouseUp:
		page.resetHandlers()
		w, _ := WindowUnder(page, page.lastPos)
		if w != nil {
			page.pickWindow = w
			switch page.pickAction {
			case "resize":
				page.startSweep("resize")
			default:
				page.log("Unrecognized pickAction=%s\n", page.pickAction)
			}
		}
	}
}

// NoSyncInterface xxx
func NoSyncInterface(name string) (result interface{}, err error) {
	return nil, fmt.Errorf("DoSync() in %s has no commands", name)
}
