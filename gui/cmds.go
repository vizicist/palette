package gui

import (
	"image"
	"image/color"
	"log"

	"golang.org/x/image/font"
)

// Cmd xxx
type Cmd struct {
	name string
	arg  interface{}
}

// UpstreamCmd xxx
type UpstreamCmd interface {
}

////////////////////////////////////////////
// These are the standard Downstream commands
////////////////////////////////////////////

// MouseCmd xxx
type MouseCmd struct {
	Pos     image.Point
	ButtNum int
	Ddu     DownDragUp
}

// CloseYourselfCmd xxx
type CloseYourselfCmd struct {
}

////////////////////////////////////////////
// These are the standard Upstream commands
////////////////////////////////////////////

// DrawLineCmd xxx
type DrawLineCmd struct {
	XY0, XY1 image.Point
}

// DrawTextCmd xxx
type DrawTextCmd struct {
	Text string
	Face font.Face
	Pos  image.Point
}

// SweepCallbackCmd xxx
type SweepCallbackCmd struct {
	callbackWindow Window
	callbackCmd    string
	callbackArg    interface{}
}

// DownDragUp xxx
type DownDragUp int

// MouseUp xxx
const (
	MouseUp DownDragUp = iota
	MouseDown
	MouseDrag
)

// ToRect xxx
func ToRect(arg interface{}) image.Rectangle {
	r, ok := arg.(image.Rectangle)
	if !ok {
		log.Printf("Unable to convert interface to Rect!\n")
		r = image.Rect(0, 0, 0, 0)
	}
	return r
}

// ToPoint xxx
func ToPoint(arg interface{}) image.Point {
	p, ok := arg.(image.Point)
	if !ok {
		log.Printf("Unable to convert interface to Point!\n")
		p = image.Point{0, 0}
	}
	return p
}

// ToMenu xxx
func ToMenu(arg interface{}) *Menu {
	r, ok := arg.(*Menu)
	if !ok {
		log.Printf("Unable to convert interface to Menu!\n")
		r = nil
	}
	return r
}

// ToDrawLine xxx
func ToDrawLine(arg interface{}) DrawLineCmd {
	r, ok := arg.(DrawLineCmd)
	if !ok {
		log.Printf("Unable to convert interface to DrawLineCmd!\n")
		r = DrawLineCmd{}
	}
	return r
}

// ToDrawText xxx
func ToDrawText(arg interface{}) DrawTextCmd {
	r, ok := arg.(DrawTextCmd)
	if !ok {
		log.Printf("Unable to convert interface to DrawTextCmd!\n")
		r = DrawTextCmd{}
	}
	return r
}

// ToMouse xxx
func ToMouse(arg interface{}) MouseCmd {
	r, ok := arg.(MouseCmd)
	if !ok {
		log.Printf("Unable to convert interface to MouseCmd!\n")
		r = MouseCmd{}
	}
	return r
}

// ToColor xxx
func ToColor(arg interface{}) color.RGBA {
	r, ok := arg.(color.RGBA)
	if !ok {
		log.Printf("Unable to convert interface to Color!\n")
		r = color.RGBA{0x00, 0x00, 0x00, 0xff}
	}
	return r
}

// ToBool xxx
func ToBool(arg interface{}) bool {
	r, ok := arg.(bool)
	if !ok {
		log.Printf("Unable to convert interface to bool!\n")
		r = false
	}
	return r
}

// ToString xxx
func ToString(arg interface{}) string {
	if arg == nil {
		return ""
	}
	r, ok := arg.(string)
	if !ok {
		log.Printf("Unable to convert interface to string!\n")
		r = ""
	}
	return r
}

// ToWindow xxx
func ToWindow(arg interface{}) Window {
	r, ok := arg.(Window)
	if !ok {
		log.Printf("Unable to convert interface to Window!\n")
		r = nil
	}
	return r
}
