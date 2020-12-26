package gui

import (
	"image"
	"log"
	"math"

	// Don't be tempted to use go-gl
	"github.com/goxjs/glfw"
	"github.com/micaelAlastor/nanovgo"
)

// VizScreen xxx
type VizScreen struct {
	VizObjData
	mousePos        image.Point
	mouseButtonDown []bool
	ctx             *nanovgo.Context
	status          *VizStatus
}

// NewScreen xxx
func NewScreen(name string, ctx *nanovgo.Context) *VizScreen {

	style := Style{
		fontSize:    18.0,
		fontFace:    "lucida",
		textColor:   black,
		strokeColor: black,
		fillColor:   white,
	}

	screen := &VizScreen{
		VizObjData: VizObjData{
			name:    name,
			style:   style,
			rect:    image.Rectangle{},
			objects: make(map[string]VizObj),
		},
		mousePos:        image.Point{},
		mouseButtonDown: make([]bool, 3),
		ctx:             ctx,
	}

	// Initialize it to contain a Status tool
	// just to test APIs etc
	screen.status = NewStatus("status")
	AddObject(screen.objects, screen.status)

	return screen
}

// Objects xxx
func (screen *VizScreen) Objects() map[string]VizObj {
	return screen.objects
}

// Draw xxx
func (screen *VizScreen) Draw(ctx *nanovgo.Context) {
	for _, o := range screen.objects {
		o.Draw(ctx)
	}
}

// HandleMouseInput xxx
func (screen *VizScreen) HandleMouseInput(pos image.Point, mdown bool) {
	for _, o := range screen.objects {
		o.HandleMouseInput(pos, mdown)
	}
}

// Resize xxx
func (screen *VizScreen) Resize(rect image.Rectangle) {
	screen.style = screen.style.SetFontSizeByHeight(20)
	screen.rect = rect

	nrect := rect.Inset(10)
	screen.status.Resize(nrect)

	/*
		for nm, o := range screen.objects {
			log.Printf("VizScreen: resizing wind=%s rect=%v\n", nm, rect)
			o.Resize(rect)
		}
	*/
}

// Mousebutton is a callback from glfw
func (screen *VizScreen) Mousebutton(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	if button < 0 || int(button) >= len(screen.mouseButtonDown) {
		log.Printf("mousebutton: unexpected button=%d\n", button)
		return
	}
	down := false
	if action == 1 {
		down = true
	}
	screen.mouseButtonDown[button] = down
	wind := ObjectUnder(screen.objects, screen.mousePos)
	if wind != nil {
		wind.HandleMouseInput(screen.mousePos, down)
	}
}

// Mousepos is a callabck from glfw
func (screen *VizScreen) Mousepos(w *glfw.Window, xpos float64, ypos float64, xdelta float64, ydelta float64) {
	// All palette.gui coordinates are integer, but some glfw platforms may support
	// sub-pixel cursor positions.
	if math.Mod(xpos, 1.0) != 0.0 || math.Mod(ypos, 1.0) != 0.0 {
		log.Printf("Mousepos: we're getting sub-pixel mouse coordinates!\n")
	}
	screen.mousePos = image.Point{X: int(xpos), Y: int(ypos)}
	// log.Printf("Mousepos: %v\n", screen.mousePos)
}
