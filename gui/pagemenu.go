package gui

import (
	"log"
)

// NewPageMenu xxx
func NewPageMenu(parent Window) *Menu {

	menu := NewMenu(parent)
	menu.items = []MenuItem{
		{label: "About", target: parent, cmd: "about"},
		{label: "Move", target: parent, cmd: "move"},
		{label: "New Console", target: parent, cmd: "sweeptool", arg: "console"},
		{label: "Delete", target: parent, cmd: "delete"},
		{label: "Tools  ->", target: parent, cmd: "toolsmenu"},
		{label: "Misc   ->", target: parent, cmd: "miscmenu"},
		{label: "Window ->", target: parent, cmd: "windowmenu"},
	}
	return menu
}

// PageToolCallback xxx
func PageToolCallback(name string) {
	log.Printf("PageToolCallback!\n")
}

func (menu *Menu) startNewConsole() {

	log.Printf("startNewConsole: start\n")

	// Take over the mouse to sweep out the console area

	// newInput := PushWindowInput(menu)

	// go menu.sweepConsoleWindow(newInput)

}

func (menu *Menu) sweepConsoleWindow(cmd MouseCmd) {
	switch cmd.Ddu {
	case MouseDown:
		log.Printf("!!!!!!!!!!!!!!!!!!sweepWindow: MouseDown pos=%v\n", cmd.Pos)
	case MouseDrag:
		log.Printf("!!!!!!!!!!!!!!!!!!sweepWindow: MouseDrag pos=%v\n", cmd.Pos)
	case MouseUp:
		log.Printf("!!!!!!!!!!!!!!!!!!sweepWindow: MouseUp pos=%v\n", cmd.Pos)
		// PopWindowInput(menu)
	}
}
