package gui

import (
	"image"
	"image/color"
	"log"
	"time"
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
func NewMenu(parent Window) *Menu {
	m := &Menu{
		WindowData:   NewWindowData(parent),
		isPressed:    false,
		items:        make([]MenuItem, 0),
		itemSelected: -1,
		isTransient:  true,
	}
	log.Printf("NewMenu: go m.Run()\n")
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

// Draw xxx
func (menu *Menu) Draw() {

	menu.OutChan <- DrawFilledRectCmd{menu.Rect, color.RGBA{0, 0, 0, 0xff}}
	menu.OutChan <- DrawRectCmd{menu.Rect, color.RGBA{0xff, 0xff, 0xff, 0xff}}

	liney := menu.Rect.Min.Y + menu.handleHeight

	// Draw the bar at the top of the menu
	menu.OutChan <- DrawLineCmd{image.Point{menu.Rect.Min.X, liney}, image.Point{menu.Rect.Max.X, liney}, white}

	midx := menu.Rect.Max.X - menu.Rect.Dx()/4

	// Draw the X at the right side of the handle area
	menu.OutChan <- DrawLineCmd{image.Point{midx, menu.Rect.Min.Y}, image.Point{midx, liney}, white}
	// the midx-1 is just so the X looks a little nicer
	menu.OutChan <- DrawLineCmd{image.Point{midx - 1, menu.Rect.Min.Y}, image.Point{menu.Rect.Max.X, liney}, white}
	menu.OutChan <- DrawLineCmd{image.Point{midx - 1, liney}, image.Point{menu.Rect.Max.X, menu.Rect.Min.Y}, white}

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

			menu.OutChan <- DrawFilledRectCmd{itemRect, back}
		}

		menu.OutChan <- DrawTextCmd{item.label, menu.Style.fontFace, image.Point{item.posX, item.posY}, fore}
		if n != menu.itemSelected && n < (nitems-1) {
			menu.OutChan <- DrawLineCmd{image.Point{menu.Rect.Min.X, liney}, image.Point{menu.Rect.Max.X, liney}, fore}
		}
	}
}

// Run xxx
func (menu *Menu) Run() {

RunLoop:
	for {
		select {

		case cmd := <-menu.InChan:

			switch c := cmd.(type) {

			case CloseYourselfCmd:
				break RunLoop

			case MouseCmd:
				me := c
				if me.Pos.Y <= menu.Rect.Min.Y+menu.handleHeight {
					menu.itemSelected = -1
					continue RunLoop
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

					// tell parent to close me
					menu.OutChan <- CloseMeCmd{W: menu}

					// wait for CloseYourselfCmd to come back
					continue RunLoop
				}
			}
		default:
			time.Sleep(time.Millisecond)
		}
	}

	log.Printf("menu.Run: before CloseWindow\n")
	CloseWindow(menu)
	log.Printf("menu.Run: FINAL RETURN!\n")
}

func (menu *Menu) addItem(s string, cb MenuCallback) {
	menu.items = append(menu.items, MenuItem{
		label:    s,
		callback: cb,
	})
}
