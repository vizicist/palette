package winsys

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"strings"

	"github.com/vizicist/palette/engine"
	"golang.org/x/image/font"
)

// DrawTextMsg xxx
type DrawTextMsg struct {
	Text string
	Face font.Face
	Pos  image.Point
}

// NewDrawTextMsg xxx
func NewDrawTextCmd(text string, face font.Face, pos image.Point) engine.Cmd {
	arr, _ := engine.StringMap(fmt.Sprintf(`"text":"%s","face":"%s","x":"%d","y":"%d"`, text, face, pos.X, pos.Y))
	return engine.Cmd{Subj: "drawtext", Values: arr}
}

////////////////////////////////////////////////////////////////////

// NewDrawLineMsg xxx
func NewDrawLineCmd(xy0 image.Point, xy1 image.Point) engine.Cmd {
	arr, _ := engine.StringMap(fmt.Sprintf(`"x0":"%d","y0":"%d","x1":"%d","y1":"%d"`, xy0.Y, xy0.Y, xy1.X, xy1.Y))
	return engine.Cmd{Subj: "drawline", Values: arr}
}

//////////////////////////////////////////////////////////////////

// MouseMsg xxx
// type MouseMsg struct {
// 	Pos     image.Point
// 	ButtNum int
// 	Ddu     DownDragUp
// }

//////////////////////////////////////////////////////////////////

// NewCloseTransientsCmd xxx
func NewCloseTransientsCmd(menu string) engine.Cmd {
	arr, _ := engine.StringMap(fmt.Sprintf(`"menu":"%s"`, menu))
	return engine.Cmd{Subj: "closetransients", Values: arr}
}

// NewCloseMe xxx
func NewCloseMeCmd(menu string) engine.Cmd {
	arr, _ := engine.StringMap(fmt.Sprintf(`"menu":"%s"`, menu))
	return engine.Cmd{Subj: "closeme", Values: arr}
}

// NewMouseMsg xxx
func NewShowMouseCursorCmd(b bool) engine.Cmd {
	arr, _ := engine.StringMap(fmt.Sprintf(`"show":"%v"`, b))
	return engine.Cmd{Subj: "showmouse", Values: arr}
}

// NewMouseMsg xxx
func NewMouseCmd(ddu string, pos image.Point, bnum int) engine.Cmd {
	arr, _ := engine.StringMap(fmt.Sprintf(`"x":"%d","y":"%d","button":"%d"`, pos.X, pos.Y, bnum))
	return engine.Cmd{Subj: "mouse", Values: arr}
}

// NewDrawRectMsg xxx
func NewDrawRectCmd(rect image.Rectangle) engine.Cmd {
	arr, _ := engine.StringMap(fmt.Sprintf(`"minx":"%d","miny":"%d","maxx":"%d","maxy":"%d"`,
		rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y))
	return engine.Cmd{Subj: "drawrect", Values: arr}
}

// NewDrawFilledRectMsg xxx
func NewDrawFilledRectCmd(rect image.Rectangle) engine.Cmd {
	arr, _ := engine.StringMap(fmt.Sprintf(`"minx":"%d","miny":"%d","maxx":"%d","maxy":"%d"`, rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y))
	return engine.Cmd{Subj: "drawfilledrect", Values: arr}
}

// NewButtonDownMsg xxx
func NewButtonDownCmd(label string) engine.Cmd {
	arr, _ := engine.StringMap(fmt.Sprintf(`"label":"%s"`, label))
	return engine.Cmd{Subj: "buttondown", Values: arr}
}

// NewButtonUpMsg xxx
func NewButtonUpCmd(label string) engine.Cmd {
	arr, _ := engine.StringMap(fmt.Sprintf(`"label":"%s"`, label))
	return engine.Cmd{Subj: "buttonup", Values: arr}
}

// NewSetColorMsg xxx
func NewSetColorCmd(c color.RGBA) engine.Cmd {
	arr, _ := engine.StringMap(fmt.Sprintf(`"rgba":"%v"`, c))
	return engine.Cmd{Subj: "setcolor", Values: arr}

}

// NewResizeMsg xxx
func NewResizeCmd(size image.Point) engine.Cmd {
	arr, _ := engine.StringMap(fmt.Sprintf(`"x":"%d","y":"%d"`, size.X, size.Y))
	return engine.Cmd{Subj: "resize", Values: arr}
}

// NewResizeMeMsg xxx
func NewResizeMeCmd(size image.Point) engine.Cmd {
	arr, _ := engine.StringMap(fmt.Sprintf(`"x":"%d","y":"%d"`, size.X, size.Y))
	return engine.Cmd{Subj: "resizeme", Values: arr}
}

// NewAddLineMsg xxx
func NewAddLineCmd(line string) engine.Cmd {
	arr, _ := engine.StringMap(fmt.Sprintf(`"lines":"%s"`, line))
	return engine.Cmd{Subj: "addline", Values: arr}
}

// ShowMouseCursorMsg xxx
func NewShowMouseCursor(show bool) engine.Cmd {
	arr, _ := engine.StringMap(fmt.Sprintf(`"show":"%v"`, show))
	return engine.Cmd{Subj: "showmousecursor", Values: arr}
}

// MenuCallbackMsg xxx
type MenuCallbackMsg struct {
	MenuName string
	Arg      string
}

// NewPickToolCmd xxx
func NewPickToolCmd(action string) engine.Cmd {
	arr, _ := engine.StringMap(fmt.Sprintf(`"action":"%s"`, action))
	return engine.Cmd{Subj: "picktool", Values: arr}
}

// NewSweepToolCmd xxx
func NewSweepToolCmd(toolname string) engine.Cmd {
	arr, _ := engine.StringMap(fmt.Sprintf(`"toolname":"%s"`, toolname))
	return engine.Cmd{Subj: "sweeptool", Values: arr}
}

// NewMoveMenuCmd xxx
func NewMoveMenuCmd(menu string) engine.Cmd {
	arr, _ := engine.StringMap(fmt.Sprintf(`"menu":"%s"`, menu))
	return engine.Cmd{Subj: "movemenu", Values: arr}
}

// NewMakePermanentCmd xxx
func NewMakePermanentCmd(menuName string) engine.Cmd {
	arr, _ := engine.StringMap(fmt.Sprintf(`"menu":"%s"`, menuName))
	return engine.Cmd{Subj: "movemenu", Values: arr}
}

// NewSubMenuCmd xxx
func NewSubMenuCmd(submenu string) engine.Cmd {
	arr, _ := engine.StringMap(fmt.Sprintf(`"submenu":"%s"`, submenu))
	return engine.Cmd{Subj: "submenu", Values: arr}
}

// NewLogMsg xxx
func NewLogCmd(text string) engine.Cmd {
	arr, _ := engine.StringMap(fmt.Sprintf(`"text":"%s"`, text))
	return engine.Cmd{Subj: "log", Values: arr}
}

// NewAboutMsg xxx
func NewAboutCmd() engine.Cmd {
	return engine.Cmd{Subj: "about", Values: nil}
}

// NewDumpfileMsg xxx
func NewDumpFileCmd(filename string) engine.Cmd {
	arr, _ := engine.StringMap(fmt.Sprintf(`"filename":"%s"`, filename))
	return engine.Cmd{Subj: "dumpfile", Values: arr}
}

// NewRestorefileMsg xxx
func NewRestoreFileCmd(filename string) engine.Cmd {
	arr, _ := engine.StringMap(fmt.Sprintf(`"filename":"%s"`, filename))
	return engine.Cmd{Subj: "restorefile", Values: arr}
}

// NewRestorefileMsg xxx
func NewRestoreCmd(state string) engine.Cmd {
	arr, _ := engine.StringMap(fmt.Sprintf(`"state":"%s"`, state))
	return engine.Cmd{Subj: "restore", Values: arr}
}

// StateDataMsg xxx
type StateDataMsg struct {
	State string
}

// SweepCallbackMsg xxx
// type SweepCallbackMsg struct {
// 	// callbackWindow Window
// 	callbackMsg string
// 	callbackArg interface{}
// }

// DownDragUp xxx
// /type DownDragUp int

/*
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
func ToDrawLine(arg interface{}) DrawLineMsg {
	r, ok := arg.(DrawLineMsg)
	if !ok {
		log.Printf("Unable to convert interface to DrawLineMsg!\n")
		r = DrawLineMsg{}
	}
	return r
}

// ToDrawText xxx
func ToDrawText(arg interface{}) DrawTextMsg {
	r, ok := arg.(DrawTextMsg)
	if !ok {
		log.Printf("Unable to convert interface to DrawTextMsg!\n")
		r = DrawTextMsg{}
	}
	return r
}

// ToMouse xxx
func ToMouse(arg interface{}) MouseMsg {
	r, ok := arg.(MouseMsg)
	if !ok {
		log.Printf("Unable to convert interface to MouseMsg!\n")
		r = MouseMsg{}
	}
	return r
}
*/

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
	if arg == nil {
		return nil
	}
	r, ok := arg.(Window)
	if !ok {
		log.Printf("Unable to convert interface to Window!\n")
		r = nil
	}
	return r
}

// RectString xxx
func RectString(r image.Rectangle) string {
	return fmt.Sprintf("%d,%d,%d,%d", r.Min.X, r.Min.Y, r.Max.X, r.Max.Y)
}

// PointString xxx
func PointString(p image.Point) string {
	return fmt.Sprintf("%d,%d", p.X, p.Y)
}

// StringToRect xxx
func StringToRect(s string) (r image.Rectangle) {
	n, err := fmt.Sscanf(s, "%d,%d,%d,%d", &r.Min.X, &r.Min.Y, &r.Max.X, &r.Max.Y)
	if err != nil {
		log.Printf("StringRect: Bad format: %s\n", s)
		return image.Rectangle{}
	}
	if n != 4 {
		log.Printf("StringRect: Bad format: %s\n", s)
		return image.Rectangle{}
	}
	return r
}

// StringToPoint xxx
func StringToPoint(s string) (p image.Point) {
	n, err := fmt.Sscanf(s, "%d,%d", &p.X, &p.Y)
	if err != nil {
		log.Printf("StringPoint: Bad format: %s\n", s)
		return image.Point{}
	}
	if n != 2 {
		log.Printf("StringPoint: Bad format: %s\n", s)
		return image.Point{}
	}
	return p
}

// toPrettyJSON looks for {} and [] at the beginning/end of lines
// to control indenting
func toPrettyJSON(s string) string {
	// Make sure it ends in \n
	if s[len(s)-1] != '\n' {
		s = s + "\n"
	}
	from := 0
	indentSize := 4
	indent := 0
	r := ""
	slen := len(s)
	for from < slen {
		newlinePos := from + strings.Index(s[from:], "\n")
		line := s[from : newlinePos+1]

		// ignore initial whitespace, we're going to handle it
		for line[0] == ' ' {
			line = line[1:]
		}

		// See if we should un-indent before adding the line
		firstChar := line[0]
		if firstChar == '}' || firstChar == ']' {
			indent -= indentSize
			if indent < 0 {
				indent = 0
			}
		}

		// Add line with current indent
		r += strings.Repeat(" ", indent) + line

		// See if we should indent the next line more
		lastChar := s[newlinePos-1]
		if lastChar == '{' || lastChar == '[' {
			indent += indentSize
		}

		from = newlinePos + 1
	}
	return r
}
