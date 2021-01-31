package gui

// NewPageMenu xxx
func NewPageMenu(parent *Page) *Menu {

	menu := NewMenuWindow(parent, nil).(*Menu)
	// XXX - should get rid of target: in Menuitem?  use Menu.parent?
	menu.items = []MenuItem{
		{label: "About", target: parent, cmd: "about"},
		{label: "Dump", target: parent, cmd: "dumptofile", arg: "homepage.json"},
		{label: "Restore", target: parent, cmd: "restore", arg: "homepage.json"},
		{label: "Resize", target: parent, cmd: "picktool", arg: "resize"},
		{label: "Move", target: parent, cmd: "movetool", arg: "move"},
		{label: "Delete", target: parent, cmd: "picktool", arg: "delete"},
		{label: "Tools  ->", target: parent, cmd: "toolsmenu"},
		{label: "Misc   ->", target: parent, cmd: "miscmenu"},
		{label: "Window ->", target: parent, cmd: "windowmenu"},
	}
	return menu
}

// NewToolsMenu xxx
func NewToolsMenu(parent Window, parentMenu Window) *Menu {

	menu := NewMenuWindow(parent, parentMenu).(*Menu)
	menu.parentMenu = parentMenu
	menu.items = []MenuItem{
		{label: "Console", target: parent, cmd: "sweeptool", arg: "Console"},
	}
	return menu
}
