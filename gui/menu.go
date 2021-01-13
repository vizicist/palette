package gui

import (
	"image"
	"image/color"
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
}

type (
	// MenuCallbackMouse ...
	MenuCallbackMouse func(item string)
	// MenuCallbackPick ...
	MenuCallbackPick func(item string, w Window)
	// MenuCallbackSweep ...
	MenuCallbackSweep func(item string, r image.Rectangle)
)

// CallbackStyle xxx
type callbackStyle int

const (
	mouseCallbackStyle = iota
	pickCallbackStyle
	sweepCallbackStyle
)

// MenuItem xxx
type MenuItem struct {
	label string
	posX  int
	posY  int
	// Only one of these three callbacks should be set, typically
	callbackMouse MenuCallbackMouse
	callbackPick  MenuCallbackPick
	callbackSweep MenuCallbackSweep
	callbackStyle callbackStyle
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
	if pos.Y <= menu.rect.Min.Y+menu.handleHeight {
		menu.itemSelected = -1
		return true
	}
	// Find out which item the mouse is in
	for n, item := range menu.items {
		if pos.Y < item.posY {
			menu.itemSelected = n
			break
		}
	}

	// No callbackups until MouseUp
	if event == MouseUp {

		item := menu.items[menu.itemSelected]

		switch item.callbackStyle {

		case mouseCallbackStyle:
			item.callbackMouse(item.label)
			menu.itemSelected = -1

		case pickCallbackStyle:
			w := menu.screen.root
			// Grab the mouse and wait for a MouseUp
			menu.screen.log("Waiting for pick")
			menu.pickedWindow = nil
			menu.screen.setMouseHandler(func(pos image.Point, button int, event MouseEvent) bool {
				if event == MouseDown {
					menu.pickedWindow = ObjectUnder(menu.screen.root, pos)
					menu.screen.log("mousedown, pos=%v pickedwindow=%v", pos, menu.pickedWindow)
				}
				if event == MouseUp {
					menu.screen.log("mouseup, RELEASING pos=%v pickedwindow=%v", pos, menu.pickedWindow)
					menu.screen.setDefaultMouseHandler()
				}
				return true
			})
			item.callbackPick(item.label, w)
			menu.itemSelected = -1

		case sweepCallbackStyle:
			r := image.Rect(0, 0, 100, 100) // fake
			item.callbackSweep(item.label, r)
			menu.itemSelected = -1
		}
	}
	return true
}

// HandleMouseInputDuringPick xxx
func (menu *Menu) HandleMouseInputDuringPick(pos image.Point, button int, event MouseEvent) bool {
	return true
}

// HandleMouseInputDuringSweep xxx
func (menu *Menu) HandleMouseInputDuringSweep(pos image.Point, button int, event MouseEvent) bool {
	return true
}

func (menu *Menu) addMouseItem(s string, cb MenuCallbackMouse) {
	menu.items = append(menu.items, MenuItem{
		label:         s,
		callbackMouse: cb,
		callbackStyle: mouseCallbackStyle,
	})
}

func (menu *Menu) addPickItem(s string, cb MenuCallbackPick) {
	menu.items = append(menu.items, MenuItem{
		label:         s,
		callbackPick:  cb,
		callbackStyle: pickCallbackStyle,
	})
}

func (menu *Menu) addSweepItem(s string, cb MenuCallbackSweep) {
	menu.items = append(menu.items, MenuItem{
		label:         s,
		callbackSweep: cb,
		callbackStyle: sweepCallbackStyle,
	})
}
