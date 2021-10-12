package winsys

import (
	"image"
	"log"
)

// WindowData fields are exported
type WindowData struct {
	w        Window
	toolType string
	minSize  image.Point
}

// NewToolData xxx
func NewToolData(w Window, toolType string, minSize image.Point) WindowData {
	return WindowData{
		w:        w,
		toolType: toolType,
		minSize:  minSize,
	}
}

// WindowMaker xxx
type WindowMaker func(parent Window) WindowData

// WindowMakers xxx
var WindowMakers = make(map[string]WindowMaker)

// RegisterWindow xxx
func RegisterWindow(name string, newfunc WindowMaker) {
	WindowMakers[name] = newfunc
	log.Printf("Added Tool %s maker=%T\n", name, newfunc)
}

// RegisterDefaultTools xxx
func RegisterDefaultTools() {
	RegisterWindow("PageMenu", NewPageMenu)
	RegisterWindow("ToolsMenu", NewToolsMenu)
	RegisterWindow("WindowMenu", NewWindowMenu)
}
