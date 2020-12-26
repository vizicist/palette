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
}

// NewScreen xxx
func NewScreen(ctx *nanovgo.Context) *VizScreen {

	s := Style{
		fontSize:    18.0,
		fontFace:    "lucida",
		textColor:   black,
		strokeColor: black,
		fillColor:   white,
	}

	return &VizScreen{
		VizObjData: VizObjData{
			parent:  nil,
			style:   s,
			rect:    image.Rectangle{},
			objects: make(map[string]VizObj),
		},
		mousePos:        image.Point{},
		mouseButtonDown: make([]bool, 3),
		ctx:             ctx,
	}
}

// Objects xxx
func (screen *VizScreen) Objects() map[string]VizObj {
	return screen.objects
}

// Style xxx
func (screen *VizScreen) Style() Style {
	return screen.style
}

// Draw xxx
func (screen *VizScreen) Draw(ctx *nanovgo.Context) {
	for _, o := range screen.objects {
		o.Draw(ctx)
	}
}

// HandleMouseInput xxx
func (screen *VizScreen) HandleMouseInput(pos image.Point, mdown bool) {
	for nm, o := range screen.objects {
		log.Printf("HEY: Should be resizing wind=%s\n", nm)
		o.HandleMouseInput(pos, mdown)
	}
}

// SetSize xxx
func (style Style) SetSize(rect image.Rectangle) Style {
	w := rect.Max.X
	h := rect.Max.Y
	if w <= 0 || h <= 0 {
		log.Printf("Style.Resize: bad dimensions? %d,%d\n", w, h)
		return style
	}
	style.lineHeight = h / 48.0
	style.charWidth = w / 80.0
	return style
}

// Resize xxx
func (screen *VizScreen) Resize(rect image.Rectangle) {
	screen.style = screen.style.SetSize(rect)
	screen.rect = rect
	for nm, o := range screen.objects {
		log.Printf("HEY: Should be resizing wind=%s\n", nm)
		o.Resize(rect)
	}
}

/*
// AddWind xxx
func (screen *VizScreen) AddWind(name string, rect image.Rectangle) (*VizObjData, error) {
	// XXX - lock?
	objs := screen.Objects()
	_, ok := objs[name]
	if ok {
		err := fmt.Errorf("AddWind: there's already a VizObjData with the name: %s", name)
		return nil, err
	}
	log.Printf("AddWind %s %+v\n", name, rect)
	// Use lineHeight and charWidth to make it
	// easier to do placement relative to window size
	w := screen.top.NewSubWind(rect)
	/ *
		w := &VizObjData{
			Parent:        nil,
			Style:         Style{},
			ctx:           screen.ctx,
			rect:          rect,
			Objects:       make(map[string]Obj),
		}
	* /
	screen.top.Objects()[name] = w
	w.Style = w.defaultStyle()
	// w := float32(wind.Width())
	// h := float32(wind.Height())
	// wind.style.fontSize = 18.0 * float32(math.Min(float64(w/600.0), float64(h/800.0)))
	// default fontSize is 18.  should it be scaled here?
	w.Style.fontSize = 18.0
	return w, nil
}
*/

// WindowUnder xxx
func (screen *VizScreen) WindowUnder(pos image.Point) VizObj {
	for _, o := range screen.objects {
		if pos.In(o.Rect()) {
			return o
		}
	}
	return nil
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
	wind := screen.WindowUnder(screen.mousePos)
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
	log.Printf("Mousepos: %v\n", screen.mousePos)
}
