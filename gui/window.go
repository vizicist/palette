package gui

import (
	"image"
	"log"
	"strings"

	"github.com/micaelAlastor/nanovgo"
)

// VizWind xxx
type VizWind struct {
	Style         Style
	ctx           *nanovgo.Context
	rect          image.Rectangle
	Objects       map[string]Obj
	localSettings map[string]string
	focused       Obj
	Visible       bool
}

// Rect xxx
func (wind *VizWind) Rect() image.Rectangle {
	return wind.rect
}

// Do xxx
func (wind *VizWind) Do() {
	wind.Draw()
}

// HandleMouseInput xxx
func (wind *VizWind) HandleMouseInput(pos image.Point, down bool) {
	// log.Printf("VizWind.HandleMouseInput: pos=%+v\n", pos)
	for _, o := range wind.Objects {
		if pos.In(o.Rect()) {
			o.HandleMouseInput(pos, down)
		}

	}
}

// NewSubWind xxx
func (wind *VizWind) NewSubWind(r image.Rectangle) *VizWind {
	return &VizWind{
		Style:         wind.Style,
		ctx:           wind.ctx,
		rect:          r,
		Objects:       make(map[string]Obj),
		localSettings: make(map[string]string),
		focused:       nil,
		Visible:       false,
	}
}

// NewButton xxx
func (wind *VizWind) NewButton(name string, text string, pos image.Point, style Style, cb VizButtonCallback) *VizButton {
	if !strings.HasPrefix(name, "button.") {
		name = "button." + name
	}
	w := len(text) * wind.Style.charWidth
	h := wind.Style.lineHeight
	r := image.Rect(pos.X, pos.Y, pos.X+w, pos.Y+h)
	bwind := wind.NewSubWind(r)
	return &VizButton{
		VizWind:   *bwind,
		name:      name,
		isPressed: false,
		text:      text,
		style:     wind.Style,
		callback:  cb,
	}
}

// Draw xxx
func (wind *VizWind) Draw() {

	wind.ctx.Save()
	defer wind.ctx.Restore()

	wind.ctx.Save()
	wind.ctx.BeginPath()
	r := wind.Rect()
	x := r.Min.X
	y := r.Min.Y
	w := r.Dx()
	h := r.Dy()
	cornerRadius := float32(4.0)
	wind.ctx.RoundedRect(float32(x), float32(y), float32(w), float32(h), cornerRadius-0.5)
	wind.ctx.SetStrokeWidth(3.0)
	wind.ctx.Stroke()
	wind.ctx.Restore()

	wind.ctx.SetTextLetterSpacing(0)

	/*
		for _, o := range wind.objects {
			switch {
			case wind.focused != nil && o == wind.focused:
				// if ButtonDown[0] {
				// 	log.Printf("Calling FOCUSED HandleMouseInput mdown=true of %s\n", o.Name())
				// }
				o.HandleMouseInput(MouseX, MouseY, ButtonDown[0])
			case wind.focused == nil:
				// if ButtonDown[0] {
				// 	log.Printf("Calling unfocused HandleMouseInput mdown=true of %s\n", o.Name())
				// }
				o.HandleMouseInput(MouseX, MouseY, ButtonDown[0])
			}
		}
	*/
	for _, o := range wind.Objects {
		o.Draw(wind.ctx)
	}
}

// AddObject xxx
func (wind *VizWind) AddObject(o Obj) {
	name := o.Name()
	_, ok := wind.Objects[name]
	if ok {
		log.Printf("There's already an object named %s in that VizWind\n", name)
	} else {
		wind.Objects[name] = o
	}
}

// SetFocus xxx
func (wind *VizWind) SetFocus(o Obj) {
	wind.focused = o
}

// Focus xxx
func (wind *VizWind) Focus() Obj {
	return wind.focused
}
