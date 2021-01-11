package gui

import (
	"fmt"
	"image"
	"log"
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
		WindowData: WindowData{
			screen:  screen,
			style:   screen.style,
			rect:    image.Rectangle{},
			objects: make(map[string]Window),
		},
		console: nil,
	}

	// Add initial contents of RootWindow, for testing
	root.console = NewConsole(root)
	AddObject(root.objects, "console1", root.console)

	return root
}

// Data xxx
func (root *Root) Data() WindowData {
	return root.WindowData
}

// Resize xxx
func (root *Root) Resize(r image.Rectangle) image.Rectangle {
	root.rect = r
	// midy := (r.Min.Y + r.Max.Y) / 2

	r1 := r.Inset(20)
	r1.Min.Y = (r.Min.Y + r.Max.Y) / 2
	root.console.Resize(r1)

	if root.menu != nil {
		r1 := r.Inset(40)
		r1.Max.Y = (r.Min.Y + r.Max.Y) / 2
		root.menu.Resize(r1)
	}
	return root.rect
}

// Draw xxx
func (root *Root) Draw() {
	for _, o := range root.objects {
		o.Draw()
	}
}

// HandleMouseInput xxx
func (root *Root) HandleMouseInput(pos image.Point, button int, mdown bool) (rval bool) {
	o := ObjectUnder(root.objects, pos)
	if o != nil {
		// If a menu is up, and it's not this object, shut the menu
		if root.menu != nil && o != root.menu {
			RemoveObject(root.objects, "menu", root.menu)
		}
		rval = o.HandleMouseInput(pos, button, mdown)
	} else {
		// If it's a mouse click out in the open, nothing under it...
		if root.menu != nil {
			log.Printf("Removing old menu object before adding new one\n")
			RemoveObject(root.objects, "menu", root.menu)
			root.menu = nil
		} else {
			// No popup menu, create one on mousedown
			if mdown {
				root.menu = NewRootMenu(root)
				AddObject(root.objects, "menu", root.menu)
				mrect := image.Rect(pos.X, pos.Y, pos.X+200, pos.Y+200)
				root.menu.Resize(mrect)
				root.log(fmt.Sprintf("NewMenu: pos=%v\n", pos))
			}
		}
		rval = true
	}
	return rval
}

func (root *Root) log(s string) {
	root.console.AddLine(s)
}
