package gui

// NewPageMenu xxx
func NewPageMenu(parent Window) *Menu {

	menu := NewMenu(parent)
	menu.items = []MenuItem{
		{label: "About", target: parent, cmd: "about"},
		{label: "New Console", target: parent, cmd: "sweeptool", arg: "console"},
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
func NewToolsMenu(parent Window, parentMenu *Menu) *Menu {

	menu := NewMenu(parent)
	menu.parentMenu = parentMenu
	menu.items = []MenuItem{
		{label: "About", target: parent, cmd: "about"},
		{label: "Console", target: parent, cmd: "sweeptool", arg: "console"},
		{label: "Console2", target: parent, cmd: "sweeptool", arg: "console"},
	}
	return menu
}
