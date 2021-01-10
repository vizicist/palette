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

// Screen contains the rootWindow, with interfaces to the real display,
// currently managed by ebiten.  The Screen and Style
// structs are the only one that should be calling ebiten
type Screen struct {
	root      *Root
	rect      image.Rectangle
	eimage    *ebiten.Image
	time0     time.Time
	lastprint time.Time
}

// Run xxx
func Run() {
	ebiten.SetWindowSize(600, 800)
	ebiten.SetWindowResizable(true)
	ebiten.SetWindowTitle("Palette GUI (ebiten)")
	screen := NewScreen()
	if err := ebiten.RunGame(screen); err != nil {
		log.Fatal(err)
	}
}

// NewScreen xxx
func NewScreen() *Screen {

	screen := &Screen{
		root:      nil,
		rect:      image.Rectangle{},
		eimage:    &ebiten.Image{},
		time0:     time.Now(),
		lastprint: time.Now(),
	}

	root := NewRoot(NewStyle("regular", 32))

	// Add initial contents of RootWindow, for testing
	root.console = NewConsole(NewStyle("mono", 32))
	AddObject(root.objects, "console1", root.console)

	// root.console2 = NewConsole(NewStyle("mono", 32))
	// AddObject(root.objects, "console2", root.console2)

	screen.root = root

	return screen
}

// Resize xxx
func (screen *Screen) Resize(newWidth, newHeight int) {
	screen.rect = image.Rect(0, 0, newWidth, newHeight)
	nrect := screen.rect.Inset(10)
	screen.root.Resize(nrect)
}

// SetImage xxx
func (screen *Screen) SetImage(eimage *ebiten.Image) {
	screen.eimage = eimage
}

// Layout satisfies the ebiten.Game interface
func (screen *Screen) Layout(outsideWidth, outsideHeight int) (int, int) {
	if screen == nil {
		log.Printf("Screen.Layout: Hey, screen shouldn't be nil!\n")
		return outsideWidth, outsideHeight
	}
	if screen.rect.Dx() != outsideWidth || screen.rect.Dy() != outsideHeight {
		screen.Resize(outsideWidth, outsideHeight)
	}
	return outsideWidth, outsideHeight
}

// Update xxx
func (screen *Screen) Update() (err error) {
	type butt struct {
		b  int
		eb ebiten.MouseButton
	}
	// This is essentially a map from 0,1,2 to the ebiten values
	butts := []ebiten.MouseButton{
		ebiten.MouseButtonLeft,
		ebiten.MouseButtonRight,
		ebiten.MouseButtonMiddle,
	}

	/*
		now := time.Now()
		dt := int(now.Sub(screen.lastprint).Seconds())
		if dt > 0 {
			s := fmt.Sprintf("now=%v", now)
			screen.root.console.AddLine(s)
			screen.lastprint = now
		}
	*/

	for n, eb := range butts {

		buttNum := -1
		isPressed := false
		switch {
		case inpututil.IsMouseButtonJustPressed(eb):
			buttNum = n
			isPressed = true
		case inpututil.IsMouseButtonJustReleased(eb):
			buttNum = n
		}
		if buttNum >= 0 {
			x, y := ebiten.CursorPosition()
			pos := image.Point{x, y}
			screen.HandleMouseInput(pos, buttNum, isPressed)
		}
	}
	return nil
}

// Draw xxx
func (screen *Screen) Draw(eimage *ebiten.Image) {

	screen.SetImage(eimage)

	// black := color.RGBA{0x0, 0x0, 0x0, 0xff}
	// background := color.RGBA{0xfa, 0xf8, 0xef, 0xff}
	// white := color.RGBA{0xff, 0xff, 0xff, 0xff}
	red := color.RGBA{0xff, 0x0, 0x0, 0xff}

	screen.DrawRect(screen.rect, red)
	screen.root.Draw(screen)

	/*
		for _, o := range screen.root.objects {
			o.Draw(screen)
		}
	*/
}

// HandleMouseInput xxx
func (screen *Screen) HandleMouseInput(pos image.Point, button int, down bool) {
	screen.root.HandleMouseInput(pos, button, down)
}

// DrawRect xxx
func (screen *Screen) DrawRect(rect image.Rectangle, color color.RGBA) {
	x0 := rect.Min.X
	y0 := rect.Min.Y
	x1 := rect.Max.X
	y1 := rect.Max.Y
	screen.DrawLine(x0, y0, x1, y0, color)
	screen.DrawLine(x1, y0, x1, y1, color)
	screen.DrawLine(x1, y1, x0, y1, color)
	screen.DrawLine(x0, y1, x0, y0, color)
}

// DrawLine xxx
func (screen *Screen) DrawLine(x0, y0, x1, y1 int, color color.RGBA) {
	ebitenutil.DrawLine(screen.eimage,
		float64(x0), float64(y0), float64(x1), float64(y1), color)
}

func (screen *Screen) drawText(s string, face font.Face, x int, y int, color color.Color) {
	text.Draw(screen.eimage, s, face, x, y, color)
}
