package gui

import (
	"log"
	"runtime"
)

// NewPageMenu xxx
func NewPageMenu(parent Window) Window {
	items := []MenuItem{
		{label: "About", callback: func(name string) {
			log.Printf("This is the Palette Window System.")
			log.Printf("# of goroutines: %d", runtime.NumGoroutine())
		}},

		{label: "Move", callback: func(name string) {
			log.Printf("Move callback")
		}},

		{label: "New Console", callback: func(name string) {
			log.Printf("New Console menu item")
			parent.DoUpstream(nil, StartSweepCmd{callback: PageToolCallback, toolName: name})
			// menu.startNewConsole()
		}},

		{label: "Delete", callback: func(name string) {
			log.Printf("Delete callback")
		}},

		{label: "Tools ->", callback: func(name string) {
			log.Printf("Tools callback")
		}},

		{label: "Misc  ->", callback: func(name string) {
			log.Printf("Misc callback")
		}},

		{label: "Window ->", callback: func(name string) {
			log.Printf("Misc callback")
		}}}

	menu := NewMenu(parent, items)
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
