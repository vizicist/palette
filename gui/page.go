package gui

import (
	"image"
	"log"
	"math"

	"github.com/micaelAlastor/nanovgo"
)

var currentPage string

// VizPage xxx
type VizPage struct {
	style         Style
	ctx           *nanovgo.Context
	origin        image.Point
	width         float32
	height        float32
	objects       map[string]Obj
	localSettings map[string]string
	focused       Obj
	visible       bool
}

// NewPage xxx
func NewPage(ctx *nanovgo.Context, width, height float32) *VizPage {
	// Use lineHeight and charWidth to make it
	// easier to do placement relative to window size
	pg := &VizPage{
		ctx:           ctx,
		width:         width,
		height:        height,
		objects:       make(map[string]Obj),
		localSettings: make(map[string]string),
	}
	pg.style = pg.defaultStyle()
	// default fontSize is 18.  Scale to real size of window.
	pg.style.fontSize = 18.0 * float32(math.Min(float64(width/600.0), float64(height/800.0)))
	return pg
}

// Do xxx
func (pg *VizPage) Do() {
	pg.Draw()
}

// Draw xxx
func (pg *VizPage) Draw() {

	pg.ctx.Save()
	defer pg.ctx.Restore()

	pg.ctx.SetTextLetterSpacing(0)

	for _, o := range pg.objects {
		switch {
		case pg.focused != nil && o == pg.focused:
			// if ButtonDown[0] {
			// 	log.Printf("Calling FOCUSED handleInput mdown=true of %s\n", o.Name())
			// }
			o.HandleInput(MouseX, MouseY, ButtonDown[0])
		case pg.focused == nil:
			// if ButtonDown[0] {
			// 	log.Printf("Calling unfocused handleInput mdown=true of %s\n", o.Name())
			// }
			o.HandleInput(MouseX, MouseY, ButtonDown[0])
		}
	}
	var focusobj Obj
	for _, o := range pg.objects {
		if o == pg.focused {
			focusobj = o
		} else {
			o.Draw(pg.ctx)
		}
	}
	if focusobj != nil {
		focusobj.Draw(pg.ctx)
	}
}

// AddObject xxx
func (pg *VizPage) AddObject(o Obj) {
	name := o.Name()
	_, ok := pg.objects[name]
	if ok {
		log.Printf("There's already an object named %s on that page\n", name)
	} else {
		pg.objects[name] = o
	}
}

// SetFocus xxx
func (pg *VizPage) SetFocus(o Obj) {
	pg.focused = o
}

// Focus xxx
func (pg *VizPage) Focus() Obj {
	return pg.focused
}
