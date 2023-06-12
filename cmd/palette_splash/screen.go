package main

/*
 This is the only file that should contain references to ebiten.
 I.e. it manages everything about the screen and mouse events.
*/
import (
	"fmt"
	"image"
	"image/color"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	// "github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/vizicist/palette/engine"
	"golang.org/x/image/font"
)

// Screen satisfies the ebiten.Game interface.
// Screen contains the pageWindow.
// Screen and Style should be the only things calling ebiten.
type Screen struct {
	ctx         WinContext
	currentPage Window
	styleName   string
	// rectx       image.Rectangle
	eimage    *ebiten.Image
	time0     time.Time
	lastprint time.Time
	cursorPos image.Point
	foreColor color.RGBA
	backColor color.RGBA
}

// CurrentPage xxx
var CurrentPage *Page
var CurrentWorld *Window

var Styles map[string]*StyleInfo

// Run runs the Gui and never returns
func Run() {

	// The user of palette.win can/should add custom tool types,
	// but we want to make sure that the standard ones are ours.
	// AddToolType("Menu", NewMenu)
	RegisterDefaultTools()

	minSize := image.Point{X: 640, Y: 480} // default

	// change it with config value
	winsize, err := engine.GetParam("engine.winsize")
	engine.LogIfError(err)
	if winsize != "" {
		var xsize int
		var ysize int
		n, err := fmt.Sscanf(winsize, "%d,%d", &xsize, &ysize)
		if err != nil || n != 2 {
			engine.LogWarn("Run: bad format of winsize", "winsize", winsize)
		} else {
			minSize.X = xsize
			minSize.Y = ysize
		}
	}

	Styles = make(map[string]*StyleInfo)
	Styles["fixed"] = NewStyle("fixed", 16)
	Styles["regular"] = NewStyle("regular", 16)

	ebiten.SetWindowSize(minSize.X, minSize.Y)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Palette Window System")

	// The Screen is the only Window whose parent is nil
	screenCtx := newWindowContextNoParent()
	eimage := ebiten.NewImage(minSize.X, minSize.Y)
	screen := &Screen{
		ctx:       screenCtx,
		eimage:    eimage,
		time0:     time.Now(),
		lastprint: time.Now(),
		foreColor: ForeColor,
		backColor: BackColor,
		styleName: "fixed",
	}
	WinSetSize(screen, minSize)

	td := NewPage(screen, "home")
	screen.currentPage = WinAddChild(screen, td)

	// START DEBUG - do RESTORE
	fname := engine.ConfigFilePath("homepage.json")
	bytes, err := os.ReadFile(fname)
	if err != nil {
		engine.LogIfError(err)
	} else {
		page := td.w.(*Page)
		err = page.restoreState(string(bytes))
		if err != nil {
			engine.LogIfError(err)
		}
	}

	// This is it!  RunGame runs forever
	if err := ebiten.RunGame(screen); err != nil {
		engine.LogIfError(err)
		// Should be fatal?
	}
}

// Context xxx
func (screen *Screen) Context() *WinContext {
	return &screen.ctx
}

// Do xxx
func (screen *Screen) Do(cmd engine.Cmd) string {

	switch cmd.Subj {
	case "drawline":
		xy0 := cmd.ValuesXY0(engine.PointZero)
		xy1 := cmd.ValuesXY1(engine.PointZero)
		screen.drawLine(xy0, xy1)
	case "drawrect":
		rect := cmd.ValuesRect(image.Rectangle{})
		screen.drawRect(rect)
	case "drawfilledrect":
		rect := cmd.ValuesRect(image.Rectangle{})
		screen.drawFilledRect(rect)
	case "drawtext":
		text := cmd.ValuesString("text", "")
		styleName := cmd.ValuesString("style", "")
		pos := cmd.ValuesXY("pos", engine.PointZero)
		screen.drawText(text, styleName, pos)
	case "setcolor":
		c := cmd.ValuesColor(RedColor)
		screen.foreColor = c
	case "showmousecursor":
		show := cmd.ValuesBool("show", true)
		if show {
			ebiten.SetCursorMode(ebiten.CursorModeVisible)
		} else {
			ebiten.SetCursorMode(ebiten.CursorModeHidden)
		}
	case "closeme":
		engine.LogWarn("screen.runMsgs: should not be getting CloseMeMsg!?")
	case "resizeme":
		size := cmd.ValuesSize(engine.PointZero)
		ebiten.SetWindowSize(size.X, size.Y)

	default:
		engine.LogWarn("screen.runMsgs: unrecognized", "msg", cmd.Subj)
	}
	return ""
}

// Layout satisfies the ebiten.Game interface
func (screen *Screen) Layout(width, height int) (int, int) {
	if screen == nil {
		engine.LogWarn("Screen.Layout: Hey, screen shouldn't be nil!")
		return width, height
	}
	currSize := WinGetSize(screen)
	if currSize.X != width || currSize.Y != height {
		newSize := image.Point{width, height}
		WinSetSize(screen, newSize)
		WinSetChildSize(screen.currentPage, newSize)
	}
	return width, height
}

// Update satisfies the ebiten.Game interface
func (screen *Screen) Update() (err error) {

	x, y := ebiten.CursorPosition()
	newPos := image.Point{x, y}

	// Ignore updates outside the screen
	size := WinGetSize(screen)
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
			screen.currentPage.Do(NewMouseCmd("down", newPos, n))

		case inpututil.IsMouseButtonJustReleased(eb):
			screen.currentPage.Do(NewMouseCmd("up", newPos, n))

		default:
			// Drag events only happen when position changes
			if newPos.X != screen.cursorPos.X || newPos.Y != screen.cursorPos.Y {
				screen.cursorPos = newPos
				screen.currentPage.Do(NewMouseCmd("drag", newPos, n))
			}
		}
	}
	return nil
}

// Draw satisfies the ebiten.Game interface
func (screen *Screen) Draw(eimage *ebiten.Image) {
	screen.eimage = eimage
	screen.currentPage.Do(engine.NewSimpleCmd("redraw"))
}

// drawRect xxx
func (screen *Screen) drawRect(rect image.Rectangle) {
	x0 := rect.Min.X
	y0 := rect.Min.Y
	x1 := rect.Max.X
	y1 := rect.Max.Y
	engine.LogOfType("drawing", "drawRect", "x0", x0, "y0", y0, "x1", x1, "y1", y1)
	screen.drawLine(image.Point{x0, y0}, image.Point{x1, y0})
	screen.drawLine(image.Point{x1, y0}, image.Point{x1, y1})
	screen.drawLine(image.Point{x1, y1}, image.Point{x0, y1})
	screen.drawLine(image.Point{x0, y1}, image.Point{x0, y0})
}

// drawLine xxx
func (screen *Screen) drawLine(xy0, xy1 image.Point) {
	engine.LogOfType("drawing", "drawLine", "x0", xy0.X, "y0", xy0.Y, "x1", xy1.X, "y1", xy1.Y, "color", screen.foreColor)
	vector.StrokeLine(screen.eimage,
		float32(xy0.X), float32(xy0.Y), float32(xy1.X), float32(xy1.Y), 1, screen.foreColor, true)
}

func (screen *Screen) drawText(s string, styleName string, pos image.Point) {
	styleInfo := Styles[styleName]
	engine.LogOfType("drawing", "drawText", "s", s, "x", pos.X, "y", pos.Y)
	text.Draw(screen.eimage, s, styleInfo.fontFace, pos.X, pos.Y, screen.foreColor)
}

func (screen *Screen) drawFilledRect(rect image.Rectangle) {
	w := rect.Max.X - rect.Min.X
	h := rect.Max.Y - rect.Min.Y
	engine.LogOfType("drawing", "drawFilledRect", "x0", rect.Min.X)
	vector.DrawFilledRect(screen.eimage, float32(rect.Min.X), float32(rect.Min.Y), float32(w), float32(h), screen.foreColor, true)
}

func TextBoundString(face font.Face, s string) image.Rectangle {
	return text.BoundString(face, s)
}
