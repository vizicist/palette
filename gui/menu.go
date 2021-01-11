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
	isPressed bool
	items     []MenuItem
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
		isPressed: false,
		items:     make([]MenuItem, 0),
	}
}

// Data xxx
func (menu *Menu) Data() WindowData {
	return menu.WindowData
}

// Resize xxx
func (menu *Menu) Resize(rect image.Rectangle) image.Rectangle {

	// Recompute all the y positions in the items
	textHeight := menu.style.TextHeight()

	// Adjust the rect so we're exactly that height
	rect.Max.Y = rect.Min.Y + len(menu.items)*textHeight
	menu.rect = rect
	log.Printf("menu.Resize: rect=%v\n", menu.rect)

	textx := menu.rect.Min.X
	for n, item := range menu.items {
		texty := menu.rect.Min.Y + n*textHeight
		brect := menu.style.BoundString(item.label)
		texty -= brect.Min.Y
		menu.items[n].posX = textx
		menu.items[n].posY = texty
	}
	return menu.rect
}

// Draw xxx
func (menu *Menu) Draw() {

	color := color.RGBA{0xff, 0xff, 0, 0xff}
	menu.screen.drawRect(menu.rect, color)

	for _, item := range menu.items {
		menu.screen.drawText(item.label, menu.style.fontFace, item.posX, item.posY, color)
	}
}

// HandleMouseInput xxx
func (menu *Menu) HandleMouseInput(pos image.Point, button int, mdown bool) bool {
	for _, item := range menu.items {
		if pos.Y < item.posY {
			if mdown {
				item.callback(item.label)
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
	menu.AddItem("Item1Ggy", func(name string) {
		log.Printf("Menu item1 callback!\n")
		menu.screen.log("Menu item1 callback")
	})
	menu.AddItem("Item2Ggy", func(name string) {
		log.Printf("Menu item2 callback!\n")
		menu.screen.log("Menu item2 callback")
	})
	return menu
}
