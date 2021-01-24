package gui

import (
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
// Screen contains the pageWindow.
// Screen and Style should be the only things calling ebiten.
type Screen struct {
	WindowData
	page        Window
	style       *Style
	rect        image.Rectangle
	eimage      *ebiten.Image
	time0       time.Time
	lastprint   time.Time
	cursorPos   image.Point
	cursorStyle cursorStyle
	foreColor   color.RGBA
	backColor   color.RGBA
}

// MouseHandler xxx
// type MouseHandler func(image.Point, int, MouseCmd) bool

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
		page:      nil,
		style:     NewStyle("fixed", 16),
		rect:      image.Rectangle{},
		eimage:    &ebiten.Image{},
		time0:     time.Now(),
		lastprint: time.Now(),
		foreColor: white,
		backColor: black,
	}

	screen.page = NewPageWindow(screen)

	// This is it!  RunGame runs forever
	if err := ebiten.RunGame(screen); err != nil {
		log.Fatal(err)
	}
}

// Data xxx
func (screen *Screen) Data() *WindowData {
	return &screen.WindowData
}

// DoSync xxx
func (screen *Screen) DoSync(from Window, cmd string, arg interface{}) (result interface{}, err error) {
	return NoSyncInterface("Screen")
}

// Do xxx
func (screen *Screen) Do(from Window, cmd string, arg interface{}) {

	switch cmd {
	case "drawline":
		dl := ToDrawLine(arg)
		screen.drawLine(dl.XY0, dl.XY1)
	case "drawrect":
		screen.drawRect(ToRect(arg))
	case "drawfilledrect":
		screen.drawFilledRect(ToRect(arg))
	case "drawtext":
		t := ToDrawText(arg)
		screen.drawText(t.Text, t.Face, t.Pos)
	case "setcolor":
		screen.foreColor = ToColor(arg)
	case "showcursor":
		if ToBool(arg) {
			ebiten.SetCursorMode(ebiten.CursorModeVisible)
		} else {
			ebiten.SetCursorMode(ebiten.CursorModeHidden)
		}
	case "closeme":
		log.Printf("screen.runCmds: SHOULD NOT BE GETTING CloseMeCmd!?\n")
	default:
		log.Printf("screen.runCmds: UNRECOGNIZED cmd=%s\n", cmd)
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
		screen.page.Do(screen, "resize", screen.rect)
	}
	return width, height
}

// Update satisfies the ebiten.Game interface
func (screen *Screen) Update() (err error) {

	// log.Printf("Screen.Update! -------------------\n")
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

	// Do we really need to check all 3 mouse buttons?
	// Will more than 1 button ever be used?  Not sure.
	// pageinput := screen.page.Data().fromUpstream
	for n, eb := range butts {
		switch {
		case inpututil.IsMouseButtonJustPressed(eb):
			screen.page.Do(screen, "mouse", MouseCmd{newPos, n, MouseDown})

		case inpututil.IsMouseButtonJustReleased(eb):
			screen.page.Do(screen, "mouse", MouseCmd{newPos, n, MouseUp})

		default:
			// Drag events only happen when position changes
			if newPos.X != screen.cursorPos.X || newPos.Y != screen.cursorPos.Y {
				screen.cursorPos = newPos
				screen.page.Do(screen, "mouse", MouseCmd{newPos, n, MouseDrag})
			}
		}
	}
	return nil
}

// Draw satisfies the ebiten.Game interface
func (screen *Screen) Draw(eimage *ebiten.Image) {

	screen.eimage = eimage

	screen.page.Do(screen, "redraw", nil)

	/*
		pos := screen.cursorPos
		switch screen.cursorStyle {
		case normalCursorStyle:
		case pickCursorStyle:
			// Draw a cross at the cursor position
			delta := 10
			screen.drawLine(image.Point{pos.X - delta, pos.Y}, image.Point{pos.X + delta, pos.Y}, screen.foreColor)
			screen.drawLine(image.Point{pos.X, pos.Y - delta}, image.Point{pos.X, pos.Y + delta}, screen.foreColor)
		case sweepCursorStyle:
			// Draw a cross at the cursor position
			delta := 10
			screen.drawLine(image.Point{pos.X - delta, pos.Y - delta}, image.Point{pos.X + delta, pos.Y - delta}, screen.foreColor)
			screen.drawLine(image.Point{pos.X - delta, pos.Y - delta}, image.Point{pos.X - delta, pos.Y + delta}, screen.foreColor)
		}
	*/

}

// drawRect xxx
func (screen *Screen) drawRect(rect image.Rectangle) {
	x0 := rect.Min.X
	y0 := rect.Min.Y
	x1 := rect.Max.X
	y1 := rect.Max.Y
	screen.drawLine(image.Point{x0, y0}, image.Point{x1, y0})
	screen.drawLine(image.Point{x1, y0}, image.Point{x1, y1})
	screen.drawLine(image.Point{x1, y1}, image.Point{x0, y1})
	screen.drawLine(image.Point{x0, y1}, image.Point{x0, y0})
}

// drawLine xxx
func (screen *Screen) drawLine(xy0, xy1 image.Point) {
	ebitenutil.DrawLine(screen.eimage,
		float64(xy0.X), float64(xy0.Y), float64(xy1.X), float64(xy1.Y), screen.foreColor)
}

func (screen *Screen) drawText(s string, face font.Face, pos image.Point) {
	text.Draw(screen.eimage, s, face, pos.X, pos.Y, screen.foreColor)
}

func (screen *Screen) drawFilledRect(rect image.Rectangle) {
	w := rect.Max.X - rect.Min.X
	h := rect.Max.Y - rect.Min.Y
	ebitenutil.DrawRect(screen.eimage, float64(rect.Min.X), float64(rect.Min.Y), float64(w), float64(h), screen.foreColor)
}
