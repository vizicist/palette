package gui

import (
	"image"
	"image/color"
	"log"
)

// MenuCallback xxx
type MenuCallback func(item string)

// Menu xxx
type Menu struct {
	WindowData
	isPressed    bool
	items        []MenuItem
	itemSelected int
}

// MenuItem xxx
type MenuItem struct {
	label    string
	posX     int
	posY     int
	callback MenuCallback
}

// NewMenu xxx
func NewMenu(parent Window) *Menu {
	return &Menu{
		WindowData: WindowData{
			screen:  parent.Data().screen,
			style:   parent.Data().style,
			rect:    image.Rectangle{},
			objects: map[string]Window{},
		},
		isPressed:    false,
		items:        make([]MenuItem, 0),
		itemSelected: -1,
	}
}

// Data xxx
func (menu *Menu) Data() WindowData {
	return menu.WindowData
}

// Resize xxx
func (menu *Menu) Resize(rect image.Rectangle) image.Rectangle {

	// Recompute all the y positions in the items
	rowHeight := menu.style.TextHeight() + 4
	menu.rect = rect
	log.Printf("menu.Resize: rect=%v\n", menu.rect)

	textx := menu.rect.Min.X + 2
	maxX := -1
	for n, item := range menu.items {
		brect := menu.style.BoundString(item.label)
		if brect.Max.X > maxX {
			maxX = brect.Max.X
		}
		texty := menu.rect.Min.Y + n*rowHeight + 2 - brect.Min.Y
		menu.items[n].posX = textx
		menu.items[n].posY = texty
	}
	// Adjust the rect so we're exactly that height
	menu.rect.Max.Y = rect.Min.Y + len(menu.items)*rowHeight
	menu.rect.Max.X = menu.rect.Min.X + 4 + maxX
	return menu.rect
}

// Draw xxx
func (menu *Menu) Draw() {

	menu.screen.drawFilledRect(menu.rect, color.RGBA{0, 0, 0, 0xff})
	menu.screen.drawRect(menu.rect, color.RGBA{0xff, 0xff, 0xff, 0xff})

	for n, item := range menu.items {
		clr := color.RGBA{0xff, 0xff, 0xff, 0xff}
		if n == menu.itemSelected {
			clr = color.RGBA{0x00, 0xff, 0x00, 0xff}
		}
		menu.screen.drawText(item.label, menu.style.fontFace, item.posX, item.posY, clr)
	}
}

// HandleMouseInput xxx
func (menu *Menu) HandleMouseInput(pos image.Point, button int, event MouseEvent) bool {
	for n, item := range menu.items {
		if pos.Y < item.posY {
			switch event {
			case MouseDown:
				menu.itemSelected = n
				item.callback(item.label)
			case MouseDrag:
				menu.itemSelected = n
			case MouseUp:
				menu.itemSelected = -1
			}
			break
		}
	}
	return true
}

// AddItem xxx
func (menu *Menu) AddItem(s string, cb MenuCallback) {
	menu.items = append(menu.items, MenuItem{label: s, callback: cb})
}

// NewRootMenu xxx
func NewRootMenu(parent Window) *Menu {
	menu := NewMenu(parent)
	menu.AddItem("Move", func(name string) {
		menu.screen.log("Move callback")
	})
	menu.AddItem("Resize", func(name string) {
		menu.screen.log("Resize callback")
	})
	menu.AddItem("Delete", func(name string) {
		menu.screen.log("Delete callback")
	})
	menu.AddItem("Tools  ->", func(name string) {
		menu.screen.log("Tools callback")
	})
	menu.AddItem("Misc   ->", func(name string) {
		menu.screen.log("Misc callback")
	})
	return menu
}
