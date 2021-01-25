package gui

import (
	"image"
	"image/color"
	"log"
	"strings"
)

// Menu xxx
type Menu struct {
	WindowData
	isPressed    bool
	items        []MenuItem
	itemSelected int
	rowHeight    int
	handleHeight int
	handleMidx   int
	pickedWindow Window
	sweepBegin   image.Point
	sweepEnd     image.Point
	resizeState  int
	parentMenu   Window
}

// MenuCallback xxx
type MenuCallback func(item string)

// MenuItem xxx
type MenuItem struct {
	label  string
	posX   int
	posY   int
	target Window
	cmd    string
	arg    interface{}
}

// NewMenu xxx
func NewMenu(parent Window, parentMenu Window) *Menu {
	m := &Menu{
		WindowData:   NewWindowData(parent),
		isPressed:    false,
		items:        make([]MenuItem, 0),
		itemSelected: -1,
		parentMenu:   parentMenu,
	}
	SetAttValue(m, "istransient", "true")

	return m
}

// Data xxx
func (menu *Menu) Data() *WindowData {
	return &menu.WindowData
}

// SetItems xxx
func (menu *Menu) SetItems(items []MenuItem) {
	menu.items = items
}

// Resize xxx
func (menu *Menu) resize(rect image.Rectangle) {

	// Recompute all the y positions in the items
	// DO NOT use Style.RowHeight, we want wider spacing
	menu.rowHeight = menu.Style.TextHeight() + 6
	menu.handleHeight = 9

	textx := rect.Min.X + 6
	maxX := -1
	for n, item := range menu.items {
		brect := menu.Style.BoundString(item.label)
		if brect.Max.X > maxX {
			maxX = brect.Max.X
		}
		// do not use Y info from brect,
		// the vertical placement of items should
		// be independent of the labels
		texty := rect.Min.Y + 6 + menu.handleHeight + n*menu.rowHeight + (menu.rowHeight / 2)
		menu.items[n].posX = textx
		menu.items[n].posY = texty
	}
	// Adjust the rect so we're exactly the right height
	rect.Max.Y = rect.Min.Y + len(menu.items)*menu.rowHeight + menu.handleHeight + 2
	rect.Max.X = rect.Min.X + 12 + maxX

	// this depends on the width after we've adjusted
	menu.handleMidx = rect.Max.X - rect.Dx()/4

	menu.Rect = rect
}

// Draw xxx
func (menu *Menu) redraw() {

	DoUpstream(menu, "setcolor", color.RGBA{0x00, 0x00, 0x00, 0xff})
	DoUpstream(menu, "drawfilledrect", menu.Rect.Inset(1))

	DoUpstream(menu, "setcolor", color.RGBA{0xff, 0xff, 0xff, 0xff})
	DoUpstream(menu, "drawrect", menu.Rect)

	liney := menu.Rect.Min.Y + menu.handleHeight

	// Draw the bar at the top of the menu
	DoUpstream(menu, "drawline", DrawLineCmd{image.Point{menu.Rect.Min.X, liney}, image.Point{menu.Rect.Max.X, liney}})

	midx := menu.handleMidx

	// Draw the X at the right side of the handle area
	DoUpstream(menu, "drawline", DrawLineCmd{image.Point{midx, menu.Rect.Min.Y}, image.Point{midx, liney}})
	// the midx-1 is just so the X looks a little nicer
	DoUpstream(menu, "drawline", DrawLineCmd{image.Point{midx - 1, menu.Rect.Min.Y}, image.Point{menu.Rect.Max.X, liney}})
	DoUpstream(menu, "drawline", DrawLineCmd{image.Point{midx - 1, liney}, image.Point{menu.Rect.Max.X, menu.Rect.Min.Y}})

	nitems := len(menu.items)
	liney0 := menu.Rect.Min.Y + menu.handleHeight + menu.rowHeight/4 - 4
	for n, item := range menu.items {
		fore := foreColor
		back := backColor
		liney := liney0 + (n+1)*menu.rowHeight
		if n == menu.itemSelected {
			fore = backColor
			back = foreColor
			itemRect := image.Rect(menu.Rect.Min.X+1, liney-menu.rowHeight, menu.Rect.Max.X-1, liney)

			DoUpstream(menu, "setcolor", back)
			DoUpstream(menu, "drawfilledrect", itemRect)
		}

		DoUpstream(menu, "setcolor", fore)
		DoUpstream(menu, "drawtext", DrawTextCmd{item.label, menu.Style.fontFace, image.Point{item.posX, item.posY}})
		if n != menu.itemSelected && n < (nitems-1) {
			DoUpstream(menu, "drawline", DrawLineCmd{image.Point{menu.Rect.Min.X, liney}, image.Point{menu.Rect.Max.X, liney}})
		}
	}
}

func (menu *Menu) mouseHandler(cmd MouseCmd) (removeMenu bool) {
	me := cmd
	istransient := (GetAttValue(menu, "istransient") == "true")
	// If it's in the handle area...
	if me.Pos.Y <= menu.Rect.Min.Y+menu.handleHeight {
		menu.itemSelected = -1
		if me.Ddu == MouseDown {
			if me.Pos.X > menu.handleMidx {
				// Clicked in the X, remove the menu no matter what
				DoUpstream(menu, "closeme", menu)
				return true
			}
			DoUpstream(menu, "movemenu", menu)
			return false
		}
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
		w := item.target
		if w == nil {
			log.Printf("HEY! item.callbackWindow is nil?\n")
			if istransient {
				DoUpstream(menu, "closeme", menu)
			}
			return istransient
		}
		w.Do(menu, item.cmd, item.arg)
		menu.itemSelected = -1
		// If we're invoking a sub-menu, don't delete parent yet
		issubmenu := strings.HasSuffix(item.label, "->")
		if !issubmenu && istransient {
			DoUpstream(menu, "closeme", menu)
		}
		return !strings.HasSuffix(item.label, "->")
	}
	return false
}

/*
func (menu *Menu) callback(item MenuItem) {
	item.callback(item.label)
}
*/

// Do xxx
func (menu *Menu) Do(from Window, cmd string, arg interface{}) {

	switch cmd {

	case "closeyourself":
		log.Printf("menu.Do: closeyourself needs work!\n")

	case "resize":
		menu.resize(ToRect(arg))

	case "redraw":
		menu.redraw()

	case "mouse":
		mouse := ToMouse(arg)
		if menu.mouseHandler(mouse) {
			/*
				// XXX if it's in the X, we need to close no matter what
				if GetAttValue(menu, "istransient") == "true" {
					DoUpstream(menu, "closeme", menu)
				}
			*/
		}
	}
}

// DoSync xxx
func (menu *Menu) DoSync(w Window, cmd string, arg interface{}) (result interface{}, err error) {
	return NoSyncInterface("Menu")
}

/*
func (menu *Menu) addItem(s string, cb MenuCallback) {
	menu.items = append(menu.items, MenuItem{
		label:  s,
		posX:   0,
		posY:   0,
		target: nil,
		cmd:    "",
		arg:    nil,
	})
}
*/
