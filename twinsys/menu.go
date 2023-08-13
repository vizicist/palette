package twinsys

import (
	"image"
	"image/color"
	"strings"

	"github.com/vizicist/palette/hostwin"
)

// Menu xxx
type Menu struct {
	ctx          WinContext
	items        []MenuItem
	isPressed    bool
	itemSelected int
	rowHeight    int
	handleHeight int
	handleMidx   int
	// pickedWindow  Window
	// sweepBegin    image.Point
	// sweepEnd      image.Point
	// resizeState   int
	// parentMenu    Window
	// activeSubMenu *Menu
}

// MenuCallback xxx
type MenuCallback func(item string)

// MenuItem xxx
type MenuItem struct {
	Label string
	posX  int
	posY  int
	Cmd   hostwin.Cmd
}

var lastMenuX int
var doingMenu *Menu

// NewMenu xxx
func NewMenu(parent Window, menuType string, items []MenuItem) WindowData {

	menu := &Menu{
		ctx:          NewWindowContext(parent),
		isPressed:    false,
		items:        items,
		itemSelected: -1,
	}

	styleInfo := WinStyleInfo(menu)
	// DO NOT use Style.RowHeight, we want wider row spacing
	menu.rowHeight = styleInfo.TextHeight() + 6
	menu.handleHeight = 9
	minSize := image.Point{
		X: 12 + menu.maxX(styleInfo, menu.items),
		Y: len(menu.items)*menu.rowHeight + menu.handleHeight + 2,
	}

	winSaveTransient(parent, menu)
	return NewToolData(menu, menuType, minSize)
}

// Context xxx
func (menu *Menu) Context() *WinContext {
	return &menu.ctx
}

//////////////////////// End of Window interface methods

// This only pays attention to rect.Min
// and computes the posX,posY of each menu item
func (menu *Menu) resize(size image.Point) {

	styleInfo := WinStyleInfo(menu)
	// Recompute all the y positions in the items

	maxX := -1
	for n, item := range menu.items {
		brect := styleInfo.BoundString(item.Label)
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
func (menu *Menu) maxX(style *StyleInfo, items []MenuItem) int {
	maxX := -1
	for _, item := range items {
		brect := style.BoundString(item.Label)
		if brect.Max.X > maxX {
			maxX = brect.Max.X
		}
	}
	return maxX
}

// Draw xxx
func (menu *Menu) redraw() {

	currSize := WinGetSize(menu)
	rect := image.Rect(0, 0, currSize.X, currSize.Y)
	styleName := WinStyleName(menu)

	WinDoUpstream(menu, NewSetColorCmd(color.RGBA{0x00, 0x00, 0x00, 0xff}))
	WinDoUpstream(menu, NewDrawFilledRectCmd(rect.Inset(1)))

	WinDoUpstream(menu, NewSetColorCmd(color.RGBA{0xff, 0xff, 0xff, 0xff}))
	WinDoUpstream(menu, NewDrawRectCmd(rect))

	liney := rect.Min.Y + menu.handleHeight

	// Draw the bar at the top of the menu
	WinDoUpstream(menu, NewDrawLineCmd(image.Point{rect.Min.X, liney}, image.Point{rect.Max.X, liney}))

	midx := menu.handleMidx

	// Draw the X at the right side of the handle area

	WinDoUpstream(menu, NewDrawLineCmd(image.Point{midx, rect.Min.Y}, image.Point{midx, liney}))
	// the midx-1 is just so the X looks a little nicer
	WinDoUpstream(menu, NewDrawLineCmd(image.Point{midx - 1, rect.Min.Y}, image.Point{rect.Max.X, liney}))
	WinDoUpstream(menu, NewDrawLineCmd(image.Point{midx - 1, liney}, image.Point{rect.Max.X, rect.Min.Y}))

	nitems := len(menu.items)
	liney0 := rect.Min.Y + menu.handleHeight + menu.rowHeight/4 - 4
	for n, item := range menu.items {
		fore := ForeColor
		// back := BackColor
		liney := liney0 + (n+1)*menu.rowHeight
		if n == menu.itemSelected {
			fore = BackColor
			back := ForeColor
			itemRect := image.Rect(rect.Min.X+1, liney-menu.rowHeight, rect.Max.X-1, liney)

			WinDoUpstream(menu, NewSetColorCmd(back))
			WinDoUpstream(menu, NewDrawFilledRectCmd(itemRect))
		}

		WinDoUpstream(menu, NewSetColorCmd(fore))
		WinDoUpstream(menu, NewDrawTextCmd(item.Label, styleName, rect.Min.Add(image.Point{item.posX, item.posY})))
		if n != menu.itemSelected && n < (nitems-1) {
			WinDoUpstream(menu, NewDrawLineCmd(image.Point{rect.Min.X, liney}, image.Point{rect.Max.X, liney}))
		}
	}
}

// If mouseHandler return value is true, the menu should be removed
func (menu *Menu) mouseHandler(cmd hostwin.Cmd) (removeMenu bool) {

	parent := WinParent(menu)
	menu.itemSelected = -1

	pos := cmd.ValuesPos(hostwin.PointZero)
	ddu := cmd.ValuesString("ddu", "")
	// If it's in the handle area...
	if pos.Y <= menu.handleHeight {
		if ddu == "down" {
			menuname := WinChildName(parent, menu)
			if pos.X > menu.handleMidx {
				// Clicked in the X, remove the menu no matter what
				WinDoUpstream(menu, NewCloseMeCmd(menuname))
				return true
			}
			WinDoUpstream(menu, NewMoveMenuCmd(menuname))
			WinDoUpstream(menu, NewMakePermanentCmd(menuname))
			return false
		}
		return false
	}

	// Find out which item the mouse is in
	for n, item := range menu.items {
		if pos.Y < item.posY {
			menu.itemSelected = n
			break
		}
	}

	if menu.itemSelected < 0 {
		return false
	}

	// No callbackups until MouseUp
	if ddu == "up" {
		item := menu.items[menu.itemSelected]
		// Save lastMenuX so sub-menu can position itself
		lastMenuX = WinChildRect(parent, menu).Max.X
		doingSubMenu := strings.HasSuffix(item.Label, "->")

		doingMenu = menu    // this enables "dump" to ignore the menu that's triggering it
		parent.Do(item.Cmd) // This is the menu "callback" to the parent

		menu.itemSelected = -1
		if !doingSubMenu {
			WinDoUpstream(menu, NewCloseTransientsCmd(""))
		}
		return !strings.HasSuffix(item.Label, "->")
	}
	return false
}

// Do xxx
func (menu *Menu) Do(cmd hostwin.Cmd) string {

	switch cmd.Subj {

	case "close":
		hostwin.LogWarn("menu.Do: close needs work?  Maybe not")

	case "resize":
		size := cmd.ValuesXY("size", hostwin.PointZero)
		menu.resize(size)

	case "redraw":
		menu.redraw()

	case "restore":
		// We assume the New*Menu routines add the static items
		// in a menu, and possible other dynamic items.
		// I.e. dynamic contents of a menu isn't saved in State

	case "getstate":
		// For the moment, Menus are somewhat static, at least
		// in terms of what's saved in a state.  This doesn't
		// preclude a menu creation routine from adding menu items
		// based on other info (e.g. getting the list of files
		// in a directory, or MIDI device output names, etc.)
		return "\"state\":\"\""

	case "mouse":
		if menu.mouseHandler(cmd) {
			// if GetAttValue(menu, "istransient") == "true" {
			parent := WinParent(menu)
			menuname := WinChildName(parent, menu)
			WinDoUpstream(menu, NewCloseTransientsCmd(menuname))
			// }
		}
	}
	return hostwin.OkResult()
}
