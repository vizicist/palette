package gui

import (
	"image"
)

// Root is the top-most Window
type Root struct {
	WindowData
	console *Console
	menu    *Menu // popup menu
}

// NewRoot xxx
func NewRoot(screen *Screen) *Root {
	root := &Root{
		WindowData: WindowData{ // exception to using NewWindowData
			screen:    screen,
			style:     screen.style,
			rect:      image.Rectangle{},
			objects:   map[string]Window{},
			mouseChan: make(chan MouseEvent), // XXX - unbuffered?
		},
		console: nil,
	}

	// Add initial contents of RootWindow, may go away once
	// windows are persistent
	root.console = NewConsole(root)
	AddObject(root, "console1", root.console)

	go root.Run()

	return root
}

// Data xxx
func (root *Root) Data() *WindowData {
	return &root.WindowData
}

// Resize xxx
func (root *Root) Resize(r image.Rectangle) image.Rectangle {
	root.rect = r
	// midy := (r.Min.Y + r.Max.Y) / 2

	r1 := r.Inset(20)
	r1.Min.Y = (r.Min.Y + r.Max.Y) / 2
	root.console.Resize(r1)

	root.screen.log("Root.Resize: should be doing menus (and other things)?")
	/*
		if root.menu != nil {
			r1 := r.Inset(40)
			r1.Max.Y = (r.Min.Y + r.Max.Y) / 2
			root.menu.Resize(r1)
		}
	*/
	return root.rect
}

// Draw xxx
func (root *Root) Draw() {
	data := root.Data()
	// Draw windows in reverse order so more recent ones are on top.
	for n, name := range data.order {
		if name == "" {
			root.screen.log("Hey, name %d in data.order is empty?", n)
		} else {
			w := data.objects[name]
			w.Draw()
		}
	}
}

// Run xxx
func (root *Root) Run() {

	for {
		me := <-root.mouseChan
		// root.screen.log("Root.Run: me=%v", me)
		pos := me.pos
		if !pos.In(root.rect) {
			root.screen.log("pos=%v not under Root", pos)
			continue
		}

		o := ObjectUnder(root, pos)
		if o != nil {
			/*
				// If a menu is up, and it's not this object, shut the menu
				if root.menu != nil && o != root.menu {
					root.screen.log("Removing object")
					RemoveObject(root, "menu")
					root.screen.log("After Removing object")
					root.menu = nil
				}
			*/

			// root.screen.log("Sending me to o.Data.mouseChan me=%v", me)
			o.Data().mouseChan <- me
			// root.screen.log("Afte Sending me to o.Data.mouseChan me=%v", me)
		}

		if me.ddu == MouseDown {
			// If it's a mouse down out in the open, nothing under it...
			// Ignore Drag and Up, but pop up the RootMenu on Down
			if root.menu != nil {
				root.screen.log("Out in open, with existing menu!")
			} else {
				root.menu = NewRootMenu(root)
				AddObject(root, "menu", root.menu)
				root.menu.Resize(image.Rect(pos.X, pos.Y, pos.X+200, pos.Y+200))
			}
		}
	}
}

func (root *Root) log(s string) {
	root.console.AddLine(s)
}
