package gui

import (
	"image"
	"log"
	"time"
)

// Page is the top-most Window
type Page struct {
	WindowData
	// console *Console
	menu *Menu // popup menu
}

// NewPageWindow xxx
func NewPageWindow(screen *Screen) *Page {
	// Normally, Windows use NewWindowData(), but
	// the top-level page Window is special.
	page := &Page{
		WindowData: WindowData{ //
			InChan:  make(chan WinInput, 10),
			OutChan: screen.cmdChan,
			Style:   screen.style,
			Rect:    image.Rectangle{},
			windows: make(map[Window]string),
		},
		// console: nil,
	}

	// Add initial contents of PageWindow, may go away once
	// windows are persistent

	// page.console = NewConsole(page)
	// AddWindow(page, "console1", page.console)

	log.Printf("NewPageWindow: go page.Run()\n")
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

// Draw xxx
func (page *Page) Draw() {
	data := page.Data()
	for _, win := range data.order {
		win.Draw()
	}
}

// Run xxx
func (page *Page) Run() {

	for {

		// log.Printf("page.Run: top of loop, page.Input=%v\n", page.InChan)

		select {

		case inCmd := <-page.InChan:

			switch t := inCmd.(type) {

			case ResizeCmd:
				log.Printf("page.ResizeCmd=%v", t)

			case RedrawCmd:
				log.Printf("page.RedrawCmd=%v", t)

			case MouseCmd:
				me := t
				pos := me.Pos

				// log.Printf("page got MouseCmd = %v\n", me)

				if !pos.In(page.Rect) {
					log.Printf("Page.Run: pos=%v not under Page!?", pos)
					continue
				}

				o := WindowUnder(page, pos)
				if o != nil {
					o.Data().InChan <- me
					continue
				}

				// nothing underneath the mouse
				if me.Ddu == MouseDown {
					// pop up the PageMenu on Down
					menuName := "pagemenu"
					if page.menu != nil {

						// We only want one page.menu on the screen at a time.
						log.Printf("page.Run: sending CloseMe for page.menu = %v\n", page.menu)
						page.OutChan <- CloseMeCmd{W: page.menu}
						page.menu = nil
					} else {
						page.menu = NewPageMenu(page)
						log.Printf("page.Run: New page.menu = %v\n", page.menu)
						AddWindow(page, page.menu, menuName)
						page.menu.Resize(image.Rect(pos.X, pos.Y, pos.X+200, pos.Y+200))
					}
				}
			}

		default:
			// log.Printf("Select default!\n")
			time.Sleep(time.Millisecond)
		}
	}
}

func (page *Page) log(s string) {
	// log.Printf("Page.log: %s\n", s)
	log.Printf("%s\n", s)
	// page.console.AddLine(s)
}
