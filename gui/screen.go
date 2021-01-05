package gui

import (
	"image"
	"image/color"

	// Don't be tempted to use go-gl
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// Screen contains the rootWindow, with interfaces to the real display
type Screen struct {
	root   *Root
	rect   image.Rectangle
	eimage *ebiten.Image
}

// NewScreen xxx
func NewScreen() *Screen {

	screen := &Screen{
		root:   nil,
		rect:   image.Rectangle{},
		eimage: &ebiten.Image{},
	}

	root := NewRoot(NewStyle("regular", 32))

	// Add initial contents of RootWindow, for testing
	root.console = NewConsole(NewStyle("mono", 20))
	AddObject(root.objects, "console1", root.console)

	root.console2 = NewConsole(NewStyle("mono", 32))
	AddObject(root.objects, "console2", root.console2)

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

// DrawLine xxx
func (screen *Screen) DrawLine(x0, y0, x1, y1 int, color color.RGBA) {
	ebitenutil.DrawLine(screen.eimage,
		float64(x0), float64(y0), float64(x1), float64(y1), color)
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

// Draw xxx
func (screen *Screen) Draw() {

	// black := color.RGBA{0x0, 0x0, 0x0, 0xff}
	// background := color.RGBA{0xfa, 0xf8, 0xef, 0xff}
	// white := color.RGBA{0xff, 0xff, 0xff, 0xff}
	red := color.RGBA{0xff, 0x0, 0x0, 0xff}

	screen.DrawRect(screen.rect, red)

	for _, o := range screen.root.objects {
		o.Draw(screen)
	}
}

// HandleMouseInput xxx
func (screen *Screen) HandleMouseInput(pos image.Point, button int, down bool) {
	o := ObjectUnder(screen.root.objects, pos)
	if o != nil {
		o.HandleMouseInput(pos, button, down)
	}
}
