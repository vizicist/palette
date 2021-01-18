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
	objects    map[string]Window
	order      []string // display order
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
		objects:    map[string]Window{},
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

////////////////////////////////////////////
// These are the standard WinOutput commands
////////////////////////////////////////////

// KillWindowCmd xxx
type KillWindowCmd struct {
	W Window
}

// DrawLineCmd xxx
type DrawLineCmd struct {
	X0, Y0, X1, Y1 int
	Color          color.RGBA
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

// ObjectUnder xxx
func ObjectUnder(o Window, pos image.Point) Window {
	windata := o.Data()
	// Check in reverse order
	for n := len(windata.order) - 1; n >= 0; n-- {
		name := windata.order[n]
		w, ok := windata.objects[name]
		if !ok {
			log.Printf("ObjectUnder: no entry in object for %s\n", name)
			return nil
		}
		if w == nil {
			log.Printf("ObjectUnder: objects entry for %s is nul?\n", name)
			return nil
		}
		if pos.In(w.Data().Rect) {
			return w
		}
	}
	return nil
}

// AddWindow xxx
func AddWindow(parent Window, name string, o Window) {
	windata := parent.Data()
	objects := windata.objects
	_, ok := objects[name]
	if ok {
		log.Printf("There's already an object named %s in that Window\n", name)
	} else {
		objects[name] = o
		// add it to the end of the display order
		windata.order = append(windata.order, name)
	}
}

// RemoveWindow xxx
func RemoveWindow(parent Window, name string) {

	windata := parent.Data()
	_, ok := windata.objects[name]
	if !ok {
		log.Printf("RemoveObject: no object named %s\n", name)
		return
	}

	delete(windata.objects, name)
	// find and delete it in the .order array
	for n, nm := range windata.order {
		if nm == name {
			copy(windata.order[n:], windata.order[n+1:])
			newlen := len(windata.order) - 1
			windata.order[newlen] = "" // XXX does this do anything?
			windata.order = windata.order[:newlen]
			break
		}
	}
}

// GrabWindowInput xxx
func GrabWindowInput(w Window) chan WinInput {
	windata := w.Data()
	// Save the current Input
	newInput := make(chan WinInput)
	windata.inputStack = append(windata.inputStack, newInput)
	windata.InChan = newInput
	return newInput
}

// ReleaseWindowInput xxx
func ReleaseWindowInput(w Window) {
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
