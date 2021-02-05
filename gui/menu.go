package gui

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"strings"
)

// Menu xxx
type Menu struct {
	ctx           WinContext
	items         []MenuItem
	isPressed     bool
	itemSelected  int
	rowHeight     int
	handleHeight  int
	handleMidx    int
	pickedWindow  Window
	sweepBegin    image.Point
	sweepEnd      image.Point
	resizeState   int
	parentMenu    Window
	activeSubMenu *Menu
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

var lastMenuX int

// NewMenu xxx
func NewMenu(parent Window, toolType string, items []MenuItem) ToolData {

	menu := &Menu{
		ctx:          NewWindowContext(parent),
		isPressed:    false,
		items:        items,
		itemSelected: -1,
	}

	style := WinStyle(menu)
	// DO NOT use Style.RowHeight, we want wider row spacing
	menu.rowHeight = style.TextHeight() + 6
	menu.handleHeight = 9
	minSize := image.Point{
		X: 12 + menu.maxX(style, menu.items),
		Y: len(menu.items)*menu.rowHeight + menu.handleHeight + 2,
	}

	winSaveTransient(parent, menu)
	return NewToolData(menu, toolType, minSize)
}

// Context xxx
func (menu *Menu) Context() *WinContext {
	return &menu.ctx
}

//////////////////////// End of Window interface methods

// SetItems xxx
func (menu *Menu) SetItems(items []MenuItem) {
	menu.items = items
}

// This only pays attention to rect.Min
// and computes the posX,posY of each menu item
func (menu *Menu) resize(size image.Point) {

	style := WinStyle(menu)
	// Recompute all the y positions in the items

	maxX := -1
	for n, item := range menu.items {
		brect := style.BoundString(item.label)
		if brect.Max.X > maxX {
			maxX = brect.Max.X
		}
		// do not use Y info from brect,
		// the vertical placement of items should
		// be independent of the labels
		menu.items[n].posX = 6
		menu.items[n].posY = 6 + menu.handleHeight + n*menu.rowHeight + (menu.rowHeight / 2)
	}
	// this depends on the width after we've adjusted
	menu.handleMidx = size.X - size.X/4
}

// minSize computes the minimum size of the menu
func (menu *Menu) maxX(style *Style, items []MenuItem) int {
	maxX := -1
	for _, item := range items {
		brect := style.BoundString(item.label)
		if brect.Max.X > maxX {
			maxX = brect.Max.X
		}
	}
	return maxX
}

// Draw xxx
func (menu *Menu) redraw() {

	currSize := WinCurrSize(menu)
	rect := image.Rect(0, 0, currSize.X, currSize.Y)
	style := WinStyle(menu)

	DoUpstream(menu, "setcolor", color.RGBA{0x00, 0x00, 0x00, 0xff})
	DoUpstream(menu, "drawfilledrect", rect.Inset(1))

	DoUpstream(menu, "setcolor", color.RGBA{0xff, 0xff, 0xff, 0xff})
	DoUpstream(menu, "drawrect", rect)

	liney := rect.Min.Y + menu.handleHeight

	// Draw the bar at the top of the menu
	DoUpstream(menu, "drawline", DrawLineCmd{image.Point{rect.Min.X, liney}, image.Point{rect.Max.X, liney}})

	midx := menu.handleMidx

	// Draw the X at the right side of the handle area
	DoUpstream(menu, "drawline", DrawLineCmd{image.Point{midx, rect.Min.Y}, image.Point{midx, liney}})
	// the midx-1 is just so the X looks a little nicer
	DoUpstream(menu, "drawline", DrawLineCmd{image.Point{midx - 1, rect.Min.Y}, image.Point{rect.Max.X, liney}})
	DoUpstream(menu, "drawline", DrawLineCmd{image.Point{midx - 1, liney}, image.Point{rect.Max.X, rect.Min.Y}})

	nitems := len(menu.items)
	liney0 := rect.Min.Y + menu.handleHeight + menu.rowHeight/4 - 4
	for n, item := range menu.items {
		fore := ForeColor
		back := BackColor
		liney := liney0 + (n+1)*menu.rowHeight
		if n == menu.itemSelected {
			fore = BackColor
			back = ForeColor
			itemRect := image.Rect(rect.Min.X+1, liney-menu.rowHeight, rect.Max.X-1, liney)

			DoUpstream(menu, "setcolor", back)
			DoUpstream(menu, "drawfilledrect", itemRect)
		}

		DoUpstream(menu, "setcolor", fore)
		DoUpstream(menu, "drawtext", DrawTextCmd{item.label, style.fontFace, rect.Min.Add(image.Point{item.posX, item.posY})})
		if n != menu.itemSelected && n < (nitems-1) {
			DoUpstream(menu, "drawline", DrawLineCmd{image.Point{rect.Min.X, liney}, image.Point{rect.Max.X, liney}})
		}
	}
}

// If mouseHandler return value is true, the menu should be removed
func (menu *Menu) mouseHandler(mouse MouseCmd) (removeMenu bool) {

	menu.itemSelected = -1

	// If it's in the handle area...
	if mouse.Pos.Y <= menu.handleHeight {
		if mouse.Ddu == MouseDown {
			if mouse.Pos.X > menu.handleMidx {
				// Clicked in the X, remove the menu no matter what
				DoUpstream(menu, "closeme", menu)
				return true
			}
			DoUpstream(menu, "movemenu", menu)
			DoUpstream(menu, "makepermanent", menu)
			return false
		}
		return false
	}

	// Find out which item the mouse is in
	for n, item := range menu.items {
		if mouse.Pos.Y < item.posY {
			menu.itemSelected = n
			break
		}
	}

	if menu.itemSelected < 0 {
		return false
	}

	parent := WinParent(menu)
	// No callbackups until MouseUp
	if mouse.Ddu == MouseUp {
		item := menu.items[menu.itemSelected]
		// Save lastMenuX so sub-menu can position itself
		lastMenuX = WinChildRect(parent, menu).Max.X
		doingSubMenu := strings.HasSuffix(item.label, "->")

		parent.Do(item.cmd, item.arg) // This is the menu "callback" to the parent

		menu.itemSelected = -1
		if !doingSubMenu {
			DoUpstream(menu, "closetransients", nil)
		}
		return !strings.HasSuffix(item.label, "->")
	}
	return false
}

// Do xxx
func (menu *Menu) Do(cmd string, arg interface{}) (interface{}, error) {

	switch cmd {

	case "closeyourself":
		log.Printf("menu.Do: closeyourself needs work!\n")

	case "resize":
		menu.resize(ToPoint(arg))

	case "redraw":
		menu.redraw()

	case "restore":
		state := arg.(map[string]interface{})
		items := state["items"].([]interface{})
		newitems := make([]MenuItem, len(items))
		for n, i := range items {
			item := i.(map[string]interface{})
			label := ToString(item["label"])
			cmd := ToString(item["cmd"])
			arg := ToString(item["arg"])
			newitems[n] = MenuItem{label: label, cmd: cmd, arg: arg}
		}
		menu.items = newitems

	case "dumpstate":
		s := fmt.Sprintf("{\n\"items\": [\n")
		sep := ""
		parent := WinParent(menu)
		for _, item := range menu.items {
			argstr := ToString(item.arg)
			targetWid := GetWindowID(parent, item.target)
			s += fmt.Sprintf("%s{ \"label\": \"%s\", \"cmd\": \"%s\", \"arg\": \"%s\", \"target\": \"%s\" }",
				sep, item.label, item.cmd, argstr, targetWid)
			sep = ",\n"
		}
		s += "\n]\n}"
		return s, nil

	case "mouse":
		mouse := ToMouse(arg)
		if menu.mouseHandler(mouse) {
			// if GetAttValue(menu, "istransient") == "true" {
			DoUpstream(menu, "closetransients", menu)
			// }
		}
	}
	return nil, nil
}

// GetWindowID xxx
func GetWindowID(parent Window, w Window) string {
	pc := parent.Context()
	for wid, child := range pc.childWindow {
		if child == w {
			return fmt.Sprintf("%d", wid)
		}
	}
	return "0"
}
