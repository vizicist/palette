package gui

import "image"

// NewPageMenu xxx
func NewPageMenu(style *Style) (w Window, minRect image.Rectangle) {

	items := []MenuItem{
		{label: "About", cmd: "about"},
		{label: "Dump", cmd: "dumptofile", arg: "homepage.json"},
		{label: "Restore", cmd: "restore", arg: "homepage.json"},
		{label: "Resize", cmd: "picktool", arg: "resize"},
		{label: "Move", cmd: "movetool", arg: "move"},
		{label: "Delete", cmd: "picktool", arg: "delete"},
		{label: "Tools  ->", cmd: "toolsmenu"},
		{label: "Misc   ->", cmd: "miscmenu"},
		{label: "Window ->", cmd: "windowmenu"},
	}
	w, minRect = NewMenu(style, items)
	return w, minRect
}

// NewToolsMenu xxx
func NewToolsMenu(style *Style) (w Window, minRect image.Rectangle) {

	items := []MenuItem{
		{label: "Console", cmd: "sweeptool", arg: "Console"},
	}
	w, minRect = NewMenu(style, items)
	return w, minRect
}
