package gui

import "runtime"

// NewRootMenu xxx
func NewRootMenu(parent Window) *Menu {
	menu := NewMenu(parent)
	menu.addItem("About", func(name string) {
		menu.screen.log("This is the Palette Window System.")
		menu.screen.log("# of goroutines: %d", runtime.NumGoroutine())
	})
	menu.addItem("Move", func(name string) {
		menu.screen.log("Move callback")
	})
	menu.addItem("Resize", func(name string) {
		menu.startResize()
	})
	menu.addItem("Delete", func(name string) {
		menu.screen.log("Delete callback")
	})
	menu.addItem("Tools->", func(name string) {
		menu.screen.log("Tools callback")
	})
	menu.addItem("Misc ->", func(name string) {
		menu.screen.log("Misc callback")
	})
	return menu
}

func (menu *Menu) startResize() {
	menu.screen.setCursorStyle(sweepCursorStyle)
	if menu.resizeChan != nil {
		close(menu.resizeChan)
	}
	menu.resizeChan = make(chan MouseEvent)
	/*
		menu.screen.spawnMouseHandler(menu.mouseHandlerForResize)
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
