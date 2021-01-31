package gui

import (
	"encoding/json"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"runtime"
	"strings"
)

// MouseHandler xxx
type MouseHandler func(MouseCmd)

// CursorDrawer xxx
type CursorDrawer func()

// Page is the top-most Window
type Page struct {
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
func NewPage(parent Window, name string) *Page {
	style := WinStyle(parent)
	minRect := image.Rect(0, 0, 640, 480)

	wdata := NewWindowData(parent, minRect, style)
	page := &Page{
		pageName: name,
	}
	log.Printf("NewPage: page=%p  wdata.parent=%p\n", page, wdata.parent)
	WinMap[page] = wdata
	page.mouseHandler = page.defaultHandler
	return page
}

// Do xxx
func (page *Page) Do(from Window, cmd string, arg interface{}) (interface{}, error) {

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
		output, err := page.Do(from, "dumpstate", fname)
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
		w := ToWindow(arg)
		menu := ToMenu(w)
		if menu != nil && menu.parentMenu != nil {
			// For menus, we want to close parent menus (if transient)
			pmenu := ToMenu(menu.parentMenu)
			if pmenu != nil && GetAttValue(pmenu, "istransient") == "true" {
				pmenu.Do(menu, "closeme", menu)
				RemoveChild(page, pmenu)
			}
		}
		RemoveChild(page, w)

	case "toolsmenu":

		rect := WindowRect(from)
		pos := image.Point{rect.Max.X + 2, page.lastPos.Y}
		childRect := image.Rect(pos.X, pos.Y, pos.X+200, pos.Y+200)
		page.AddTool("ToolsMenu", childRect)

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
		page.currentAction = ToString(arg)
		page.showCursor(false)

	case "movetool":
		page.mouseHandler = page.moveHandler
		page.cursorDrawer = page.drawPickCursor
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

// toPrettyJSON looks for {} and [] at the beginning/end of lines
// to control indenting
func toPrettyJSON(s string) string {
	// Make sure it ends in \n
	if s[len(s)-1] != '\n' {
		s = s + "\n"
	}
	from := 0
	indentSize := 4
	indent := 0
	r := ""
	slen := len(s)
	for from < slen {
		newlinePos := from + strings.Index(s[from:], "\n")
		line := s[from : newlinePos+1]

		// ignore initial whitespace, we're going to handle it
		for line[0] == ' ' {
			line = line[1:]
		}

		// See if we should un-indent before adding the line
		firstChar := line[0]
		if firstChar == '}' || firstChar == ']' {
			indent -= indentSize
			if indent < 0 {
				indent = 0
			}
		}

		// Add line with current indent
		r += strings.Repeat(" ", indent) + line

		// See if we should indent the next line more
		lastChar := s[newlinePos-1]
		if lastChar == '{' || lastChar == '[' {
			indent += indentSize
		}

		from = newlinePos + 1
	}
	return r
}

// RectString xxx
func RectString(r image.Rectangle) string {
	return fmt.Sprintf("%d,%d,%d,%d", r.Min.X, r.Min.Y, r.Max.X, r.Max.Y)
}

// StringRect xxx
func StringRect(s string) (r image.Rectangle) {
	n, err := fmt.Sscanf(s, "%d,%d,%d,%d", &r.Min.X, &r.Min.Y, &r.Max.X, &r.Max.Y)
	if err != nil {
		log.Printf("Bad Rect format: %s\n", s)
		return image.Rectangle{}
	}
	if n != 4 {
		log.Printf("Bad Rect format: %s\n", s)
		return image.Rectangle{}
	}
	return r
}

func (page *Page) restoreState(s string) error {

	// clear this page of all children
	wd := WinData(page)
	for _, w := range wd.children {
		if GetAttValue(w, "istransient") != "true" {
			RemoveChild(page, w)
		}
	}

	var dat map[string]interface{}
	if err := json.Unmarshal([]byte(s), &dat); err != nil {
		return err
	}

	name := dat["page"].(string)
	rectString := dat["rect"].(string)
	log.Printf("restore name=%s rectString=%s\n", name, rectString)

	children := dat["children"].([]interface{})
	for _, ch := range children {
		childmap := ch.(map[string]interface{})
		// wid := childmap["wid"].(string)
		childType := childmap["type"].(string)
		rstr := childmap["rect"].(string)
		childRect := StringRect(rstr)
		childState := childmap["state"].(interface{})
		// Create the window
		childW := page.AddTool(childType, childRect)
		if childW == nil {
			log.Printf("Hey, AddTool fails for childType=%s\n", childType)
			continue
		}
		// restore state
		childW.Do(page, "restore", childState)
		SetAttValue(childW, "istransient", "false")
		childW.Do(page, "resize", childRect)
	}

	return nil
}

func (page *Page) dumpState() (string, error) {

	wd := WinData(page)
	s := "{\n"
	s += fmt.Sprintf("\"page\": \"%s\",\n", page.pageName)
	s += fmt.Sprintf("\"rect\": \"%s\",\n", RectString(wd.rect))
	s += fmt.Sprintf("\"children\": [\n") // start children array
	sep := ""
	for wid, child := range wd.children {
		state, err := child.Do(page, "dumpstate", nil)
		if err != nil {
			log.Printf("Page.dumpState: child=%d wintype=%s err=%s\n", wid, WindowType(child), err)
			continue
		}
		s += fmt.Sprintf("%s{\n", sep)
		s += fmt.Sprintf("\"wid\": \"%d\",\n", wid)
		s += fmt.Sprintf("\"type\": \"%s\",\n", WindowType(child))
		s += fmt.Sprintf("\"rect\": \"%s\",\n", RectString(WinRect(child)))
		s += fmt.Sprintf("\"state\": %s\n", state)
		s += fmt.Sprintf("}\n")
		sep = ",\n"
	}
	s += fmt.Sprintf("]\n") // end of children array

	s += "}\n"

	return s, nil
}

// AddTool xxx
func (page *Page) AddTool(name string, rect image.Rectangle) Window {

	tool := page.NewTool(name)
	if tool == nil {
		log.Printf("Unable to create a new Tool named %s\n", name)
		return nil
	}
	w := AddChild(page, tool)
	if w != nil {
		w.Do(page, "resize", rect)
	}
	return w
}

// NewTool xxx
func (page *Page) NewTool(name string) Window {

	maker, ok := Tools[name]
	style := WinStyle(page)
	if ok {
		w, minRect := maker(style)
		WinMap[w] = NewWindowData(page, minRect, style)
		log.Printf("w=%v\n", w)
		return w
	}
	log.Printf("NewTool: There is no registered Tool named %s\n", name)
	return nil

	/*
		// var t Page
		capName := strings.ToUpper(string(name[0])) + name[1:]
		methodName := "New" + capName + "Window"
		toolArgs := make([]reflect.Value, 0)
		toolArgs = append(toolArgs, reflect.ValueOf("foo"))
		method := reflect.ValueOf(page).MethodByName(methodName)
		if !method.IsValid() {
			log.Printf("NewTool: no method in Page named %s\n", methodName)
			return nil
		}
		retval := method.Call(toolArgs)
		if len(retval) == 0 {
			log.Printf("NewTool: method %s didn't return anything!?\n", methodName)
			return nil
		}
		ret := retval[0].Interface()
		tool, ok := ret.(Window)
		if !ok {
			log.Printf("NewTool: method %s didn't return a Window!\n", methodName)
			return nil
		}
		return tool
	*/
}

func (page *Page) log(format string, v ...interface{}) {
	wd := WinData(page)
	for _, w := range wd.children {
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

func (page *Page) resize(r image.Rectangle) {
	wd := WinData(page)
	wd.rect = r
	log.Printf("Page.Resize: should be doing menus (and other things)?")
}

func (page *Page) defaultHandler(cmd MouseCmd) {

	wd := WinData(page)
	pos := cmd.Pos

	if !pos.In(wd.rect) {
		log.Printf("Page.Run: pos=%v not under Page!?", pos)
		return
	}

	o := WindowUnder(page, pos)
	if o != nil {
		if cmd.Ddu == MouseDown {
			WindowRaise(page, o)
		}
		o.Do(page, "mouse", cmd)
		return
	}

	// nothing underneath the mouse
	if cmd.Ddu == MouseDown {
		// Remove any transient windows (i.e. popup menus)
		for _, w := range wd.children {
			if GetAttValue(w, "istransient") == "true" {
				RemoveChild(page, w)
			}
		}
		// pop up the PageMenu on Down,
		rect := image.Rect(pos.X, pos.Y, pos.X+200, pos.Y+200)

		log.Printf("page.AddTool: before adding PageMenu, page.parent = %p\n", wd.parent)
		page.AddTool("PageMenu", rect)

		log.Printf("page.AddTool: after adding PageMenu, page.parent = %p\n", wd.parent)

		// pg := NewPageMenu(page)
		// child := AddChild(page, pg)
		// child.Do(page, "resize", rect)
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
			WinData(w).rect = r
			w.Do(page, "resize", r)
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
		page.targetWindow = WindowUnder(page, page.lastPos)
		page.dragStart = page.lastPos
	}
}
