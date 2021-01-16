package gui

import "runtime"

// NewPageMenu xxx
func NewPageMenu(parent Window) *Menu {
	menu := NewMenu(parent)
	menu.addItem("About", func(name string) {
		menu.Screen.Log("This is the Palette Window System.")
		menu.Screen.Log("# of goroutines: %d", runtime.NumGoroutine())
	})
	menu.addItem("Move", func(name string) {
		menu.Screen.Log("Move callback")
	})
	menu.addItem("Resize", func(name string) {
		menu.startResize()
	})
	menu.addItem("Delete", func(name string) {
		menu.Screen.Log("Delete callback")
	})
	menu.addItem("Tools->", func(name string) {
		menu.Screen.Log("Tools callback")
	})
	menu.addItem("Misc ->", func(name string) {
		menu.Screen.Log("Misc callback")
	})
	return menu
}

func (menu *Menu) startResize() {
	menu.Screen.setCursorStyle(sweepCursorStyle)
	if menu.resizeChan != nil {
		close(menu.resizeChan)
	}
	menu.resizeChan = make(chan MouseMsg)
	/*
		menu.Screen.spawnMouseHandler(menu.mouseHandlerForResize)
		go menu.runResize()
	*/
}

/*
func (menu *Menu) mouseHandlerForResize(pos image.Point, button int, event MouseEvent) bool {
	return true
}

func (menu *Menu) mouseHandlerForResize(pos image.Point, button int, event MouseEvent) bool {
	return true
}
*/
