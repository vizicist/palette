package twinsys

import "github.com/vizicist/palette/kit"

// NewPageMenu xxx
func NewPageMenu(parent Window) WindowData {

	items := []MenuItem{
		{Label: "About", Cmd: kit.NewSimpleCmd("about")},
		{Label: "Dump", Cmd: NewDumpFileCmd("homepage.json")},
		{Label: "Restore", Cmd: NewRestoreFileCmd("homepage.json")},
		{Label: "Tools  ->", Cmd: NewSubMenuCmd("ToolsMenu")},
		{Label: "Window ->", Cmd: NewSubMenuCmd("WindowMenu")},
	}
	return NewMenu(parent, "PageMenu", items)
}

// NewToolsMenu xxx
func NewToolsMenu(parent Window) WindowData {
	items := []MenuItem{
		{Label: "Console", Cmd: NewSweepToolCmd("Console")},
		{Label: "Riff", Cmd: NewSweepToolCmd("Riff")},
	}
	return NewMenu(parent, "ToolsMenu", items)
}

// NewWindowMenu xxx
func NewWindowMenu(parent Window) WindowData {
	items := []MenuItem{
		{Label: "Resize", Cmd: NewPickToolCmd("resize")},
		{Label: "Move", Cmd: kit.NewSimpleCmd("movetool")},
		{Label: "Delete", Cmd: NewPickToolCmd("delete")},
		{Label: "More  ->", Cmd: NewSubMenuCmd("ToolsMenu")},
	}
	return NewMenu(parent, "WindowMenu", items)
}
