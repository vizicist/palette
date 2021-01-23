package gui

import (
	"fmt"
	"image"
	"log"
)

// Page is the top-most Window
type Page struct {
	WindowData
	lastPos    image.Point
	sweepState int // 0: not sweeping, 1: looking for point 1, 2: looking for point 2
	sweepRect  image.Rectangle
}

// NewPageWindow xxx
func NewPageWindow(parent Window) *Page {
	page := &Page{
		WindowData: NewWindowData(parent),
	}
	return page
}

// Data xxx
func (page *Page) Data() *WindowData {
	return &page.WindowData
}

// DoSync xxx
func (page *Page) DoSync(from Window, cmd string, arg interface{}) (result interface{}, err error) {
	return nil, fmt.Errorf("Page has no DoSync commands")
}

// Do xxx
func (page *Page) Do(from Window, cmd string, arg interface{}) {

	switch cmd {
	case "resize":
		page.resize(ToRect(arg))

	case "redraw":
		RedrawChildren(page)

		// If we're sweeping, draw the sweep cursor or rectangle
		switch page.sweepState {
		case 1:
			page.drawSweepCursor(page.sweepRect.Min)
		case 2:
			// waiting for the MouseUp, draw the rectangle
			DoUpstream(page, "drawrect", page.sweepRect)
		}

	case "mouse":
		mouse := ToMouse(arg)
		page.lastPos = mouse.Pos
		if page.sweepState > 0 {
			page.sweepHandler(mouse)
		} else {
			page.mouseHandler(mouse)
		}

	case "closeme":
		RemoveChild(page, ToWindow(arg))

	case "startsweep":
		page.sweepState = 1
		page.sweepRect.Min = page.lastPos
		page.sweepRect.Max = page.lastPos
		DoUpstream(page, "showcursor", false)

	default:
		DoUpstream(page, cmd, arg)
	}
}

func (page *Page) drawSweepCursor(pos image.Point) {
	// waiting for the Mousedown, we draw the cursor
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

func (page *Page) resize(r image.Rectangle) {
	page.Rect = r
	// midy := (r.Min.Y + r.Max.Y) / 2

	/*
		r1 := r.Inset(20)
		r1.Min.Y = (r.Min.Y + r.Max.Y) / 2
		page.console.Resize(r1)
	*/

	log.Printf("Page.Resize: should be doing menus (and other things)?")
	/*
		if page.menu != nil {
			r1 := r.Inset(40)
			r1.Max.Y = (r.Min.Y + r.Max.Y) / 2
			page.menu.Resize(r1)
		}
	*/
}

func (page *Page) mouseHandler(cmd MouseCmd) {

	pos := cmd.Pos

	if !pos.In(page.Rect) {
		log.Printf("Page.Run: pos=%v not under Page!?", pos)
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

func (page *Page) sweepHandler(cmd MouseCmd) {

	switch cmd.Ddu {
	case MouseDown:
		if page.sweepState != 1 {
			log.Printf("Page.sweepHandler: unexpected sweepState != 1")
		} else {
			page.sweepRect.Min = cmd.Pos
			page.sweepRect.Max = cmd.Pos.Add(image.Point{20, 20})
			page.sweepState = 2
		}
	case MouseDrag:
		if page.sweepState <= 1 {
			page.sweepRect.Min = cmd.Pos
		} else {
			page.sweepRect.Max = cmd.Pos
		}
	case MouseUp:
		if page.sweepState != 2 {
			log.Printf("Page.sweepHandler: unexpected sweepState != 2")
		} else {
			page.sweepRect.Max = cmd.Pos
			log.Printf("MouseUp final sweepRect=%v\n", page.sweepRect)
		}
		page.sweepState = 0
		DoUpstream(page, "showcursor", true)

		console := NewConsole(page)
		AddChild(page, "console", console)
		console.Do(page, "resize", page.sweepRect)
	}
}

// NoSyncInterface xxx
func NoSyncInterface(name string) (result interface{}, err error) {
	return nil, fmt.Errorf("DoSync() in %s has no commands", name)
}
