package gui

import (
	"encoding/json"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"runtime"
)

// MouseHandler xxx
type MouseHandler func(MouseCmd)

// CursorDrawer xxx
type CursorDrawer func()

// Page is the top-most Window
type Page struct {
	ctx           *WinContext
	pageName      string
	lastPos       image.Point
	mouseHandler  MouseHandler
	cursorDrawer  CursorDrawer
	dragStart     image.Point // used for resize, move, etc
	targetWindow  Window      // used for resize, move, delete, etc
	currentAction string
	sweepToolName string
}

// NewPage xxx
func NewPage(parent Window, name string) ToolData {
	page := &Page{
		ctx:      NewWindowContext(parent),
		pageName: name,
	}
	page.mouseHandler = page.defaultHandler

	return NewToolData(page, "Page", image.Point{640, 480})
}

// Context xxx
func (page *Page) Context() *WinContext {
	return page.ctx
}

// Do xxx
func (page *Page) Do(cmd string, arg interface{}) (interface{}, error) {

	// XXX - should be checking to verify that the from Window
	// is allowed to do the particular commands invoked.
	switch cmd {

	case "about":
		page.log("Palette Window System - version 0.1\n")
		page.log("# of goroutines: %d\n", runtime.NumGoroutine())

	case "restore":
		fname := ToString(arg)
		bytes, err := ioutil.ReadFile(fname)
		if err != nil {
			return nil, err
		}
		err = page.restoreState(string(bytes))
		if err != nil {
			return nil, err
		}

	case "dumptofile":
		fname := ToString(arg)
		output, err := page.Do("dumpstate", fname)
		if err != nil {
			return nil, err
		}
		s := ToString(output)
		ps := toPrettyJSON(s)
		err = ioutil.WriteFile(fname, []byte(ps), 0644)
		if err != nil {
			return nil, err
		}

	case "dumpstate":
		jdata, err := page.dumpState()
		if err != nil {
			return nil, err
		}
		return jdata, nil

	case "resize":
		page.resize()

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
		w := ToWindow(arg)
		menu := ToMenu(w)
		if menu != nil && menu.parentMenu != nil {
			// For menus, we want to close parent menus (if transient)
			pmenu := ToMenu(menu.parentMenu)
			if pmenu != nil && GetAttValue(pmenu, "istransient") == "true" {
				pmenu.Do("closeme", menu)
				RemoveChild(page, pmenu)
			}
		}
		RemoveChild(page, w)

	case "closesavemenu":
		pc := page.Context()
		menu := pc.saveMenu
		if menu != nil {
			RemoveChild(page, menu)
			pc.saveMenu = nil
		}

	case "submenu":
		menuType := ToString(arg)
		pos := image.Point{lastMenuX + 4, page.lastPos.Y}
		page.AddTool(menuType, pos, image.Point{})

	case "miscmenu":

	case "windowmenu":

	case "sweeptool":
		page.startSweep("addtool")
		page.cursorDrawer = page.drawSweepCursor
		page.sweepToolName = ToString(arg)
		page.showCursor(false)

	case "picktool":
		page.mouseHandler = page.pickHandler
		page.cursorDrawer = page.drawPickCursor
		page.currentAction = ToString(arg) // e.g. "resize", "delete"
		page.showCursor(false)

	case "movetool":
		page.mouseHandler = page.moveHandler
		page.cursorDrawer = page.drawPickCursor
		page.dragStart = page.lastPos
		// page.targetWindow = ToMenu(arg)
		page.showCursor(false)

	case "movemenu":
		page.mouseHandler = page.moveHandler
		page.cursorDrawer = page.drawPickCursor
		page.dragStart = page.lastPos
		page.targetWindow = ToMenu(arg)
		page.showCursor(false)

	default:
		DoUpstream(page, cmd, arg)
	}
	return nil, nil
}

func (page *Page) restoreState(s string) error {

	// clear this page of all children
	wc := page.Context()
	for _, w := range wc.childWindow {
		if GetAttValue(w, "istransient") != "true" {
			RemoveChild(page, w)
		}
	}

	var dat map[string]interface{}
	if err := json.Unmarshal([]byte(s), &dat); err != nil {
		return err
	}

	name := dat["page"].(string)
	sizeString := dat["size"].(string)
	log.Printf("restore name=%s sizeString=%s\n", name, sizeString)

	children := dat["children"].([]interface{})
	for _, ch := range children {

		childmap := ch.(map[string]interface{})
		toolType := childmap["tooltype"].(string)
		pos := StringToPoint(childmap["pos"].(string))
		size := StringToPoint(childmap["size"].(string))

		state := childmap["state"].(interface{})

		// Create the window
		childW, err := page.AddTool(toolType, pos, size)
		if err != nil {
			log.Printf("Page.restoreState: AddTool fails for toolType=%s, err=%s\n", toolType, err)
			continue
		}
		// restore state
		childW.Do("restore", state)
		childW.Do("resize", size)
		SetAttValue(childW, "istransient", "false")
	}

	return nil
}

func (page *Page) dumpState() (string, error) {

	wc := page.Context()
	s := "{\n"
	s += fmt.Sprintf("\"page\": \"%s\",\n", page.pageName)
	s += fmt.Sprintf("\"size\": \"%s\",\n", PointString(WinCurrSize(page)))
	s += fmt.Sprintf("\"children\": [\n") // start children array
	sep := ""
	for wid, child := range wc.childWindow {
		state, err := child.Do("dumpstate", nil)
		if err != nil {
			log.Printf("Page.dumpState: child=%d wintype=%s err=%s\n", wid, WindowType(child), err)
			continue
		}
		s += fmt.Sprintf("%s{\n", sep)
		s += fmt.Sprintf("\"wid\": \"%d\",\n", wid)
		s += fmt.Sprintf("\"tooltype\": \"%s\",\n", WinToolType(child))
		s += fmt.Sprintf("\"pos\": \"%s\",\n", PointString(WinChildPos(page, child)))
		s += fmt.Sprintf("\"size\": \"%s\",\n", PointString(WinCurrSize(child)))
		s += fmt.Sprintf("\"state\": %s\n", state)
		s += fmt.Sprintf("}\n")
		sep = ",\n"
	}
	s += fmt.Sprintf("]\n") // end of children array

	s += "}\n"

	return s, nil
}

// MakeTool xxx
func (page *Page) MakeTool(name string) (ToolData, error) {
	// style := WinStyle(page)
	maker, ok := Tools[name]
	if !ok {
		return ToolData{}, fmt.Errorf("MakeTool: There is no registered Tool named %s", name)
	}
	td := maker(page)
	if td.w == nil {
		return ToolData{}, fmt.Errorf("MakeTool: maker for Tool %s returns nil Window?", name)
	}
	return td, nil
}

// AddTool xxx
func (page *Page) AddTool(name string, pos image.Point, size image.Point) (Window, error) {

	td, err := page.MakeTool(name)
	if err != nil {
		return nil, err
	}

	child := AddChild(page, td)
	WinSetChildPos(page, child, pos)
	// If size isn't specified, use MinSize
	if size.X == 0 || size.Y == 0 {
		size = td.minSize
	}
	WinSetChildSize(child, size)
	return child, nil
}

func (page *Page) log(format string, v ...interface{}) {
	wc := page.Context()
	for _, w := range wc.childWindow {
		if GetAttValue(w, "islogger") == "true" {
			w.Do("addline", fmt.Sprintf(format, v...))
		}
	}
}

func (page *Page) drawSweepRect() {
	DoUpstream(page, "drawrect", page.sweepRect())
}

func (page *Page) drawSweepCursor() {
	pos := page.lastPos
	DoUpstream(page, "setcolor", ForeColor)
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
	DoUpstream(page, "setcolor", ForeColor)
	DoUpstream(page, "drawline", DrawLineCmd{
		XY0: pos.Add(image.Point{-sz, 0}),
		XY1: pos.Add(image.Point{sz, 0}),
	})
	DoUpstream(page, "drawline", DrawLineCmd{
		XY0: pos.Add(image.Point{0, -sz}),
		XY1: pos.Add(image.Point{0, sz}),
	})
}

func (page *Page) resize() {
	size := WinCurrSize(page)
	WinSetMySize(page, size)
	log.Printf("Page.Resize: should be doing menus (and other things)?")
}

// RelativePos xxx
func RelativePos(parent Window, w Window, pos image.Point) image.Point {
	childPos := WinChildPos(parent, w)
	relativePos := pos.Sub(childPos)
	return relativePos
}

func (page *Page) defaultHandler(mouse MouseCmd) {

	child, relPos := WindowUnder(page, mouse.Pos)
	if child != nil {
		if mouse.Ddu == MouseDown {
			WindowRaise(page, child)
		}
		mouse.Pos = relPos
		child.Do("mouse", mouse)
		return
	}

	// nothing underneath the mouse
	if mouse.Ddu == MouseDown {
		wc := page.Context()
		// Remove any transient windows (i.e. popup menus)
		for _, w := range wc.childWindow {
			if GetAttValue(w, "istransient") == "true" {
				RemoveChild(page, w)
			}
		}
		// pop up the PageMenu on Down
		page.AddTool("PageMenu", mouse.Pos, image.Point{})
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
			r := page.sweepRect()
			WinSetChildPos(page, w, r.Min)
			WinSetChildSize(w, r.Max.Sub(r.Min))
		case "addtool":
			r := page.sweepRect()
			sz := image.Point{r.Dx(), r.Dy()}
			child, err := page.AddTool(page.sweepToolName, r.Min, sz)
			if err != nil {
				log.Printf("AddTool: err=%s\n", err)
			} else {
				WinSetChildSize(child, r.Max.Sub(r.Min))
			}
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
		child, _ := WindowUnder(page, page.lastPos)
		if child == nil {
			page.resetHandlers()
			break
		}
		page.targetWindow = child
		switch page.currentAction {
		case "resize":
			page.startSweep("resize")
		case "delete":
			RemoveChild(page, child)
			page.resetHandlers()
		default:
			log.Printf("Unrecognized currentAction=%s\n", page.currentAction)
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
		page.targetWindow, _ = WindowUnder(page, page.lastPos)
		page.dragStart = page.lastPos
	}
}
