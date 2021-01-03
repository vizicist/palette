package gui

import (
	"fmt"
	"image"
	"log"
	"sync"

	// Don't be tempted to use go-gl
	"github.com/fogleman/gg"
	"github.com/micaelAlastor/nanovgo"
	"github.com/vizicist/palette/engine"
)

// MouseEvent represents mouse data independent of how it's received
type MouseEvent struct {
	pos      image.Point
	button   int
	down     bool
	modifier int
}

// Screen contains the rootWindow, with interfaces to the real display
type Screen struct {
	root *Root

	rect            image.Rectangle
	menubytes       []byte
	menushown       bool
	mouseButtonDown [3]bool
	mouseMutex      sync.Mutex
	mouseEvents     []MouseEvent

	ggctx *gg.Context
	image *image.RGBA

	nanoctx     *nanovgo.Context
	imageHandle int // nanovgo image handle
}

// NewScreen xxx
func NewScreen(width, height int) (*Screen, error) {

	ctx, err := nanovgo.NewContext(0 /*nanovgo.AntiAlias | nanovgo.StencilStrokes | nanovgo.Debug*/)
	if err != nil {
		return nil, fmt.Errorf("NewScreen: Unable to create nanovgo.NewContext, err=%s", err)
	}

	screen := &Screen{
		root:        nil,
		rect:        image.Rectangle{},
		nanoctx:     ctx,
		menubytes:   make([]byte, 0),
		menushown:   false,
		mouseEvents: make([]MouseEvent, 0),
	}

	err = LoadFonts(screen.nanoctx)
	if err != nil {
		return nil, err
	}

	// Add initial contents of RootWindow
	root, err := NewRoot(NewStyle("regular", 32))
	if err != nil {
		return nil, err
	}
	screen.root = root

	root.console = NewConsole(NewStyle("mono", 32))
	AddObject(root.objects, "console1", root.console)

	root.console2 = NewConsole(NewStyle("mono", 32))
	AddObject(root.objects, "console2", root.console2)

	screen.Resize(width, height)

	return screen, nil
}

// Resize xxx
func (screen *Screen) Resize(newWidth, newHeight int) {

	screen.rect = image.Rect(0, 0, newWidth, newHeight)

	// This goimage is created once, and only changes if the screen size changes.
	screen.image = image.NewRGBA(screen.rect)
	screen.ggctx = gg.NewContextForRGBA(screen.image)

	screen.imageHandle = screen.nanoctx.CreateImageFromGoImage(0, screen.image)

	nrect := screen.rect.Inset(10)

	screen.root.Resize(nrect)

}

// DoOneFrame xxx
func (screen *Screen) DoOneFrame(fbWidth, fbHeight int) {

	pixelRatio := float32(fbWidth) / float32(screen.rect.Dx())

	screen.nanoctx.BeginFrame(screen.rect.Dx(), screen.rect.Dy(), pixelRatio)

	screen.CheckMouseInput()

	screen.Draw()

	screen.nanoctx.EndFrame()
}

// Draw xxx
func (screen *Screen) Draw() {

	// At some point, use of nanovgo should go away,
	// since the only thing it's used for is blasting
	// the screen.image to a display.

	screen.nanoctx.UpdateImage(screen.imageHandle, screen.image.Pix)

	w := float32(screen.rect.Dx())
	h := float32(screen.rect.Dy())
	img := nanovgo.ImagePattern(0, 0, w, h, 0.0, screen.imageHandle, 1.0)
	screen.nanoctx.Save()
	screen.nanoctx.BeginPath()
	screen.nanoctx.Rect(0, 0, w, h)
	screen.nanoctx.SetFillPaint(img)
	screen.nanoctx.Fill()
	screen.nanoctx.Stroke()
	screen.nanoctx.Restore()

	for _, o := range screen.root.objects {
		screen.ggctx.Push()
		o.Draw(screen.ggctx)
		screen.ggctx.Pop()
	}
}

// CheckMouseInput xxx
func (screen *Screen) CheckMouseInput() {

	if len(screen.mouseEvents) == 0 {
		return
	}
	screen.mouseMutex.Lock()
	defer screen.mouseMutex.Unlock()

	// pop the first event off
	event := screen.mouseEvents[0]
	screen.mouseEvents = append(screen.mouseEvents[1:])

	screen.mouseButtonDown[int(event.button)] = event.down

	o := ObjectUnder(screen.root.objects, event.pos)
	handled := false
	if o != nil {
		handled = o.HandleMouseInput(event.pos, int(event.button), event.down)
	}
	if !handled {
		log.Printf("Screen: pos=%v down=%v outside tool\n", event.pos, event.down)
		if event.down {
			switch screen.menushown {
			case false:
				needed := screen.rect.Dx() * screen.rect.Dy()
				newbytes := make([]byte, len(screen.menubytes), needed)
				copy(screen.menubytes, newbytes)
				screen.menushown = true
				log.Printf("down, menushown, Should be doing ReadPixels here\n")
			case true:
				screen.menushown = false
				log.Printf("down again, menushown, Should be rewriting pixels\n")
			}
		}
	}
}

// LoadFonts xxx
func LoadFonts(ctx *nanovgo.Context) error {
	fonts := map[string]string{
		"icons":  "entypo.ttf",
		"lucida": "lucon.ttf",
	}
	hadErr := false
	for name, filename := range fonts {
		path := engine.ConfigFilePath(filename)
		f := ctx.CreateFont(name, path)
		if f == -1 {
			log.Printf("LoadFonts: could not add font name=%s path=%s", name, path)
			hadErr = true
		}
	}
	if hadErr {
		return fmt.Errorf("LoadFonts: unable to load all fonts")
	}
	return nil
}

// QueueMouseEvent xxx
func (screen *Screen) QueueMouseEvent(event MouseEvent) {
	screen.mouseMutex.Lock()
	screen.mouseEvents = append(screen.mouseEvents, event)
	screen.mouseMutex.Unlock()

}
