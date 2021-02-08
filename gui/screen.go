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
	ctx         WinContext
	currentPage Window
	style       *Style
	rectx       image.Rectangle
	eimage      *ebiten.Image
	time0       time.Time
	lastprint   time.Time
	cursorPos   image.Point
	cursorStyle cursorStyle
	foreColor   color.RGBA
	backColor   color.RGBA
}

type cursorStyle int

const (
	normalCursorStyle = iota
	pickCursorStyle
	sweepCursorStyle
)

// CurrentPage xxx
var CurrentPage *Page

// Run runs the Gui and never returns
func Run() {

	// The user of palette.win can/should add custom tool types,
	// but we want to make sure that the standard ones are ours.
	// AddToolType("Menu", NewMenu)
	initStandardMenus()

	minSize := image.Point{640, 480}
	style := NewStyle("fixed", 16)

	ebiten.SetWindowSize(minSize.X, minSize.Y)
	ebiten.SetWindowResizable(true)
	ebiten.SetWindowTitle("Palette GUI (ebiten)")

	// The Screen is the only Window whose parent is nil
	screenCtx := NewWindowContext(nil)
	screen := &Screen{
		ctx:       screenCtx,
		eimage:    &ebiten.Image{},
		time0:     time.Now(),
		lastprint: time.Now(),
		foreColor: ForeColor,
		backColor: BackColor,
	}
	WinSetMySize(screen, minSize)
	screen.ctx.style = style

	td := NewPage(screen, "home")
	screen.currentPage = AddChild(screen, td)

	// This is it!  RunGame runs forever
	if err := ebiten.RunGame(screen); err != nil {
		log.Fatal(err)
	}
}

// Context xxx
func (screen *Screen) Context() *WinContext {
	return &screen.ctx
}

// Do xxx
func (screen *Screen) Do(cmd string, arg interface{}) (interface{}, error) {

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
	case "resizeme":
		size := ToPoint(arg)
		ebiten.SetWindowSize(size.X, size.Y)

	case "resize":

	default:
		log.Printf("screen.runCmds: UNRECOGNIZED cmd=%s\n", cmd)
	}
	return nil, nil
}

// Layout satisfies the ebiten.Game interface
func (screen *Screen) Layout(width, height int) (int, int) {
	if screen == nil {
		log.Printf("Screen.Layout: Hey, screen shouldn't be nil!\n")
		return width, height
	}
	currSize := WinCurrSize(screen)
	if currSize.X != width || currSize.Y != height {
		newSize := image.Point{width, height}
		WinSetMySize(screen, newSize)
		WinSetChildSize(screen.currentPage, newSize)
	}
	return width, height
}

// Update satisfies the ebiten.Game interface
func (screen *Screen) Update() (err error) {

	// log.Printf("Screen.Update! -------------------\n")
	x, y := ebiten.CursorPosition()
	newPos := image.Point{x, y}

	// Ignore updates outside the screen
	size := WinCurrSize(screen)
	rect := image.Rect(0, 0, size.X, size.Y)
	if !newPos.In(rect) {
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
			screen.currentPage.Do("mouse", MouseCmd{newPos, n, MouseDown})

		case inpututil.IsMouseButtonJustReleased(eb):
			screen.currentPage.Do("mouse", MouseCmd{newPos, n, MouseUp})

		default:
			// Drag events only happen when position changes
			if newPos.X != screen.cursorPos.X || newPos.Y != screen.cursorPos.Y {
				screen.cursorPos = newPos
				screen.currentPage.Do("mouse", MouseCmd{newPos, n, MouseDrag})
			}
		}
	}
	return nil
}

// Draw satisfies the ebiten.Game interface
func (screen *Screen) Draw(eimage *ebiten.Image) {
	screen.eimage = eimage
	screen.currentPage.Do("redraw", nil)
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
