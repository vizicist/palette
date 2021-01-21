package gui

import (
	"image"
	"log"
)

// Page is the top-most Window
type Page struct {
	WindowData
	// console *Console
	// pagemenu Window // popup menu
}

// NewPageWindow xxx
func NewPageWindow(parent Window) Window {
	page := &Page{
		WindowData: NewWindowData(parent),
	}

	// Add initial contents of PageWindow, may go away once
	// windows are persistent
	// page.console = NewConsole(page)
	// AddChild(page, "console1", page.console)

	log.Printf("NewPageWindow: go page.Run()\n")
	// go page.readFromUpstream(page.fromUpstream)
	// go page.readFromChildren()

	return page
}

// Data xxx
func (page *Page) Data() *WindowData {
	return &page.WindowData
}

// DoDownstream xxx
func (page *Page) DoDownstream(t DownstreamCmd) {

	switch cmd := t.(type) {

	case ResizeCmd:
		page.resize(cmd.Rect)

	case RedrawCmd:
		RedrawChildren(page)

	case MouseCmd:
		page.mouseHandler(cmd)
	}
}

// DoUpstream xxx
func (page *Page) DoUpstream(w Window, t UpstreamCmd) {

	switch cmd := t.(type) {
	case CloseMeCmd:
		RemoveChild(page, cmd.W)
	default:
		page.parent.DoUpstream(w, t)
	}
}

func (page *Page) resize(r image.Rectangle) image.Rectangle {
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
	return page.Rect
}

func (page *Page) mouseHandler(cmd MouseCmd) {

	pos := cmd.Pos

	if !pos.In(page.Rect) {
		log.Printf("Page.Run: pos=%v not under Page!?", pos)
		return
	}

	o, _ := WindowUnder(page, pos)
	if o != nil {
		o.DoDownstream(cmd)
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
			pagemenu.DoDownstream(ResizeCmd{image.Rect(pos.X, pos.Y, pos.X+200, pos.Y+200)})
		}
	}
}
