package gui

import (
	"fmt"
	"image"
	"log"
	"math"
	"sync"

	// Don't be tempted to use go-gl
	"github.com/fogleman/gg"
	"github.com/goxjs/glfw"
	"github.com/micaelAlastor/nanovgo"
	"github.com/vizicist/palette/engine"
)

// MouseEvent represents mouse data independent of glfw
type MouseEvent struct {
	pos      image.Point
	button   int
	down     bool
	modifier int
}

// RootWindow xxx
type RootWindow struct {
	WindowData
	console *Console
}

// NewRootWindow xxx
func NewRootWindow(name string) (*RootWindow, error) {
	w := &RootWindow{
		WindowData: WindowData{
			name:    name,
			style:   Style{},
			rect:    image.Rectangle{},
			objects: make(map[string]Window),
		},
		console: nil,
	}

	return w, nil
}

// Screen xxx
type Screen struct {
	rootWindow *RootWindow

	rect            image.Rectangle
	menubytes       []byte
	menushown       bool
	mouseButtonDown [3]bool
	// style           Style

	mouseMutex   sync.Mutex
	mouseEvents  []MouseEvent
	glfwMousePos image.Point
	ggctx        *gg.Context
	image        *image.RGBA
	imageHandle  int // nanovgo image handle

	nanoctx *nanovgo.Context
}

// NewScreen xxx
func NewScreen(glfwWindow *glfw.Window) (*Screen, error) {

	style := DefaultStyle

	ctx, err := nanovgo.NewContext(0 /*nanovgo.AntiAlias | nanovgo.StencilStrokes | nanovgo.Debug*/)
	if err != nil {
		return nil, fmt.Errorf("NewScreen: Unable to create nanovgo.NewContext, err=%s", err)
	}

	root, err := NewRootWindow("root")
	if err != nil {
		return nil, err
	}
	root.style = style

	screen := &Screen{
		rootWindow: root,
		rect:       image.Rectangle{},
		// objects:    make(map[string]Window),

		// style:        style,
		nanoctx:      ctx,
		menubytes:    make([]byte, 0),
		menushown:    false,
		mouseEvents:  make([]MouseEvent, 0),
		glfwMousePos: image.Point{0, 0},
	}

	err = LoadFonts(screen.nanoctx)
	if err != nil {
		return nil, err
	}

	glfwWindow.SetKeyCallback(key)
	glfwWindow.SetMouseButtonCallback(screen.callbackForMousebutton)
	glfwWindow.SetMouseMovementCallback(screen.callbackForMousepos)

	return screen, nil
}

/*
// Objects xxx
func (screen *Screen) Objects() map[string]Window {
	return screen.objects
}
*/

// Draw xxx
func (screen *Screen) Draw() {

	screen.ggctx.Push()
	screen.ggctx.SetLineWidth(3.0)
	screen.ggctx.SetRGBA(0, 0, 1.0, 1.0)
	screen.ggctx.DrawLine(0, 0, 500, 500)
	screen.ggctx.Stroke()
	screen.ggctx.Pop()

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

	screen.rootWindow.style.Do(screen.nanoctx)

	for _, o := range screen.rootWindow.objects {
		o.Draw(screen.nanoctx)
	}
}

// CheckMouseInput xxx
func (screen *Screen) CheckMouseInput() {

	screen.mouseMutex.Lock()
	defer screen.mouseMutex.Unlock()

	if len(screen.mouseEvents) == 0 {
		return
	}

	event := screen.mouseEvents[0]
	screen.mouseEvents = append(screen.mouseEvents[1:])
	// log.Printf("Screen.HandleMouseInput: popped %+v", event)

	screen.mouseButtonDown[int(event.button)] = event.down

	o := ObjectUnder(screen.rootWindow.objects, event.pos)
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

// BuildScreen xxx
func (screen *Screen) BuildScreen(newWidth, newHeight int) {

	// This goimage is created once, and only changes if the screen size changes.
	screen.image = image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	screen.ggctx = gg.NewContextForRGBA(screen.image)

	// This imageHandle is for the OpenGL texture, used by nanovgo.
	screen.imageHandle = screen.nanoctx.CreateImageFromGoImage(0, screen.image)

	// XXX - default font height should be configurable
	// screen.style = screen.style.SetFontSizeByHeight(20)
	screen.rect = image.Rect(0, 0, newWidth, newHeight)

	nrect := screen.rect.Inset(10)

	// XXX - should I avoid creating a
	if screen.rootWindow.console == nil {
		screen.rootWindow.console = NewConsole("root")
		AddObject(screen.rootWindow.objects, screen.rootWindow.console)
	}

	screen.rootWindow.console.Resize(nrect)

	/*
		for nm, o := range screen.objects {
			log.Printf("Screen: resizing wind=%s rect=%v\n", nm, rect)
			o.Resize(rect)
		}
	*/
}

// callbackForMousebutton is a callback from glfw
// Since it's in a glfw callback, just queue up an event that gets handled
// in the normal screen refresh.
func (screen *Screen) callbackForMousebutton(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	if button < 0 || int(button) >= len(screen.mouseButtonDown) {
		log.Printf("mousebutton: unexpected value button=%d\n", button)
		return
	}
	down := false
	if action == 1 {
		down = true
	}
	screen.mouseMutex.Lock()
	screen.mouseEvents = append(screen.mouseEvents,
		MouseEvent{
			pos:      screen.glfwMousePos,
			button:   int(button),
			down:     down,
			modifier: int(mods),
		},
	)
	screen.mouseMutex.Unlock()
}

// callbackForMousepos is a callabck from glfw
func (screen *Screen) callbackForMousepos(w *glfw.Window, xpos float64, ypos float64, xdelta float64, ydelta float64) {
	// All palette.gui coordinates are integer, but some glfw platforms may support
	// sub-pixel cursor positions.
	if math.Mod(xpos, 1.0) != 0.0 || math.Mod(ypos, 1.0) != 0.0 {
		log.Printf("Mousepos: we're getting sub-pixel mouse coordinates!\n")
	}
	screen.glfwMousePos = image.Point{X: int(xpos), Y: int(ypos)}
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

func key(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	// log.Printf("key=%d is ignored\n", key)
	/*
		if key == glfw.KeyEscape && action == glfw.Press {
			w.SetShouldClose(true)
		}
	*/
}
