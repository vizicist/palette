package gui

func initStandardMenus() {
	RegisterToolType("PageMenu", NewPageMenu)
	RegisterToolType("ToolsMenu", NewToolsMenu)
	RegisterToolType("WindowMenu", NewWindowMenu)
}

// NewPageMenu xxx
func NewPageMenu(parent Window) ToolData {

	items := []MenuItem{
		{label: "About", cmd: "about"},
		{label: "Dump", cmd: "dumptofile", arg: "homepage.json"},
		{label: "Restore", cmd: "restore", arg: "homepage.json"},
		{label: "Tools  ->", cmd: "submenu", arg: "ToolsMenu"},
		{label: "Window ->", cmd: "submenu", arg: "WindowMenu"},
	}
	return NewMenu(parent, "PageMenu", items)
}

// NewToolsMenu xxx
func NewToolsMenu(parent Window) ToolData {
	items := []MenuItem{
		{label: "Console", cmd: "sweeptool", arg: "Console"},
		{label: "Riff", cmd: "sweeptool", arg: "Riff"},
	}
	return NewMenu(parent, "ToolsMenu", items)
}

// NewWindowMenu xxx
func NewWindowMenu(parent Window) ToolData {
	items := []MenuItem{
		{label: "Resize", cmd: "picktool", arg: "resize"},
		{label: "Move", cmd: "movetool", arg: "move"},
		{label: "Delete", cmd: "picktool", arg: "delete"},
		{label: "More  ->", cmd: "submenu", arg: "ToolsMenu"},
	}
	return NewMenu(parent, "WindowMenu", items)
}
