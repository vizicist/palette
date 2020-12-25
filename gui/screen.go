package gui

import (
	"fmt"
	"image"
	"log"
	"math"

	// Don't be tempted to use go-gl
	"github.com/goxjs/glfw"
	"github.com/micaelAlastor/nanovgo"
)

// VizScreen xxx
type VizScreen struct {
	width           int
	height          int
	mouseX          int
	mouseY          int
	ctx             *nanovgo.Context
	wind            map[string]*VizWind
	mouseButtonDown []bool
}

// NewScreen xxx
func NewScreen(ctx *nanovgo.Context) *VizScreen {
	return &VizScreen{
		width:           0,
		height:          0,
		mouseButtonDown: make([]bool, 3),
		ctx:             ctx,
		wind:            make(map[string]*VizWind),
	}
}

// Do xxx
func (screen *VizScreen) Do() {
	for _, w := range screen.wind {
		// log.Printf("VizScreen.Do: wind=%s\n", nm)
		w.Do()
	}
}

// Resize xxx
func (screen *VizScreen) Resize(w, h int) {
	if w <= 0 || h <= 0 {
		log.Printf("VizScreen.Resize: bad dimensions? %d,%d\n", w, h)
		return
	}
	screen.width = w
	screen.height = h
	log.Printf("VizScreen.Resize = %d,%d\n", w, h)
	for nm := range screen.wind {
		log.Printf("HEY: Should be resizing wind=%s\n", nm)
	}
}

// AddWind xxx
func (screen *VizScreen) AddWind(name string, rect image.Rectangle) (*VizWind, error) {
	// XXX - lock?
	_, ok := screen.wind[name]
	if ok {
		err := fmt.Errorf("AddWind: there's already a VizWind with the name: %s", name)
		return nil, err
	}
	log.Printf("AddWind %s %+v\n", name, rect)
	// Use lineHeight and charWidth to make it
	// easier to do placement relative to window size
	w := &VizWind{
		style:         Style{},
		ctx:           screen.ctx,
		rect:          rect,
		objects:       make(map[string]Obj),
		localSettings: make(map[string]string),
		focused:       nil,
		visible:       false,
	}
	screen.wind[name] = w
	w.style = w.defaultStyle()
	// w := float32(wind.Width())
	// h := float32(wind.Height())
	// wind.style.fontSize = 18.0 * float32(math.Min(float64(w/600.0), float64(h/800.0)))
	// default fontSize is 18.  should it be scaled here?
	w.style.fontSize = 18.0
	return w, nil
}

// Mousebutton xxx
func (screen *VizScreen) Mousebutton(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	if button < 0 || int(button) >= len(screen.mouseButtonDown) {
		log.Printf("mousebutton: unexpected button=%d\n", button)
		return
	}
	if action == 1 {
		screen.mouseButtonDown[button] = true
	} else {
		screen.mouseButtonDown[button] = false
	}
	log.Printf("Mousebutton %d %d\n", button, action)
}

// Mousepos xxx
func (screen *VizScreen) Mousepos(w *glfw.Window, xpos float64, ypos float64, xdelta float64, ydelta float64) {
	// All palette.gui coordinates are integer, but some glfw platforms may support
	// sub-pixel cursor positions.
	if math.Mod(xpos, 1.0) != 0.0 || math.Mod(ypos, 1.0) != 0.0 {
		log.Printf("Mousepos: we're getting sub-pixel mouse coordinates!\n")
	}
	screen.mouseX = int(xpos)
	screen.mouseY = int(ypos)
	log.Printf("Mousepos: %d %d\n", screen.mouseX, screen.mouseY)
}
