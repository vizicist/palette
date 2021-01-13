package gui

// NewRootMenu xxx
func NewRootMenu(parent Window) *Menu {
	menu := NewMenu(parent)
	menu.addMouseItem("About", func(name string) {
		menu.screen.log("This is the Palette Window System.")
	})
	menu.addPickItem("Move", func(name string, w Window) {
		menu.screen.log("Move callback, name=%s window=%T", name, w)
	})
	menu.addMouseItem("Resize", func(name string) {
		menu.screen.log("Resize callback")
	})
	menu.addMouseItem("Delete", func(name string) {
		menu.screen.log("Delete callback")
	})
	menu.addMouseItem("Tools->", func(name string) {
		menu.screen.log("Tools callback")
	})
	menu.addMouseItem("Misc ->", func(name string) {
		menu.screen.log("Misc callback")
	})
	return menu
}
