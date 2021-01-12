package gui

import (
	"image"
	"image/color"
)

// MenuCallback xxx
type MenuCallback func(item string)

// Menu xxx
type Menu struct {
	WindowData
	isPressed    bool
	items        []MenuItem
	itemSelected int
	rowHeight    int
	handleHeight int
	isTransient  bool
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
		isTransient:  true,
	}
}

// Data xxx
func (menu *Menu) Data() *WindowData {
	return &menu.WindowData
}

// Resize xxx
func (menu *Menu) Resize(rect image.Rectangle) image.Rectangle {

	// Recompute all the y positions in the items
	menu.rowHeight = menu.style.TextHeight() + 6
	menu.handleHeight = 9
	menu.rect = rect

	textx := menu.rect.Min.X + 6
	maxX := -1
	for n, item := range menu.items {
		brect := menu.style.BoundString(item.label)
		if brect.Max.X > maxX {
			maxX = brect.Max.X
		}
		// do not use Y info from brect,
		// the vertical placement of items should
		// be independent of the labels
		texty := menu.rect.Min.Y + 6 + menu.handleHeight + n*menu.rowHeight + (menu.rowHeight / 2)
		menu.items[n].posX = textx
		menu.items[n].posY = texty
	}
	// Adjust the rect so we're exactly the right height
	menu.rect.Max.Y = rect.Min.Y + len(menu.items)*menu.rowHeight + menu.handleHeight + 2
	menu.rect.Max.X = menu.rect.Min.X + 12 + maxX
	return menu.rect
}

// Draw xxx
func (menu *Menu) Draw() {

	menu.screen.drawFilledRect(menu.rect, color.RGBA{0, 0, 0, 0xff})
	menu.screen.drawRect(menu.rect, color.RGBA{0xff, 0xff, 0xff, 0xff})
	liney := menu.rect.Min.Y + menu.handleHeight

	// Draw the bar at the top of the menu
	menu.screen.drawLine(menu.rect.Min.X, liney, menu.rect.Max.X, liney, white)
	midx := menu.rect.Max.X - menu.rect.Dx()/4
	menu.screen.drawLine(midx, menu.rect.Min.Y, midx, liney, white)
	// the midx-1 is just so the X looks a little nicer
	menu.screen.drawLine(midx-1, menu.rect.Min.Y, menu.rect.Max.X, liney, white)
	menu.screen.drawLine(midx-1, liney, menu.rect.Max.X, menu.rect.Min.Y, white)

	white := color.RGBA{0xff, 0xff, 0xff, 0xff}
	black := color.RGBA{0x00, 0x00, 0x00, 0xff}
	// green := color.RGBA{0x00, 0xff, 0x00, 0xff}
	// red := color.RGBA{0xff, 0x00, 0x00, 0xff}
	nitems := len(menu.items)
	liney0 := menu.rect.Min.Y + menu.handleHeight + menu.rowHeight/4 - 4
	for n, item := range menu.items {
		fore := white
		back := black
		liney := liney0 + (n+1)*menu.rowHeight
		if n == menu.itemSelected {
			fore = black
			back = white
			itemRect := image.Rect(menu.rect.Min.X+1, liney-menu.rowHeight, menu.rect.Max.X-1, liney)
			menu.screen.drawFilledRect(itemRect, back)
		}

		menu.screen.drawText(item.label, menu.style.fontFace, item.posX, item.posY, fore)
		if n != menu.itemSelected && n < (nitems-1) {
			menu.screen.drawLine(menu.rect.Min.X, liney, menu.rect.Max.X, liney, fore)
		}
	}
}

// HandleMouseInput xxx
func (menu *Menu) HandleMouseInput(pos image.Point, button int, event MouseEvent) bool {
	for n, item := range menu.items {
		if pos.Y < item.posY {
			switch event {
			case MouseDown:
				menu.itemSelected = n
				// menu.screen.log(fmt.Sprintf("down itemSelected=%d", n))
				item.callback(item.label)
			case MouseDrag:
				menu.itemSelected = n
				// menu.screen.log(fmt.Sprintf("drag itemSelected=%d", n))
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
		menu.screen.log("Move callback, parent=%T", parent)
	})
	menu.AddItem("Resize", func(name string) {
		menu.screen.log("Resize callback")
	})
	menu.AddItem("Delete", func(name string) {
		menu.screen.log("Delete callback")
	})
	menu.AddItem("Tools->", func(name string) {
		menu.screen.log("Tools callback")
	})
	menu.AddItem("Misc ->", func(name string) {
		menu.screen.log("Misc callback")
	})
	return menu
}
