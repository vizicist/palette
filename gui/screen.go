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
	root      *Root
	style     *Style
	rect      image.Rectangle
	eimage    *ebiten.Image
	time0     time.Time
	lastprint time.Time
	cursorPos image.Point
}

// Run displays and runs the Gui and never returns
func Run() {

	ebiten.SetWindowSize(600, 800)
	ebiten.SetWindowResizable(true)
	ebiten.SetWindowTitle("Palette GUI (ebiten)")

	screen := &Screen{
		root:      nil,
		style:     NewStyle("fixed", 16),
		rect:      image.Rectangle{},
		eimage:    &ebiten.Image{},
		time0:     time.Now(),
		lastprint: time.Now(),
	}
	screen.root = NewRoot(screen)

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
		screen.Resize(width, height)
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

	for n, eb := range butts {
		switch {
		case inpututil.IsMouseButtonJustPressed(eb):
			screen.root.HandleMouseInput(newPos, n, MouseDown)
		case inpututil.IsMouseButtonJustReleased(eb):
			screen.root.HandleMouseInput(newPos, n, MouseUp)
		default:
			// Drag events only happen when position changes
			if newPos.X != screen.cursorPos.X || newPos.Y != screen.cursorPos.Y {
				screen.cursorPos = newPos
				screen.root.HandleMouseInput(newPos, n, MouseDrag)
			}

		}
	}
	return nil
}

// Draw satisfies the ebiten.Game interface
func (screen *Screen) Draw(eimage *ebiten.Image) {
	screen.eimage = eimage
	// red border, will be removed eventually
	screen.drawRect(screen.rect, color.RGBA{0xff, 0x0, 0x0, 0xff})
	screen.root.Draw()
}

// Resize xxx
func (screen *Screen) Resize(newWidth, newHeight int) {
	screen.rect = image.Rect(0, 0, newWidth, newHeight)
	screen.root.Resize(screen.rect)
}

// HandleMouseInput xxx
func (screen *Screen) HandleMouseInput(pos image.Point, button int, event MouseEvent) {
	screen.root.HandleMouseInput(pos, button, event)
}

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
