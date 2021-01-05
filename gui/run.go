package gui

import (
	"image"
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Gui xxx
type Gui struct {
	screen *Screen
}

// Draw xxx
func (g *Gui) Draw(eimage *ebiten.Image) {

	g.screen.SetImage(eimage) // in case it changes
	g.screen.Draw()
}

// Layout xxx
func (g *Gui) Layout(outsideWidth, outsideHeight int) (int, int) {
	if g.screen != nil {
		if g.screen.rect.Dx() != outsideWidth || g.screen.rect.Dy() != outsideHeight {
			g.screen.Resize(outsideWidth, outsideHeight)
		}
	}
	return outsideWidth, outsideHeight
}

// Update xxx
func (g *Gui) Update() (err error) {
	// ebitenutil.DebugPrint(screen, "Hello, World!")
	x, y := ebiten.CursorPosition()
	pos := image.Point{x, y}

	type butt struct {
		b  int
		eb ebiten.MouseButton
	}
	butts := []ebiten.MouseButton{
		ebiten.MouseButtonLeft,
		ebiten.MouseButtonRight,
		ebiten.MouseButtonMiddle,
	}

	for n, eb := range butts {
		switch {
		case inpututil.IsMouseButtonJustPressed(eb):
			g.screen.HandleMouseInput(pos, n, true)
		case inpututil.IsMouseButtonJustReleased(eb):
			g.screen.HandleMouseInput(pos, n, false)
		}
	}
	return nil
}

// Run xxx
func Run() {
	ebiten.SetWindowSize(600, 800)
	ebiten.SetWindowTitle("Text (Ebiten Demo)")
	ebiten.SetWindowResizable(true)
	ebiten.SetWindowTitle("Palette GUI (ebiten)")
	gui := &Gui{
		screen: NewScreen(),
	}
	if err := ebiten.RunGame(gui); err != nil {
		log.Fatal(err)
	}
}
