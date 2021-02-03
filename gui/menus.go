package gui

// NewPageMenu xxx
func NewPageMenu(parent Window) ToolData {

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
	return NewMenu(parent, items)
}

// NewToolsMenu xxx
func NewToolsMenu(parent Window) ToolData {
	items := []MenuItem{
		{label: "Console", cmd: "sweeptool", arg: "Console"},
	}
	return NewMenu(parent, items)
}
