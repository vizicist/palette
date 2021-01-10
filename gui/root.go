package gui

import (
	"image"
	"log"
)

// Root xxx
type Root struct {
	WindowData
	console  *Console
	console2 *Console
	menu     *Menu // popup menu
}

// NewRoot xxx
func NewRoot(style *Style) *Root {
	w := &Root{
		WindowData: WindowData{
			style:   style,
			rect:    image.Rectangle{},
			objects: make(map[string]Window),
		},
		console: nil,
	}
	return w
}

// Data xxx
func (root *Root) Data() WindowData {
	return root.WindowData
}

// Resize xxx
func (root *Root) Resize(r image.Rectangle) {
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
}

// Draw xxx
func (root *Root) Draw(screen *Screen) {
	for _, o := range root.objects {
		o.Draw(screen)
	}
}

// HandleMouseInput xxx
func (root *Root) HandleMouseInput(pos image.Point, button int, mdown bool) (rval bool) {
	o := ObjectUnder(root.objects, pos)
	if o != nil {
		// XXX - If a menu is up, and this object isn't that menu,
		// then shut the menu
		if root.menu != nil && o != root.menu {
			RemoveObject(root.objects, "menu", root.menu)
		}
		rval = o.HandleMouseInput(pos, button, mdown)
	} else {
		if root.menu != nil {
			log.Printf("Removing old menu object before adding new one\n")
			RemoveObject(root.objects, "menu", root.menu)
		}
		root.menu = NewMenu(root.style)
		root.menu.AddItem("item1")
		root.menu.AddItem("item2")
		AddObject(root.objects, "menu", root.menu)
		mrect := image.Rect(pos.X, pos.Y, pos.X+200, pos.Y+200)
		root.menu.Resize(mrect)
		rval = true
	}
	return rval
}
