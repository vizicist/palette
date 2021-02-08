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
	ctx           WinContext
	pageName      string
	lastPos       image.Point
	mouseHandler  MouseHandler
	cursorDrawer  CursorDrawer
	dragStart     image.Point // used for resize, move, etc
	targetWindow  Window      // used for resize, move, delete, etc
	currentAction string
	sweepToolName string
	pageMenu      Window
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
	return &page.ctx
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
		state, err := page.Do("dumpstate", fname)
		if err != nil {
			return nil, err
		}
		s := ToString(state)
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
		child := ToWindow(arg)
		RemoveChild(page, child)

	case "makepermanent":
		menu := ToMenu(arg)
		winMakePermanent(page, menu)

	case "closetransients":
		exceptMenu := ToWindow(arg)
		winRemoveTransients(page, exceptMenu)

	case "submenu":
		menuType := ToString(arg)
		pos := image.Point{lastMenuX + 4, page.lastPos.Y}
		_, err := page.AddTool(menuType, pos, image.Point{})
		if err != nil {
			log.Printf("Page.Do: submenu err=%s\n", err)
			return nil, err
		}

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
		RemoveChild(page, w)
	}

	var dat map[string]interface{}
	if err := json.Unmarshal([]byte(s), &dat); err != nil {
		return err
	}

	name := dat["page"].(string)
	sz := dat["size"].(string)
	size := StringToPoint(sz)

	log.Printf("restore name=%s size=%v\n", name, size)

	DoUpstream(page, "resizeme", size)

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
		winMakePermanent(page, childW)
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
		if child == doingMenu { // don't dump the menu used to do a dump
			continue
		}
		state, err := child.Do("dumpstate", nil)
		if err != nil {
			log.Printf("Page.dumpState: child=%d type=%T err=%s\n", wid, child, err)
			continue
		}
		s += fmt.Sprintf("%s{\n", sep)
		s += fmt.Sprintf("\"wid\": \"%d\",\n", wid)
		s += fmt.Sprintf("\"tooltype\": \"%s\",\n", ToolType(child))
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
	maker, ok := ToolMakers[name]
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
	DoUpstream(page, "setcolor", ForeColor)
	DoUpstream(page, "drawrect", page.sweepRect())
}

func (page *Page) drawSweepCursor() {
	DoUpstream(page, "setcolor", ForeColor)
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
	DoUpstream(page, "setcolor", ForeColor)
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

func (page *Page) resize() {
	size := WinCurrSize(page)
	WinSetMySize(page, size)
	log.Printf("Page.Resize: should be doing menus (and other things)?")
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

		pageMenuWasAlreadyUp := false
		if page.pageMenu != nil && winIsTransient(page, page.pageMenu) {
			pageMenuWasAlreadyUp = true
		}
		winRemoveTransients(page, nil) // remove all transients

		// Normally, pop up the PageMenu on MouseDown, but don't pop it up
		// if it was in the transients that were just removeda.
		if pageMenuWasAlreadyUp {
			page.pageMenu = nil
		} else {
			w, err := page.AddTool("PageMenu", mouse.Pos, image.Point{})
			if err != nil {
				log.Printf("defaultHandler: unable to create PageMenu err=%s\n", err)
			} else {
				page.pageMenu = w
			}
		}
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

		defer page.resetHandlers() // no matter what
		r := page.sweepRect()
		toolPos := r.Min
		toolSize := r.Max.Sub(r.Min)

		switch page.currentAction {

		case "resize":
			w := page.targetWindow
			if w == nil {
				log.Printf("Page.sweepHandler: w==nil?\n")
				return
			}
			// If you don't sweep out anything, look for
			// as much space as you can find starting at that point.
			if toolSize.X < 5 || toolSize.Y < 5 {
				under, _ := WindowUnder(page, toolPos)
				// If there's a window (other than the one we're resizing)
				// underneath the point, then don't do anything
				if under != nil && under != w {
					log.Printf("Page.sweepHandler: can't resize above a Window\n")
					return
				}
				foundRect := page.findSpace(r.Min, w)
				if foundRect.Dx()*foundRect.Dy() == 0 {
					// Last resort
					foundRect = image.Rectangle{Min: toolPos, Max: toolPos}.Inset(-20)
				}
				toolSize = foundRect.Max.Sub(foundRect.Min)
				toolPos = foundRect.Min
			}
			WinSetChildPos(page, w, toolPos)
			WinSetChildSize(w, toolSize)

		case "addtool":
			// If you don't sweep out anything, look for
			// as much space as you can find starting at that point.
			if toolSize.X < 5 || toolSize.Y < 5 {
				foundRect := page.findSpace(r.Min, nil)
				toolSize = foundRect.Max.Sub(foundRect.Min)
				toolPos = foundRect.Min
			}
			child, err := page.AddTool(page.sweepToolName, toolPos, toolSize)
			if err != nil {
				log.Printf("AddTool: err=%s\n", err)
			} else {
				WinSetChildSize(child, toolSize)
			}
		}
	}
}

func (page *Page) findSpace(p image.Point, ignore Window) image.Rectangle {

	rect := image.Rectangle{Min: p, Max: p}.Inset(-4)
	maxsz := page.ctx.currSz
	finalRect := rect
	dx := 1
	dy := 1

	// Expand as much as we can to the left
	for rect.Min.X > dx && page.anyOverlap(rect, ignore) == false {
		finalRect = rect
		rect.Min.X -= dx
	}
	if finalRect.Min.X > dx { // i.e. it overlapped something
		finalRect.Min.X += dx // back off a bit
	}

	rect = finalRect // start with the last successful one

	// Expand as much as we can to the right
	for rect.Max.X < (maxsz.X-dx) && page.anyOverlap(rect, ignore) == false {
		finalRect = rect
		rect.Max.X += dx
	}
	if finalRect.Max.X < (maxsz.X - dx) { // i.e. it overlapped something
		finalRect.Max.X -= dx // back off a bit
	}

	rect = finalRect // start with the last successful one

	// Expand as much as we can go up
	for rect.Min.Y > dy && page.anyOverlap(rect, ignore) == false {
		finalRect = rect
		rect.Min.Y -= dy
	}
	if finalRect.Min.Y > dy { // i.e. it overlapped something
		finalRect.Min.Y += dy // back off a bit
	}

	rect = finalRect // start with the last successful one

	// Expand as much as we can go down
	for rect.Max.Y < (maxsz.Y-dy) && page.anyOverlap(rect, ignore) == false {
		finalRect = rect
		rect.Max.Y += dy
	}
	if finalRect.Max.Y < (maxsz.Y - dy) { // i.e. it overlapped something
		finalRect.Max.Y -= dy // back off a bit
	}

	return finalRect
}

func (page *Page) anyOverlap(rect image.Rectangle, ignore Window) bool {
	for _, w := range page.ctx.childWindow {
		if w == ignore {
			continue
		}
		r := WinChildRect(page, w)
		intersect := r.Intersect(rect)
		if intersect.Dx() > 0 || intersect.Dy() > 0 {
			return true
		}
	}
	return false
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
			page.Do("makepermanent", page.targetWindow)
		}
		winRemoveTransients(page, nil)
		page.resetHandlers()

	case MouseDown:
		page.targetWindow, _ = WindowUnder(page, page.lastPos)
		page.dragStart = page.lastPos
	}
}
