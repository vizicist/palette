package gui

import (
	"image"
)

// Page is the top-most Window
type Page struct {
	WindowData
	console *Console
	menu    *Menu // popup menu
}

// NewPageWindow xxx
func NewPageWindow(screen *Screen) *Page {
	page := &Page{
		WindowData: WindowData{ // can't use NewWindowData, no Window parent
			Screen:    screen,
			Style:     screen.style,
			Rect:      image.Rectangle{},
			objects:   map[string]Window{},
			MouseChan: make(chan MouseMsg), // XXX - unbuffered?
			KeyChan:   make(chan KeyMsg),   // XXX - unbuffered?
			CmdChan:   make(chan CmdMsg),   // XXX - unbuffered?
		},
		// console: nil,
	}

	// Add initial contents of PageWindow, may go away once
	// windows are persistent
	page.console = NewConsole(page)
	AddWindow(page, "console1", page.console)

	go page.Run()

	return page
}

// Data xxx
func (page *Page) Data() *WindowData {
	return &page.WindowData
}

// Resize xxx
func (page *Page) Resize(r image.Rectangle) image.Rectangle {
	page.Rect = r
	// midy := (r.Min.Y + r.Max.Y) / 2

	r1 := r.Inset(20)
	r1.Min.Y = (r.Min.Y + r.Max.Y) / 2
	page.console.Resize(r1)

	page.Screen.Log("Page.Resize: should be doing menus (and other things)?")
	/*
		if page.menu != nil {
			r1 := r.Inset(40)
			r1.Max.Y = (r.Min.Y + r.Max.Y) / 2
			page.menu.Resize(r1)
		}
	*/
	return page.Rect
}

// Draw xxx
func (page *Page) Draw() {
	data := page.Data()
	// Draw windows in reverse order so more recent ones are on top.
	for n, name := range data.order {
		if name == "" {
			page.Screen.Log("Hey, name %d in data.order is empty?", n)
		} else {
			w := data.objects[name]
			w.Draw()
		}
	}
}

// Run xxx
func (page *Page) Run() {

	for {
		me := <-page.MouseChan
		// page.Screen.Log("Page.Run: me=%v", me)
		pos := me.Pos
		if !pos.In(page.Rect) {
			page.Screen.Log("pos=%v not under Page", pos)
			continue
		}

		o := ObjectUnder(page, pos)
		if o != nil {
			o.Data().MouseChan <- me
		}

		if me.Ddu == MouseDown {
			// If it's a mouse down out in the open, nothing under it...
			// Ignore Drag and Up, but pop up the PageMenu on Down
			menuName := "main"
			if page.menu != nil {
				page.Screen.Log("Out in open, with existing menu!")
				RemoveWindow(page, menuName)
				page.menu = nil
			} else {
				page.menu = NewPageMenu(page)
				AddWindow(page, menuName, page.menu)
				page.menu.Resize(image.Rect(pos.X, pos.Y, pos.X+200, pos.Y+200))
			}
		}
	}
}

func (page *Page) log(s string) {
	// log.Printf("Page.log: %s\n", s)
	page.console.AddLine(s)
}
