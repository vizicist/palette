package gui

import (
	"image"
	"image/color"
	"log"
)

// MenuCallback xxx
type MenuCallback func(updown string)

// Menu xxx
type Menu struct {
	WindowData
	isPressed bool
	items     []string
}

// NewMenu xxx
func NewMenu(style *Style) *Menu {
	return &Menu{
		WindowData: WindowData{
			style:   style,
			rect:    image.Rectangle{},
			objects: map[string]Window{},
		},
		isPressed: false,
		items:     make([]string, 0),
	}
}

// Data xxx
func (menu *Menu) Data() WindowData {
	return menu.WindowData
}

// Resize xxx
func (menu *Menu) Resize(rect image.Rectangle) {
	textHeight := menu.style.TextHeight()
	// Adjust the rect so we're exactly that height
	rect.Max.Y = rect.Min.Y + len(menu.items)*textHeight
	menu.rect = rect
	log.Printf("menu.Resize: rect=%v\n", menu.rect)
}

// Draw xxx
func (menu *Menu) Draw(screen *Screen) {

	color := color.RGBA{0xff, 0xff, 0, 0xff}
	screen.DrawRect(menu.rect, color)

	textHeight := menu.style.TextHeight()

	textx := menu.rect.Min.X
	for n, line := range menu.items {

		texty := menu.rect.Min.Y + n*textHeight

		brect := menu.style.BoundString(line)
		bminy := brect.Min.Y
		bmaxy := brect.Max.Y
		if bminy < 0 {
			bmaxy -= bminy
			bminy = 0
		}
		texty += bminy
		texty += bmaxy
		screen.drawText(line, menu.style.fontFace, textx, texty, color)
	}
}

// HandleMouseInput xxx
func (menu *Menu) HandleMouseInput(pos image.Point, button int, mdown bool) bool {
	// Should it be highlighting for copying ?
	log.Printf("Menu.HandleMouseInput: pos=%v\n", pos)
	return true
}

// AddItem xxx
func (menu *Menu) AddItem(s string) {
	// XXX - this can be done better
	menu.items = append(menu.items, s)
}
