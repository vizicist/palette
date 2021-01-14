package gui

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
)

// Screen satisfies the ebiten.Game interface.
// Screen contains the rootWindow.
// Screen and Style should be the only things calling ebiten.
type Screen struct {
	root         *Root
	style        *Style
	rect         image.Rectangle
	eimage       *ebiten.Image
	time0        time.Time
	lastprint    time.Time
	cursorPos    image.Point
	cursorStyle  cursorStyle
	mouseHandler chan MouseEvent
	foreColor    color.RGBA
	backColor    color.RGBA
}

// MouseHandler xxx
type MouseHandler func(image.Point, int, MouseEvent) bool

type cursorStyle int

const (
	normalCursorStyle = iota
	pickCursorStyle
	sweepCursorStyle
)

// Run displays and runs the Gui and never returns
func Run() {

	ebiten.SetWindowSize(600, 800)
	ebiten.SetWindowResizable(true)
	ebiten.SetWindowTitle("Palette GUI (ebiten)")

	screen := &Screen{
		root:         nil,
		style:        NewStyle("fixed", 16),
		rect:         image.Rectangle{},
		eimage:       &ebiten.Image{},
		time0:        time.Now(),
		lastprint:    time.Now(),
		mouseHandler: nil,
		foreColor:    white,
		backColor:    black,
	}
	screen.root = NewRoot(screen)

	// This is it!  RunGame runs forever
	if err := ebiten.RunGame(screen); err != nil {
		log.Fatal(err)
	}
}

// Layout satisfies the ebiten.Game interface
func (screen *Screen) Layout(width, height int) (int, int) {
	if screen == nil {
		log.Printf("Screen.Layout: Hey, screen shouldn't be nil!\n")
		return width, height
	}
	if screen.rect.Dx() != width || screen.rect.Dy() != height {
		screen.rect = image.Rect(0, 0, width, height)
		screen.root.Resize(screen.rect)
	}
	return width, height
}

// Update satisfies the ebiten.Game interface
func (screen *Screen) Update() (err error) {

	x, y := ebiten.CursorPosition()
	newPos := image.Point{x, y}

	// Ignore updates outside the screen
	if !newPos.In(screen.rect) {
		return nil
	}

	// This array is a map from 0,1,2 to the ebiten values
	butts := []ebiten.MouseButton{
		ebiten.MouseButtonLeft,
		ebiten.MouseButtonRight,
		ebiten.MouseButtonMiddle,
	}

	// screen.log("Update: checking mouse now=%v", time.Now())

	// Do we really need to check all 3 mouse buttons?
	// Will more than 1 button ever be used?  Not sure.
	for n, eb := range butts {
		switch {
		case inpututil.IsMouseButtonJustPressed(eb):
			me := MouseEvent{newPos, n, MouseDown}
			screen.root.mouseChan <- me
		case inpututil.IsMouseButtonJustReleased(eb):
			me := MouseEvent{newPos, n, MouseUp}
			screen.root.mouseChan <- me
		default:
			// Drag events only happen when position changes
			if newPos.X != screen.cursorPos.X || newPos.Y != screen.cursorPos.Y {
				screen.cursorPos = newPos
				me := MouseEvent{newPos, n, MouseDrag}
				screen.root.mouseChan <- me
			}
		}
	}
	return nil
}

// Draw satisfies the ebiten.Game interface
func (screen *Screen) Draw(eimage *ebiten.Image) {
	screen.eimage = eimage
	screen.root.Draw()

	pos := screen.cursorPos
	switch screen.cursorStyle {
	case normalCursorStyle:
	case pickCursorStyle:
		// Draw a cross at the cursor position
		delta := 10
		screen.drawLine(pos.X-delta, pos.Y, pos.X+delta, pos.Y, screen.foreColor)
		screen.drawLine(pos.X, pos.Y-delta, pos.X, pos.Y+delta, screen.foreColor)
	case sweepCursorStyle:
		// Draw a cross at the cursor position
		delta := 10
		screen.drawLine(pos.X-delta, pos.Y-delta, pos.X+delta, pos.Y-delta, screen.foreColor)
		screen.drawLine(pos.X-delta, pos.Y-delta, pos.X-delta, pos.Y+delta, screen.foreColor)
	}

}

func (screen *Screen) setCursorStyle(style cursorStyle) {
	screen.cursorStyle = style
	if style == normalCursorStyle {
		ebiten.SetCursorMode(ebiten.CursorModeVisible)
	} else {
		// We draw it ourselves
		ebiten.SetCursorMode(ebiten.CursorModeHidden)
	}
}

/*
func (screen *Screen) spawnMouseHandler(handler MouseHandler) {
	// take over all mouse handling until releaseMouse is called
	screen.mouseHandler = handler
}
func (screen *Screen) setDefaultMouseHandler() {
	// give it back to the root window
	screen.mouseHandler = screen.root.HandleMouseInput
}
*/

// drawRect xxx
func (screen *Screen) drawRect(rect image.Rectangle, clr color.RGBA) {
	x0 := rect.Min.X
	y0 := rect.Min.Y
	x1 := rect.Max.X
	y1 := rect.Max.Y
	screen.drawLine(x0, y0, x1, y0, clr)
	screen.drawLine(x1, y0, x1, y1, clr)
	screen.drawLine(x1, y1, x0, y1, clr)
	screen.drawLine(x0, y1, x0, y0, clr)
}

// drawLine xxx
func (screen *Screen) drawLine(x0, y0, x1, y1 int, color color.RGBA) {
	ebitenutil.DrawLine(screen.eimage,
		float64(x0), float64(y0), float64(x1), float64(y1), color)
}

func (screen *Screen) drawText(s string, face font.Face, x int, y int, clr color.Color) {
	text.Draw(screen.eimage, s, face, x, y, clr)
}

func (screen *Screen) drawFilledRect(rect image.Rectangle, clr color.Color) {
	w := rect.Max.X - rect.Min.X
	h := rect.Max.Y - rect.Min.Y
	ebitenutil.DrawRect(screen.eimage, float64(rect.Min.X), float64(rect.Min.Y), float64(w), float64(h), clr)
}

// func (screen *Screen) log(s string) {
// 	screen.root.log(s)
// }

func (screen *Screen) log(format string, args ...interface{}) {
	screen.root.log(fmt.Sprintf(format, args...))
}
