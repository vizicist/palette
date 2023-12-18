package twinsys

import (
	"encoding/json"
	"fmt"
	"image"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/vizicist/palette/kit"
)

// MouseHandler xxx
type MouseHandler func(kit.Cmd)

// MouseDrawer xxx
type MouseDrawer func()

// Page is the top-most Window
type Page struct {
	ctx              WinContext
	pageName         string
	lastPos          image.Point
	mouseHandler     MouseHandler
	mouseDrawer      MouseDrawer
	dragStart        image.Point // used for resize, move, etc
	targetWindowName string      // used for resize, move, delete, etc
	currentAction    string
	sweepToolName    string
	pageMenu         Window
	logWriter        io.Writer
	logFile          *os.File
}

// MakePermanentMsg xxx
type MakePermanentMsg struct {
	W Window
}

// NewPage xxx
func NewPage(parent Window, name string) WindowData {
	page := &Page{
		ctx:              NewWindowContext(parent),
		pageName:         name,
		lastPos:          image.Point{},
		mouseHandler:     nil,
		mouseDrawer:      nil,
		dragStart:        image.Point{},
		targetWindowName: "",
		currentAction:    "",
		sweepToolName:    name,
		pageMenu:         nil,
		logWriter:        nil,
		logFile:          &os.File{},
	}
	page.mouseHandler = page.defaultHandler
	// page.logWriter = PageLogWriter{page: page}
	// log.SetOutput(page.logWriter)

	// Putting things in page.log is a last-resort,
	// since sometimes there's no Console up
	path := kit.LogFilePath("page.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("NewPage: Unable to open %s err=%s", path, err)
	}
	page.logFile = f // could be nil
	kit.LogInfo("Page logs are being saved in", "path", path)

	return NewToolData(page, "Page", image.Point{640, 480})
}

// PageLogWriter xxx
type PageLogWriter struct {
	page *Page
}

func (w PageLogWriter) Write(p []byte) (n int, err error) {
	s := string(p)
	newline := ""
	if !strings.HasSuffix(s, "\n") {
		newline = "\n"
	}
	final := fmt.Sprintf("%s%s", s, newline)
	w.page.log(final)
	os.Stderr.Write([]byte(final))

	w.page.logToFile(final) // last resort, if no "islogger" windows up

	return len(p), nil
}

func (page *Page) logToFile(s string) {
	if page.logFile != nil {
		_, err := page.logFile.WriteString(s)
		kit.LogIfError(err)
	}
}

// Context xxx
func (page *Page) Context() *WinContext {
	return &page.ctx
}

// Do xxx
func (page *Page) Do(cmd kit.Cmd) string {

	// XXX - should be checking to verify that the from Window
	// is allowed to do the particular commands invoked.

	switch cmd.Subj {

	case "log":
		text := cmd.ValuesString("text", "")
		page.log("%s\n", text)

	case "about":
		page.log("Palette Window System - version 0.1\n")
		page.log("# of goroutines: %d\n", runtime.NumGoroutine())

	case "restorefile":
		fname := cmd.ValuesString("filename", "")
		bytes, err := os.ReadFile(fname)
		if err != nil {
			return kit.ErrorResult(err.Error())
		}
		err = page.restoreState(string(bytes))
		if err != nil {
			return kit.ErrorResult(err.Error())
		}

	case "dumpfile":
		fname := cmd.ValuesString("filename", "")
		retmsg := page.Do(kit.NewSimpleCmd("getstate"))
		kit.LogInfo("dumpfile message", "retmsg", retmsg)
		m, _ := kit.StringMap(retmsg)
		state, ok := m["state"]
		if !ok {
			kit.LogWarn("DumpFileMsg didn't receive valid StateDatamsg")
		} else {
			ps := toPrettyJSON(state)
			fpath := kit.ConfigFilePath(fname)
			kit.LogIfError(os.WriteFile(fpath, []byte(ps), 0644))
		}

	case "getstate":
		state, err := page.dumpState()
		kit.LogIfError(err)
		if err == nil {
			return state
		}

	case "resize":
		page.resize()

	case "redraw":
		WinRedrawChildren(page)
		if page.mouseDrawer != nil {
			page.mouseDrawer()
		}

	case "mouse":
		page.lastPos = cmd.ValuesPos(kit.PointZero)
		page.mouseHandler(cmd)

	case "closeme":
		w := WinChildNamed(page, cmd.ValuesString("window", ""))
		WinRemoveChild(page, w)

	case "makepermanent":
		w := WinChildNamed(page, cmd.ValuesString("window", ""))
		winMakePermanent(page, w)

	case "closetransients":
		exceptMenuName := cmd.ValuesString("exceptmenu", "")
		var exceptWindow Window
		if exceptMenuName != "" {
			exceptWindow = WinChildNamed(page, exceptMenuName)
		}
		winRemoveTransients(page, exceptWindow)

	case "submenu":
		submenutype := cmd.ValuesString("submenu", "")
		pos := image.Point{lastMenuX + 4, page.lastPos.Y}
		_, err := page.AddTool(submenutype, pos, image.Point{})
		if err != nil {
			kit.LogIfError(err)
		}

	case "sweeptool":
		page.startSweep("addtool")
		page.mouseDrawer = page.drawSweepMouse
		toolname := cmd.ValuesString("toolname", "")
		page.sweepToolName = toolname
		page.showMouse(false)

	case "picktool":
		page.mouseHandler = page.pickHandler
		page.mouseDrawer = page.drawPickMouse
		action := cmd.ValuesString("action", "")
		page.currentAction = action // e.g. "resize", "delete"
		page.showMouse(false)

	case "movetool":
		page.mouseHandler = page.moveHandler
		page.mouseDrawer = page.drawPickMouse
		page.dragStart = page.lastPos
		// Do I care what value targetWindow has?
		// page.targetWindow = ToMenu(arg)
		page.showMouse(false)

	case "movemenu":
		page.mouseHandler = page.moveHandler
		page.mouseDrawer = page.drawPickMouse
		page.dragStart = page.lastPos
		menuName := cmd.ValuesString("menu", "")
		page.targetWindowName = menuName
		page.showMouse(false)

	default:
		WinDoUpstream(page, cmd)
	}
	return kit.OkResult()
}

func (page *Page) restoreState(s string) error {

	// clear this page of all children
	wc := page.Context()
	for _, w := range wc.childWindow {
		WinRemoveChild(page, w)
	}

	var dat map[string]any
	// var dat map[string]string
	if err := json.Unmarshal([]byte(s), &dat); err != nil {
		return err
	}

	name := dat["page"].(string)
	sz := dat["size"].(string)
	size := StringToPoint(sz)

	kit.LogInfo("restore", "name", name, "size", size)

	WinDoUpstream(page, NewResizeMeCmd(size))

	kit.LogWarn("HEY!! restoreState needs work!")

	children := dat["children"].([]any)
	for _, ch := range children {

		childmap := ch.(map[string]any)
		toolType := childmap["tooltype"].(string)
		pos := StringToPoint(childmap["pos"].(string))
		size := StringToPoint(childmap["size"].(string))
		state := childmap["state"].(string)

		// Create the window
		childW, err := page.AddTool(toolType, pos, size)
		if err != nil {
			kit.LogIfError(err)
			continue
		}
		// restore state
		childW.Do(NewRestoreCmd(state))
		childW.Do(NewResizeCmd(size))
		winMakePermanent(page, childW)
	}

	return nil
}

func (page *Page) dumpState() (string, error) {

	wc := page.Context()
	s := "{\n"
	s += fmt.Sprintf("\"page\": \"%s\",\n", page.pageName)
	s += fmt.Sprintf("\"size\": \"%s\",\n", PointString(WinGetSize(page)))
	s += "\"children\": [\n" // start children array
	sep := ""
	for wid, child := range wc.childWindow {
		if child == doingMenu { // don't dump the menu used to do a dump
			continue
		}
		state := child.Do(kit.NewSimpleCmd("getstate"))

		s += fmt.Sprintf("%s{\n", sep)
		s += fmt.Sprintf("\"wid\": \"%s\",\n", wid)
		s += fmt.Sprintf("\"tooltype\": \"%s\",\n", winToolType(child))
		s += fmt.Sprintf("\"pos\": \"%s\",\n", PointString(WinChildPos(page, child)))
		s += fmt.Sprintf("\"size\": \"%s\",\n", PointString(WinGetSize(child)))
		s += fmt.Sprintf("\"state\": %s\n", state)
		s += "}\n"
		sep = ",\n"
	}
	s += "]\n" // end of children array

	s += "}\n"

	return s, nil
}

// MakeTool xxx
func (page *Page) MakeTool(name string) (WindowData, error) {
	maker, ok := WindowMakers[name]
	if !ok {
		return WindowData{}, fmt.Errorf("MakeTool: There is no registered Tool named %s", name)
	}
	td := maker(page)
	if td.w == nil {
		return WindowData{}, fmt.Errorf("MakeTool: maker for Tool %s returns nil Window?", name)
	}
	return td, nil
}

// AddTool xxx
func (page *Page) AddTool(name string, pos image.Point, size image.Point) (Window, error) {

	td, err := page.MakeTool(name)
	if err != nil {
		return nil, err
	}

	child := WinAddChild(page, td)
	WinSetChildPos(page, child, pos)
	// If size isn't specified, use MinSize
	if size.X == 0 || size.Y == 0 {
		size = td.minSize
	}
	WinSetChildSize(child, size)
	kit.LogInfo("Page.AddTool", "name", name, "pos", pos, "size", size)
	return child, nil
}

func (page *Page) log(format string, v ...any) {
	wc := page.Context()
	for _, w := range wc.childWindow {
		if WinGetAttValue(w, "islogger") == "true" {
			line := fmt.Sprintf(format, v...)
			w.Do(NewAddLineCmd(line))
		}
	}
}

func (page *Page) drawSweepRect() {
	WinDoUpstream(page, NewSetColorCmd(ForeColor))
	WinDoUpstream(page, NewDrawRectCmd(page.sweepRect()))
}

func (page *Page) drawSweepMouse() {
	WinDoUpstream(page, NewSetColorCmd(ForeColor))
	pos := page.lastPos
	WinDoUpstream(page, NewDrawLineCmd(pos, pos.Add(image.Point{20, 0})))
	WinDoUpstream(page, NewDrawLineCmd(pos, pos.Add(image.Point{0, 20})))
	pos = pos.Add(image.Point{10, 10})
	WinDoUpstream(page, NewDrawLineCmd(pos, pos.Add(image.Point{10, 0})))
	WinDoUpstream(page, NewDrawLineCmd(pos, pos.Add(image.Point{0, 10})))
}

func (page *Page) drawPickMouse() {
	WinDoUpstream(page, NewSetColorCmd(ForeColor))
	sz := 10
	pos := page.lastPos
	WinDoUpstream(page, NewDrawLineCmd(
		pos.Add(image.Point{-sz, 0}),
		pos.Add(image.Point{sz, 0})))
	WinDoUpstream(page, NewDrawLineCmd(
		pos.Add(image.Point{0, -sz}),
		pos.Add(image.Point{0, sz})))
}

func (page *Page) resize() {
	size := WinGetSize(page)
	WinSetSize(page, size)
	kit.LogWarn("Page.Resize: should be doing menus (and other things)?")
}

func (page *Page) defaultHandler(cmd kit.Cmd) {

	pos := cmd.ValuesPos(kit.PointZero)
	ddu := cmd.ValuesString("ddu", "")

	child, relPos := WinFindWindowUnder(page, pos)
	if child != nil {
		if ddu == "down" {
			winRaise(page, child)
		}
		cmd.ValuesSetPos(relPos)
		child.Do(cmd)
		return
	}

	// nothing underneath the mouse
	if ddu == "down" {

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
			w, err := page.AddTool("PageMenu", pos, image.Point{})
			if err != nil {
				kit.LogIfError(err)
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

func (page *Page) sweepHandler(cmd kit.Cmd) {

	ddu := cmd.ValuesString("ddu", "")
	switch ddu {

	case "down":
		pos := cmd.ValuesPos(kit.PointZero)
		page.dragStart = pos
		page.mouseDrawer = page.drawSweepRect

	case "drag":
		// do nothing

	case "up":

		defer page.resetHandlers() // no matter what
		r := page.sweepRect()
		toolPos := r.Min
		toolSize := r.Max.Sub(r.Min)

		switch page.currentAction {

		case "resize":
			wname := page.targetWindowName
			if wname == "" {
				kit.LogWarn("Page.sweepHandler: no targetWindow?")
				return
			}
			w := WinChildNamed(page, wname)
			if w == nil {
				kit.LogWarn("Page.sweepHandler: no window with", "name", wname)
				return
			}
			// If you don't sweep out anything, look for
			// as much space as you can find starting at that point.
			if toolSize.X < 5 || toolSize.Y < 5 {
				under, _ := WinFindWindowUnder(page, toolPos)
				// If there's a window (other than the one we're resizing)
				// underneath the point, then don't do anything
				if under != nil && under != w {
					kit.LogWarn("Page.sweepHandler: can't resize above a Window")
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
				kit.LogIfError(err)
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
	for rect.Min.X > dx && !page.anyOverlap(rect, ignore) {
		finalRect = rect
		rect.Min.X -= dx
	}
	if finalRect.Min.X > dx { // i.e. it overlapped something
		finalRect.Min.X += dx // back off a bit
	}

	rect = finalRect // start with the last successful one

	// Expand as much as we can to the right
	for rect.Max.X < (maxsz.X-dx) && !page.anyOverlap(rect, ignore) {
		finalRect = rect
		rect.Max.X += dx
	}
	if finalRect.Max.X < (maxsz.X - dx) { // i.e. it overlapped something
		finalRect.Max.X -= dx // back off a bit
	}

	rect = finalRect // start with the last successful one

	// Expand as much as we can go up
	for rect.Min.Y > dy && !page.anyOverlap(rect, ignore) {
		finalRect = rect
		rect.Min.Y -= dy
	}
	if finalRect.Min.Y > dy { // i.e. it overlapped something
		finalRect.Min.Y += dy // back off a bit
	}

	rect = finalRect // start with the last successful one

	// Expand as much as we can go down
	for rect.Max.Y < (maxsz.Y-dy) && !page.anyOverlap(rect, ignore) {
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
	page.mouseDrawer = nil
	page.targetWindowName = ""
	page.showMouse(true)
}

func (page *Page) showMouse(b bool) {
	WinDoUpstream(page, NewShowMouseCursorCmd(b))
}

func (page *Page) startSweep(action string) {
	page.mouseHandler = page.sweepHandler
	page.mouseDrawer = page.drawSweepMouse
	page.currentAction = action
	page.dragStart = page.lastPos
}

// This Handler waits until MouseUp
func (page *Page) pickHandler(cmd kit.Cmd) {

	ddu := cmd.ValuesString("ddu", "")
	switch ddu {
	case "down":
		page.targetWindowName = "" // to make it clear until we get MouseUp
	case "drag":
	case "up":
		child, _ := WinFindWindowUnder(page, page.lastPos)
		if child == nil {
			page.resetHandlers()
			break
		}
		childName := WinChildName(page, child)
		page.targetWindowName = childName
		switch page.currentAction {
		case "resize":
			page.startSweep("resize")
		case "delete":
			WinRemoveChild(page, child)
			page.resetHandlers()
		default:
			kit.LogWarn("Unrecognized action", "currentAction", page.currentAction)
		}
	}
}

// This Handler starts doing things on MouseDown
func (page *Page) moveHandler(cmd kit.Cmd) {

	ddu := cmd.ValuesString("ddu", "")
	switch ddu {

	case "drag":
		if page.targetWindowName != "" {
			dpos := page.lastPos.Sub(page.dragStart)
			wtarget := WinChildNamed(page, page.targetWindowName)
			winMoveWindow(page, wtarget, dpos)
			page.dragStart = page.lastPos
		}

	case "up":
		// When we move a menu, we want it to be permanent
		var exceptWindow Window
		if page.targetWindowName != "" {
			page.Do(NewMakePermanentCmd(page.targetWindowName))
			exceptWindow = WinChildNamed(page, page.targetWindowName)
		}
		winRemoveTransients(page, exceptWindow)
		page.resetHandlers()

	case "down":
		targetWindow, _ := WinFindWindowUnder(page, page.lastPos)
		page.targetWindowName = WinChildName(page, targetWindow)
		page.dragStart = page.lastPos
	}
}
