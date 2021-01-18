package gui

import (
	"image"
	"image/color"
	"log"

	"golang.org/x/image/font"
)

// Window xxx
type Window interface {
	Run()
	Draw()
	Resize(image.Rectangle) image.Rectangle
	Data() *WindowData
}

// WindowData xxx
type WindowData struct {
	InChan  chan WinInput  // unbuffered?
	OutChan chan WinOutput // unbuffered?
	// MouseChan chan MouseCmd  // unbuffered?
	Style      *Style
	Rect       image.Rectangle // in Screen coordinates, not relative
	windows    map[Window]string
	order      []Window // display order
	isMenu     bool
	inputStack []chan WinInput
}

// NewWindowData xxx
func NewWindowData(parent Window) WindowData {
	return WindowData{
		InChan:     make(chan WinInput),
		OutChan:    parent.Data().OutChan,
		Style:      parent.Data().Style,
		Rect:       image.Rectangle{},
		windows:    make(map[Window]string),
		inputStack: make([]chan WinInput, 0),
	}
}

// WinInput xxx
type WinInput interface {
}

// WinOutput xxx
type WinOutput interface {
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

////////////////////////////////////////////
// These are the standard WinOutput commands
////////////////////////////////////////////

// CloseMeCmd xxx
type CloseMeCmd struct {
	W Window
}

// DrawLineCmd xxx
type DrawLineCmd struct {
	XY0, XY1 image.Point
	Color    color.RGBA
}

// DrawRectCmd xxx
type DrawRectCmd struct {
	Rect  image.Rectangle
	Color color.RGBA
}

// DrawFilledRectCmd xxx
type DrawFilledRectCmd struct {
	Rect  image.Rectangle
	Color color.RGBA
}

// DrawTextCmd xxx
type DrawTextCmd struct {
	Text  string
	Face  font.Face
	Pos   image.Point
	Color color.RGBA
}

// SetCursorStyleCmd xxx
type SetCursorStyleCmd struct {
	CursorStyle cursorStyle
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
func WindowUnder(o Window, pos image.Point) Window {
	windata := o.Data()
	// Check in reverse order
	for n := len(windata.order) - 1; n >= 0; n-- {
		w := windata.order[n]
		if w == nil {
			log.Printf("WindowUnder: value %d is nil?\n", n)
			return nil
		}
		if pos.In(w.Data().Rect) {
			return w
		}
	}
	return nil
}

// AddWindow xxx
func AddWindow(parent Window, o Window, name string) {
	windata := parent.Data()
	_, ok := windata.windows[o]
	if ok {
		log.Printf("There's already a window %v in that Window\n", o)
		return
	}
	windata.windows[o] = name
	// add it to the end of the display order
	windata.order = append(windata.order, o)
}

// RemoveWindow xxx
func RemoveWindow(parent Window, w Window) {

	windata := parent.Data()
	name, ok := windata.windows[w]
	if !ok {
		log.Printf("RemoveWindow: no window w=%v\n", w)
		return
	}
	log.Printf("RemoveWindow: name=%s w=%v\n", name, w)

	delete(windata.windows, w)
	// find and delete it in the .order array
	for n, win := range windata.order {
		if w == win {
			copy(windata.order[n:], windata.order[n+1:])
			newlen := len(windata.order) - 1
			windata.order[newlen] = nil // XXX does this do anything?
			windata.order = windata.order[:newlen]
			break
		}
	}
}

// PushWindowInput xxx
func PushWindowInput(w Window) chan WinInput {
	windata := w.Data()
	// Save the current Input
	newInput := make(chan WinInput)
	windata.inputStack = append(windata.inputStack, newInput)
	windata.InChan = newInput
	return newInput
}

// PopWindowInput xxx
func PopWindowInput(w Window) {
	windata := w.Data()
	stackSize := len(windata.inputStack)
	if stackSize == 0 {
		log.Printf("menu.PopInput: no input to pop!?\n")
		return
	}
	close(windata.InChan)
	// pop off the tail of the array
	windata.InChan = windata.inputStack[stackSize-1]
	windata.inputStack = windata.inputStack[:stackSize-1]
}

// CloseWindow xxx
func CloseWindow(w Window) {
	windata := w.Data()
	for len(windata.inputStack) > 0 {
		log.Printf("CloseWindow: inputStack isn't empty?\n")
		PopWindowInput(w)
	}
	if windata.InChan == nil {
		log.Printf("CloseWindow: InChan is nil?\n")
	} else {
		close(windata.InChan)
		windata.InChan = nil
	}
}
