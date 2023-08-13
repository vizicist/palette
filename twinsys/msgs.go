package twinsys

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/vizicist/palette/hostwin"
	"golang.org/x/image/font"
)

// DrawTextMsg xxx
type DrawTextMsg struct {
	Text string
	Face font.Face
	Pos  image.Point
}

// NewDrawTextMsg xxx
func NewDrawTextCmd(text string, styleName string, pos image.Point) hostwin.Cmd {
	arr, _ := hostwin.StringMap(fmt.Sprintf(`{"text":"%s","style":"%s","pos":"%d,%d"}`, text, styleName, pos.X, pos.Y))
	return hostwin.Cmd{Subj: "drawtext", Values: arr}
}

////////////////////////////////////////////////////////////////////

// NewDrawLineMsg xxx
func NewDrawLineCmd(xy0 image.Point, xy1 image.Point) hostwin.Cmd {
	arr, _ := hostwin.StringMap(fmt.Sprintf(`{"xy0":"%d,%d","xy1":"%d,%d"}`, xy0.X, xy0.Y, xy1.X, xy1.Y))
	return hostwin.Cmd{Subj: "drawline", Values: arr}
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
func NewCloseTransientsCmd(exceptMenuName string) hostwin.Cmd {
	arr, _ := hostwin.StringMap(fmt.Sprintf(`{"exceptmenu":"%s"}`, exceptMenuName))
	return hostwin.Cmd{Subj: "closetransients", Values: arr}
}

// NewCloseMe xxx
func NewCloseMeCmd(menu string) hostwin.Cmd {
	arr, _ := hostwin.StringMap(fmt.Sprintf(`{"menu":"%s"}`, menu))
	return hostwin.Cmd{Subj: "closeme", Values: arr}
}

// NewMouseMsg xxx
func NewShowMouseCursorCmd(b bool) hostwin.Cmd {
	arr, _ := hostwin.StringMap(fmt.Sprintf(`{"show":"%v"}`, b))
	return hostwin.Cmd{Subj: "showmousecursor", Values: arr}
}

// NewMouseMsg xxx
func NewMouseCmd(ddu string, pos image.Point, bnum int) hostwin.Cmd {
	arr, _ := hostwin.StringMap(fmt.Sprintf(`{"ddu":"%s","pos":"%d,%d","button":"%d"}`, ddu, pos.X, pos.Y, bnum))
	return hostwin.Cmd{Subj: "mouse", Values: arr}
}

// NewDrawRectMsg xxx
func NewDrawRectCmd(rect image.Rectangle) hostwin.Cmd {
	arr, _ := hostwin.StringMap(fmt.Sprintf(`{"xy0":"%d,%d","xy1":"%d,%d"}`,
		rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y))
	return hostwin.Cmd{Subj: "drawrect", Values: arr}
}

// NewDrawFilledRectMsg xxx
func NewDrawFilledRectCmd(rect image.Rectangle) hostwin.Cmd {
	arr, _ := hostwin.StringMap(fmt.Sprintf(`{"xy0":"%d,%d","xy1":"%d,%d"}`, rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y))
	return hostwin.Cmd{Subj: "drawfilledrect", Values: arr}
}

// NewButtonDownMsg xxx
func NewButtonDownCmd(label string) hostwin.Cmd {
	arr, _ := hostwin.StringMap(fmt.Sprintf(`{"label":"%s"}`, label))
	return hostwin.Cmd{Subj: "buttondown", Values: arr}
}

// NewButtonUpMsg xxx
func NewButtonUpCmd(label string) hostwin.Cmd {
	arr, _ := hostwin.StringMap(fmt.Sprintf(`{"label":"%s"}`, label))
	return hostwin.Cmd{Subj: "buttonup", Values: arr}
}

// NewSetColorMsg xxx
func NewSetColorCmd(c color.RGBA) hostwin.Cmd {
	arr, _ := hostwin.StringMap(fmt.Sprintf(`{"r":"%d","g":"%d","b":"%d","a":"%d"}`, c.R, c.G, c.B, c.A))
	return hostwin.Cmd{Subj: "setcolor", Values: arr}

}

// NewResizeMsg xxx
func NewResizeCmd(size image.Point) hostwin.Cmd {
	arr, _ := hostwin.StringMap(fmt.Sprintf(`{"size":"%d,%d"}`, size.X, size.Y))
	return hostwin.Cmd{Subj: "resize", Values: arr}
}

// NewResizeMeMsg xxx
func NewResizeMeCmd(size image.Point) hostwin.Cmd {
	arr, _ := hostwin.StringMap(fmt.Sprintf(`{"size":"%d,%d"}`, size.X, size.Y))
	return hostwin.Cmd{Subj: "resizeme", Values: arr}
}

// NewAddLineMsg xxx
func NewAddLineCmd(line string) hostwin.Cmd {
	n := strings.Index(line, "\n")
	if n >= 0 {
		line = line[0:n]
	}
	arr, _ := hostwin.StringMap(fmt.Sprintf(`{"line":"%s"}`, line))
	return hostwin.Cmd{Subj: "addline", Values: arr}
}

// ShowMouseCursorMsg xxx
// func NewShowMouseCursor(show bool) hostwin.Cmd {
// 	arr, _ := hostwin.StringMap(fmt.Sprintf(`{"show":"%v"}`, show))
// 	return hostwin.Cmd{Subj: "showmousecursor", Values: arr}
// }

// MenuCallbackMsg xxx
type MenuCallbackMsg struct {
	MenuName string
	Arg      string
}

// NewPickToolCmd xxx
func NewPickToolCmd(action string) hostwin.Cmd {
	arr, _ := hostwin.StringMap(fmt.Sprintf(`{"action":"%s"}`, action))
	return hostwin.Cmd{Subj: "picktool", Values: arr}
}

// NewSweepToolCmd xxx
func NewSweepToolCmd(toolname string) hostwin.Cmd {
	arr, _ := hostwin.StringMap(fmt.Sprintf(`{"toolname":"%s"}`, toolname))
	return hostwin.Cmd{Subj: "sweeptool", Values: arr}
}

// NewMoveMenuCmd xxx
func NewMoveMenuCmd(menu string) hostwin.Cmd {
	arr, _ := hostwin.StringMap(fmt.Sprintf(`{"menu":"%s"}`, menu))
	return hostwin.Cmd{Subj: "movemenu", Values: arr}
}

// NewMakePermanentCmd xxx
func NewMakePermanentCmd(menuName string) hostwin.Cmd {
	arr, _ := hostwin.StringMap(fmt.Sprintf(`{"menu":"%s"}`, menuName))
	return hostwin.Cmd{Subj: "movemenu", Values: arr}
}

// NewSubMenuCmd xxx
func NewSubMenuCmd(submenu string) hostwin.Cmd {
	arr, _ := hostwin.StringMap(fmt.Sprintf(`{"submenu":"%s"}`, submenu))
	return hostwin.Cmd{Subj: "submenu", Values: arr}
}

// NewLogMsg xxx
func NewLogCmd(text string) hostwin.Cmd {
	arr, _ := hostwin.StringMap(fmt.Sprintf(`{"text":"%s"}`, text))
	return hostwin.Cmd{Subj: "log", Values: arr}
}

// NewAboutMsg xxx
func NewAboutCmd() hostwin.Cmd {
	return hostwin.Cmd{Subj: "about", Values: nil}
}

// NewDumpfileMsg xxx
func NewDumpFileCmd(filename string) hostwin.Cmd {
	arr, _ := hostwin.StringMap(fmt.Sprintf(`{"filename":"%s"}`, filename))
	return hostwin.Cmd{Subj: "dumpfile", Values: arr}
}

// NewRestorefileMsg xxx
func NewRestoreFileCmd(filename string) hostwin.Cmd {
	arr, _ := hostwin.StringMap(fmt.Sprintf(`{"filename":"%s"}`, filename))
	return hostwin.Cmd{Subj: "restorefile", Values: arr}
}

// NewRestorefileMsg xxx
func NewRestoreCmd(state string) hostwin.Cmd {
	arr, _ := hostwin.StringMap(fmt.Sprintf(`{"state":"%s"}`, state))
	return hostwin.Cmd{Subj: "restore", Values: arr}
}

// StateDataMsg xxx
type StateDataMsg struct {
	State string
}

// SweepCallbackMsg xxx
// type SweepCallbackMsg struct {
// 	// callbackWindow Window
// 	callbackMsg string
// 	callbackArg any
// }

// DownDragUp xxx
// /type DownDragUp int

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
		hostwin.LogIfError(err)
		return image.Rectangle{}
	}
	if n != 4 {
		hostwin.LogWarn("StringRect: Bad format", "s", s)
		return image.Rectangle{}
	}
	return r
}

// StringToPoint converts strings of the form "###,###" to an image.Point
func StringToPoint(s string) (p image.Point) {
	n, err := fmt.Sscanf(s, "%d,%d", &p.X, &p.Y)
	if err != nil {
		hostwin.LogIfError(err)
		return image.Point{}
	}
	if n != 2 {
		hostwin.LogWarn("StringPoint: Bad format", "s", s)
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
