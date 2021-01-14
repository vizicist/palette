package gui

import (
	"image"
	"image/color"
	"runtime"
)

// Menu xxx
type Menu struct {
	WindowData
	isPressed    bool
	items        []MenuItem
	itemSelected int
	rowHeight    int
	handleHeight int
	isTransient  bool
	pickedWindow Window
	sweepBegin   image.Point
	sweepEnd     image.Point
	resizeState  int
	resizeChan   chan MouseEvent
}

// MenuCallback xxx
type MenuCallback func(item string)

// MenuItem xxx
type MenuItem struct {
	label    string
	posX     int
	posY     int
	callback MenuCallback
}

// NewMenu xxx
func NewMenu(parent Window) *Menu {
	m := &Menu{
		WindowData:   NewWindowData(parent),
		isPressed:    false,
		items:        make([]MenuItem, 0),
		itemSelected: -1,
		isTransient:  true,
	}
	go m.Run()
	return m
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

// Run xxx
func (menu *Menu) Run() {

	for {
		me := <-menu.mouseChan
		menu.screen.log("Menu.Run: me=%v", me)
		menu.screen.log("# of goroutines: %d", runtime.NumGoroutine())
		if me.pos.Y <= menu.rect.Min.Y+menu.handleHeight {
			menu.screen.log("Menu.Run: deselecting item")
			menu.itemSelected = -1
			continue
		}
		// Find out which item the mouse is in
		for n, item := range menu.items {
			if me.pos.Y < item.posY {
				menu.itemSelected = n
				menu.screen.log("Menu.Run: selected %d", n)
				break
			}
		}

		// No callbackups until MouseUp
		if me.ddu == MouseUp {
			item := menu.items[menu.itemSelected]
			menu.screen.log("Calling callback for %s", item.label)
			item.callback(item.label)
			menu.itemSelected = -1
		}
	}
}

/*
func (menu *Menu) mouseHandlerForPick(pos image.Point, button int, event MouseEvent) bool {
	item := menu.items[menu.itemSelected]
	if event == MouseDown {
		w := ObjectUnder(menu.screen.root, pos)
		menu.screen.log("mousedown, pos=%v pickedwindow=%v", pos, menu.pickedWindow)
		item.callbackPick(item.label, w)
	}
	if event == MouseUp {
		menu.screen.log("mouseup, RELEASING pos=%v pickedwindow=%v", pos, menu.pickedWindow)
		menu.screen.setDefaultMouseHandler()
	}
	return true
}
*/

func (menu *Menu) addItem(s string, cb MenuCallback) {
	menu.items = append(menu.items, MenuItem{
		label:    s,
		callback: cb,
	})
}
