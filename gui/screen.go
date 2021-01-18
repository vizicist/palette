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
	cmdChan      chan WinOutput
	page         *Page
	style        *Style
	rect         image.Rectangle
	eimage       *ebiten.Image
	time0        time.Time
	lastprint    time.Time
	cursorPos    image.Point
	cursorStyle  cursorStyle
	mouseHandler chan MouseCmd
	foreColor    color.RGBA
	backColor    color.RGBA
}

// MouseHandler xxx
type MouseHandler func(image.Point, int, MouseCmd) bool

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
		cmdChan:      make(chan WinOutput),
		page:         nil,
		style:        NewStyle("fixed", 16),
		rect:         image.Rectangle{},
		eimage:       &ebiten.Image{},
		time0:        time.Now(),
		lastprint:    time.Now(),
		mouseHandler: nil,
		foreColor:    white,
		backColor:    black,
	}
	screen.page = NewPageWindow(screen)

	go screen.runCmds()

	// This is it!  RunGame runs forever
	if err := ebiten.RunGame(screen); err != nil {
		log.Fatal(err)
	}
}

func (screen *Screen) runCmds() {

	for {
		select {

		case cmd := <-screen.cmdChan:

			switch c := cmd.(type) {
			case DrawLineCmd:
				screen.drawLine(c.XY0, c.XY1, c.Color)
			case DrawRectCmd:
				screen.drawRect(c.Rect, c.Color)
			case DrawFilledRectCmd:
				screen.drawFilledRect(c.Rect, c.Color)
			case DrawTextCmd:
				screen.drawText(c.Text, c.Face, c.Pos, c.Color)
			case CloseMeCmd:
				log.Printf("!!!!! Got CloseMeCmd c=%v", c)
				if c.W == screen.page.menu {
					log.Print("CloseMeCmd is Clearing page.menu\n")
					screen.page.menu = nil
				}
				RemoveWindow(screen.page, c.W)
			}

		default:
			time.Sleep(time.Millisecond)
		}
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
		screen.page.Resize(screen.rect)
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

	// Do we really need to check all 3 mouse buttons?
	// Will more than 1 button ever be used?  Not sure.
	for n, eb := range butts {
		switch {
		case inpututil.IsMouseButtonJustPressed(eb):
			me := MouseCmd{newPos, n, MouseDown}
			screen.page.InChan <- me

		case inpututil.IsMouseButtonJustReleased(eb):
			me := MouseCmd{newPos, n, MouseUp}
			screen.page.InChan <- me

		default:
			// Drag events only happen when position changes
			if newPos.X != screen.cursorPos.X || newPos.Y != screen.cursorPos.Y {
				screen.cursorPos = newPos
				me := MouseCmd{newPos, n, MouseDrag}
				screen.page.InChan <- me
			}
		}
	}
	return nil
}

// Draw satisfies the ebiten.Game interface
func (screen *Screen) Draw(eimage *ebiten.Image) {
	screen.eimage = eimage
	screen.page.Draw()

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

// drawRect xxx
func (screen *Screen) drawRect(rect image.Rectangle, clr color.RGBA) {
	x0 := rect.Min.X
	y0 := rect.Min.Y
	x1 := rect.Max.X
	y1 := rect.Max.Y
	screen.drawLine(image.Point{x0, y0}, image.Point{x1, y0}, clr)
	screen.drawLine(image.Point{x1, y0}, image.Point{x1, y1}, clr)
	screen.drawLine(image.Point{x1, y1}, image.Point{x0, y1}, clr)
	screen.drawLine(image.Point{x0, y1}, image.Point{x0, y0}, clr)
}

// drawLine xxx
func (screen *Screen) drawLine(xy0, xy1 image.Point, color color.RGBA) {
	ebitenutil.DrawLine(screen.eimage,
		float64(xy0.X), float64(xy0.Y), float64(xy1.X), float64(xy1.Y), color)
}

func (screen *Screen) drawText(s string, face font.Face, pos image.Point, clr color.Color) {
	text.Draw(screen.eimage, s, face, pos.X, pos.Y, clr)
}

func (screen *Screen) drawFilledRect(rect image.Rectangle, clr color.Color) {
	w := rect.Max.X - rect.Min.X
	h := rect.Max.Y - rect.Min.Y
	ebitenutil.DrawRect(screen.eimage, float64(rect.Min.X), float64(rect.Min.Y), float64(w), float64(h), clr)
}
