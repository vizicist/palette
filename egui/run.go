package egui

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
)

const (
	screenWidth  = 600
	screenHeight = 800
)

var (
	mplusNormalFont font.Face
	mplusBigFont    font.Face
)

func init() {
	tt, err := opentype.Parse(fonts.MPlus1pRegular_ttf)
	if err != nil {
		log.Fatal(err)
	}

	const dpi = 72
	mplusNormalFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    24,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}
	mplusBigFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    32,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Gui xxx
type Gui struct {
	counter        int
	kanjiText      []rune
	kanjiTextColor color.RGBA
	screen         *Screen
}

// Draw xxx
func (g *Gui) Draw(eimage *ebiten.Image) {

	// gray := color.RGBA{0x80, 0x80, 0x80, 0xff}

	/*
		{
			const x, y = 20, 40
			b := text.BoundString(mplusNormalFont, sampleText)
			ebitenutil.DrawRect(eimage, float64(b.Min.X+x), float64(b.Min.Y+y), float64(b.Dx()), float64(b.Dy()), gray)
			text.Draw(eimage, sampleText, mplusNormalFont, x, y, color.White)
		}
	*/

	g.screen.SetImage(eimage) // in case it changes
	g.screen.Draw()
}

// Layout xxx
func (g *Gui) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// Update xxx
func (g *Gui) Update() (err error) {
	// Wait until the first Update, so everything including
	// the graphics state is running
	if g.screen == nil {
		g.screen, err = NewScreen(600, 800)
		if err != nil {
			log.Printf("NewScreen: Unable to create, err=%s\n", err)
			return fmt.Errorf("egui.update: unable to create screen!?")
		}
	}
	// if ebiten.IsDrawingSkipped() {
	// 	return na
	// }
	// ebitenutil.DebugPrint(screen, "Hello, World!")
	return nil
}

// Run xxx
func Run() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Text (Ebiten Demo)")
	ebiten.SetWindowResizable(true)
	ebiten.SetWindowTitle("Palette GUI (ebiten)")
	if err := ebiten.RunGame(&Gui{}); err != nil {
		log.Fatal(err)
	}
}

/*
func (runcontext *RunContext) gotMouseButton(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	if button < 0 || int(button) >= len(runcontext.screen.mouseButtonDown) {
		log.Printf("mousebutton: unexpected value button=%d\n", button)
		return
	}
	down := false
	if action == 1 {
		down = true
	}
	runcontext.screen.QueueMouseEvent(MouseEvent{
		pos:      runcontext.glfwMousePos,
		button:   int(button),
		down:     down,
		modifier: int(mods),
	})
}

func (runcontext *RunContext) gotMousePos(w *glfw.Window, xpos float64, ypos float64, xdelta float64, ydelta float64) {
	// All palette.gui coordinates are integer, but some glfw platforms may support
	// sub-pixel cursor positions.
	if math.Mod(xpos, 1.0) != 0.0 || math.Mod(ypos, 1.0) != 0.0 {
		log.Printf("Mousepos: we're getting sub-pixel mouse coordinates!\n")
	}
	runcontext.glfwMousePos = image.Point{X: int(xpos), Y: int(ypos)}
}

func (runcontext *RunContext) gotKey(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	log.Printf("key=%d is ignored\n", key)
}

*/
