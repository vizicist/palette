package gui

import (
	"image"
	"log"
	"math"
	"sync"

	// Don't be tempted to use go-gl
	"github.com/fogleman/gg"
	"github.com/goxjs/glfw"
	"github.com/micaelAlastor/nanovgo"
)

// MouseEvent represents mouse data independent of glfw
type MouseEvent struct {
	pos      image.Point
	button   int
	down     bool
	modifier int
}

// Screen xxx
type Screen struct {
	WindowData
	ctx             *nanovgo.Context
	console         *Console
	menubytes       []byte
	menushown       bool
	mouseButtonDown [3]bool

	mouseMutex   sync.Mutex
	mouseEvents  []MouseEvent
	glfwMousePos image.Point
	ggctx        *gg.Context
	goimage      *image.RGBA
	imageHandle  int // nanovgo image handle
}

// NewScreen xxx
func NewScreen(name string, ctx *nanovgo.Context) *Screen {

	style := Style{
		fontSize:    18.0,
		fontFace:    "lucida",
		textColor:   black,
		strokeColor: black,
		fillColor:   white,
	}

	screen := &Screen{
		WindowData: WindowData{
			name:    name,
			style:   style,
			rect:    image.Rectangle{},
			objects: make(map[string]Window),
		},

		ctx:          ctx,
		menubytes:    make([]byte, 0),
		menushown:    false,
		mouseEvents:  make([]MouseEvent, 0),
		glfwMousePos: image.Point{0, 0},
	}

	// Initialize it to contain a Console tool
	// just to test APIs etc
	screen.console = NewConsole("console")
	AddObject(screen.objects, screen.console)

	return screen
}

// Objects xxx
func (screen *Screen) Objects() map[string]Window {
	return screen.objects
}

// Draw xxx
func (screen *Screen) Draw(ctx *nanovgo.Context) {
	for _, o := range screen.objects {
		o.Draw(ctx)
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

	o := ObjectUnder(screen.objects, event.pos)
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

// Resizex xxx
func (screen *Screen) Resizex(newWidth, newHeight int) {

	// This goimage is created once, and only changes if the screen size changes.
	screen.goimage = image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	screen.ggctx = gg.NewContextForRGBA(screen.goimage)

	// This imageHandle is for the OpenGL texture, used by nanovgo.
	screen.imageHandle = screen.ctx.CreateImageFromGoImage(0, screen.goimage)

	// XXX - default font height should be configurable
	screen.style = screen.style.SetFontSizeByHeight(20)
	screen.rect = image.Rect(0, 0, newWidth, newHeight)

	nrect := screen.rect.Inset(10)
	screen.console.Resize(nrect)

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
