package gui

import (
	"fmt"
	"image"
	"image/color"
	"log"

	"golang.org/x/image/font"
)

// Window xxx
type Window interface {
	Data() *WindowData
	DoDownstream(DownstreamCmd)
	DoUpstream(Window, UpstreamCmd)
}

// WindowData xxx
type WindowData struct {
	parent Window
	Style  *Style
	Rect   image.Rectangle // in Screen coordinates, not relative

	children   map[string]Window
	windowName map[Window]string

	order []string // display order
}

// DownstreamCmd xxx
type DownstreamCmd interface {
}

// UpstreamCmd xxx
type UpstreamCmd interface {
}

// NewWindowData xxx
func NewWindowData(parent Window) WindowData {
	return WindowData{
		parent: parent,
		// fromUpstream:    fromUpstream,
		// toUpstream:      toUpstream,
		Style:      NewStyle("fixed", 16),
		Rect:       image.Rectangle{},
		children:   make(map[string]Window),
		windowName: make(map[Window]string),
		// childWaitGroup: &sync.WaitGroup{},
		order: make([]string, 0),
	}
}

////////////////////////////////////////////
// These are the standard WinInput commands
////////////////////////////////////////////

// MouseCmd xxx
type MouseCmd struct {
	Pos     image.Point
	ButtNum int
	Ddu     DownDragUp
}

// ResizeCmd xxx
type ResizeCmd struct {
	Rect image.Rectangle
}

// RedrawCmd xxx
type RedrawCmd struct {
	Rect image.Rectangle
}

// CloseYourselfCmd xxx
type CloseYourselfCmd struct {
}

// ClearCmd xxx
type ClearCmd struct {
}

// AddLineCmd xxx
type AddLineCmd struct {
	line string
}

////////////////////////////////////////////
// These are the standard WinOutput commands
////////////////////////////////////////////

// CloseMeCmd xxx
type CloseMeCmd struct {
	W Window
}

// SweepCallback xxx
type SweepCallback func(name string)

// StartSweepCmd xxx
type StartSweepCmd struct {
	callback SweepCallback
	toolName string
}

// Stringer xxx
func (c CloseMeCmd) Stringer() string {
	return fmt.Sprintf("CloseMeCmd w=%v", c.W)
}

// DrawLineCmd xxx
type DrawLineCmd struct {
	XY0, XY1 image.Point
	Color    color.RGBA
}

// Stringer xxx
func (c DrawLineCmd) Stringer() string {
	return fmt.Sprintf("DrawLineCmd XY0=%v XY1=%v color=%v", c.XY0, c.XY1, c.Color)
}

// DrawRectCmd xxx
type DrawRectCmd struct {
	Rect  image.Rectangle
	Color color.RGBA
}

// Stringer xxx
func (c DrawRectCmd) Stringer() string {
	return fmt.Sprintf("DrawRectCmd rect=%v color=%v", c.Rect, c.Color)
}

// DrawFilledRectCmd xxx
type DrawFilledRectCmd struct {
	Rect  image.Rectangle
	Color color.RGBA
}

// Stringer xxx
func (c DrawFilledRectCmd) Stringer() string {
	return fmt.Sprintf("DrawFilledRectCmd rect=%v color=%v", c.Rect, c.Color)
}

// DrawTextCmd xxx
type DrawTextCmd struct {
	Text  string
	Face  font.Face
	Pos   image.Point
	Color color.RGBA
}

// Stringer xxx
func (c DrawTextCmd) Stringer() string {
	return fmt.Sprintf("DrawTextCmd Pos=%v Text=%v", c.Pos, c.Text)
}

// ShowCursorCmd xxx
type ShowCursorCmd struct {
	show bool
}

// DownDragUp xxx
type DownDragUp int

// MouseUp xxx
const (
	MouseUp DownDragUp = iota
	MouseDown
	MouseDrag
)

// WindowUnder xxx
func WindowUnder(parent Window, pos image.Point) (Window, string) {

	parentData := parent.Data()

	// Check in reverse order
	for n := len(parentData.order) - 1; n >= 0; n-- {
		name := parentData.order[n]

		child := parentData.children[name]
		childRect := child.Data().Rect

		if pos.In(childRect) {
			return parentData.children[name], name
		}
	}
	return nil, ""
}

// AddChild xxx
func AddChild(parent Window, name string, child Window) Window {

	parentData := parent.Data()
	_, ok := parentData.children[name]
	if ok {
		log.Printf("AddChild: there's already a child named %s\n", name)
		return nil
	}

	// add it to the end of the display order
	parentData.order = append(parentData.order, name)

	parentData.children[name] = child
	parentData.windowName[child] = name
	return child
}

// RemoveChild xxx
func RemoveChild(parent Window, w Window) {

	windata := parent.Data()
	name, ok := windata.windowName[w]
	if !ok {
		log.Printf("RemoveWindow: no window w=%v\n", w)
		return
	}

	delete(windata.windowName, w)
	delete(windata.children, name)

	// find and delete it in the .order array
	for n, win := range windata.order {
		if name == win {
			copy(windata.order[n:], windata.order[n+1:])
			newlen := len(windata.order) - 1
			windata.order[newlen] = "" // XXX does this do anything?
			windata.order = windata.order[:newlen]
			break
		}
	}
}

// RedrawChildren xxx
func RedrawChildren(w Window) {
	for _, name := range w.Data().order {
		w := w.Data().children[name]
		w.DoDownstream(RedrawCmd{})
	}
}
