package gui

import (
	"image"
	"image/color"
	"log"
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
func NewMenu(parent Window, items []MenuItem) Window {
	m := &Menu{
		WindowData:   NewWindowData(parent),
		isPressed:    false,
		items:        items,
		itemSelected: -1,
		isTransient:  true,
	}
	return m
}

// Data xxx
func (menu *Menu) Data() *WindowData {
	return &menu.WindowData
}

// DoUpstream xxx
func (menu *Menu) DoUpstream(w Window, cmd UpstreamCmd) {
}

// Resize xxx
func (menu *Menu) resize(rect image.Rectangle) image.Rectangle {

	// Recompute all the y positions in the items
	menu.rowHeight = menu.Style.TextHeight() + 6
	menu.handleHeight = 9
	menu.Rect = rect

	textx := menu.Rect.Min.X + 6
	maxX := -1
	for n, item := range menu.items {
		brect := menu.Style.BoundString(item.label)
		if brect.Max.X > maxX {
			maxX = brect.Max.X
		}
		// do not use Y info from brect,
		// the vertical placement of items should
		// be independent of the labels
		texty := menu.Rect.Min.Y + 6 + menu.handleHeight + n*menu.rowHeight + (menu.rowHeight / 2)
		menu.items[n].posX = textx
		menu.items[n].posY = texty
	}
	// Adjust the rect so we're exactly the right height
	menu.Rect.Max.Y = rect.Min.Y + len(menu.items)*menu.rowHeight + menu.handleHeight + 2
	menu.Rect.Max.X = menu.Rect.Min.X + 12 + maxX
	return menu.Rect
}

// Upstream xxx
func (menu *Menu) Upstream(cmd UpstreamCmd) {
	menu.parent.DoUpstream(menu, cmd)
}

// Draw xxx
func (menu *Menu) redraw() {

	menu.Upstream(DrawRectCmd{menu.Rect, color.RGBA{0xff, 0xff, 0xff, 0xff}})
	menu.Upstream(DrawFilledRectCmd{menu.Rect.Inset(4), color.RGBA{0, 0, 0, 0xff}})

	liney := menu.Rect.Min.Y + menu.handleHeight

	// Draw the bar at the top of the menu
	menu.Upstream(DrawLineCmd{image.Point{menu.Rect.Min.X, liney}, image.Point{menu.Rect.Max.X, liney}, white})

	midx := menu.Rect.Max.X - menu.Rect.Dx()/4

	// Draw the X at the right side of the handle area
	menu.Upstream(DrawLineCmd{image.Point{midx, menu.Rect.Min.Y}, image.Point{midx, liney}, white})
	// the midx-1 is just so the X looks a little nicer
	menu.Upstream(DrawLineCmd{image.Point{midx - 1, menu.Rect.Min.Y}, image.Point{menu.Rect.Max.X, liney}, white})
	menu.Upstream(DrawLineCmd{image.Point{midx - 1, liney}, image.Point{menu.Rect.Max.X, menu.Rect.Min.Y}, white})

	nitems := len(menu.items)
	liney0 := menu.Rect.Min.Y + menu.handleHeight + menu.rowHeight/4 - 4
	for n, item := range menu.items {
		fore := white
		back := black
		liney := liney0 + (n+1)*menu.rowHeight
		if n == menu.itemSelected {
			fore = black
			back = white
			itemRect := image.Rect(menu.Rect.Min.X+1, liney-menu.rowHeight, menu.Rect.Max.X-1, liney)

			menu.Upstream(DrawFilledRectCmd{itemRect, back})
		}

		menu.Upstream(DrawTextCmd{item.label, menu.Style.fontFace, image.Point{item.posX, item.posY}, fore})
		if n != menu.itemSelected && n < (nitems-1) {
			menu.Upstream(DrawLineCmd{image.Point{menu.Rect.Min.X, liney}, image.Point{menu.Rect.Max.X, liney}, fore})
		}
	}
}

func (menu *Menu) mouseHandler(cmd MouseCmd) (didCallback bool) {
	me := cmd
	if me.Pos.Y <= menu.Rect.Min.Y+menu.handleHeight {
		menu.itemSelected = -1
		return false
	}
	// Find out which item the mouse is in
	for n, item := range menu.items {
		if me.Pos.Y < item.posY {
			menu.itemSelected = n
			break
		}
	}

	// No callbackups until MouseUp
	if me.Ddu == MouseUp {
		item := menu.items[menu.itemSelected]
		item.callback(item.label)
		menu.itemSelected = -1

		return true
	}
	return false
}

// DoDownstream xxx
func (menu *Menu) DoDownstream(t DownstreamCmd) {

	switch cmd := t.(type) {

	case CloseYourselfCmd:
		log.Printf("menu.DoDownstream: CloseYourselfCmd\n")

	case ResizeCmd:
		menu.resize(cmd.Rect)

	case RedrawCmd:
		menu.redraw()

	case MouseCmd:
		didCallback := menu.mouseHandler(cmd)
		if didCallback {
			menu.parent.DoUpstream(menu, CloseMeCmd{W: menu})
		}
	}
}

func (menu *Menu) addItem(s string, cb MenuCallback) {
	menu.items = append(menu.items, MenuItem{
		label:    s,
		callback: cb,
	})
}
