package gui

import (
	"strings"

	"github.com/micaelAlastor/nanovgo"
)

// VizComboCallback xxx
type VizComboCallback func(c *VizCombo, choice int)

// VizCombo xxx
type VizCombo struct {
	name                  string
	label                 string
	wasPressed            bool
	isPopped              bool
	style                 Style
	x, y                  float32
	labelw, valuew, h     float32
	col                   nanovgo.Color
	choices               []string
	choice                int
	callback              VizComboCallback
	waitingForUp          bool
	recomputeChoicesWidth bool
	choicesWidth          float32
}

// NewCombo xxx
func NewCombo(name, label string, x, y, labelw, valuew, h float32, style Style, cb VizComboCallback) *VizCombo {
	if !strings.HasPrefix(name, "combo.") {
		name = "combo." + name
	}
	return &VizCombo{
		name:       name,
		label:      label,
		style:      style,
		x:          x,
		y:          y,
		labelw:     labelw,
		valuew:     valuew,
		h:          h,
		choices:    make([]string, 0),
		choice:     0,
		wasPressed: false,
		callback:   cb,
	}
}

// Name xxx
func (c *VizCombo) Name() string {
	return c.name
}

func (c *VizCombo) getMouseLocation(mx, my float32) (invalue bool, inpopup bool, choice int) {

	valuex0 := c.x + c.labelw
	valuex1 := c.x + c.labelw + c.valuew
	invalue = mx >= valuex0 && mx <= valuex1 && my >= c.y && my <= (c.y+c.h)

	lh := c.style.lineHeight
	y0 := c.y + lh
	y1 := c.y + lh*float32(1+len(c.choices))
	inpopup = mx >= valuex0 && mx <= valuex1 && my >= y0 && my <= y1

	if !inpopup {
		choice = -1
	} else {
		choice = int((my - y0) / lh)
	}
	return invalue, inpopup, choice
}

// HandleInput xxx
func (c *VizCombo) HandleInput(mx, my float32, mdown bool) {
	if c.isPopped {
		c.handleWhenPopped(mx, my, mdown)
	} else {
		c.handleWhenUnpopped(mx, my, mdown)
	}
}

func (c *VizCombo) handleWhenUnpopped(mx, my float32, mdown bool) {
	switch {
	case mdown == true:
		// inside the combo line?
		invalue, _, _ := c.getMouseLocation(mx, my)
		if invalue {
			if c.wasPressed == false {
				c.wasPressed = true
				Page[CurrentPageName].SetFocus(c)
				c.isPopped = true
			}
		}
	case mdown == false:
		c.wasPressed = false
	}
}

func (c *VizCombo) handleWhenPopped(mx, my float32, mdown bool) {
	switch {
	case mdown == true:
		// inside the combo line?
		if c.wasPressed == false {
			// invalue, _, _ := c.getMouseLocation(mx, my)
			// fmt.Printf("Mouse down, invalue=%v\n", invalue)
			c.wasPressed = true
		}
	case mdown == false:
		switch {
		case c.wasPressed == true:
			// The mouse has just been let up
			invalue, inpopup, choice := c.getMouseLocation(mx, my)
			switch {
			case invalue == true:
				// Do nothing, stay popped, though perhaps we should unpop
			case inpopup == true:
				// mouse is inside popup, choice=%d\n", choice)
				if c.callback != nil {
					c.callback(c, choice)
				}
				c.isPopped = false
				Page[CurrentPageName].SetFocus(nil)
			default:
				c.isPopped = false
				Page[CurrentPageName].SetFocus(nil)
			}
			c.wasPressed = false
		}
	}
}

// Draw xx
func (c *VizCombo) Draw(ctx *nanovgo.Context) {
	ctx.SetTextAlign(nanovgo.AlignLeft | nanovgo.AlignTop)
	c.style.Do(ctx)
	y := c.y
	var choice string
	if c.choice < 0 || c.choice >= len(c.choices) {
		choice = "NO CHOICE?"
	} else {
		choice = c.choices[c.choice]
	}

	// labelwidth, _ := ctx.TextBounds(0, 0, c.label)

	// ctx.Text uses FillColor for the text
	ctx.SetFillColor(c.style.textColor)
	ctx.Text(c.x, y, c.label)
	ctx.Text(c.x+c.labelw, y, choice)

	if c.recomputeChoicesWidth {
		maxcx := float32(0)
		for _, s := range c.choices {
			if cx, _ := ctx.TextBounds(0, 0, s); cx > maxcx {
				maxcx = cx
			}
		}
		c.choicesWidth = maxcx
		c.recomputeChoicesWidth = false
	}

	drawIcon(ctx, IconDOWN, c.x+c.labelw+c.valuew, y+0.5*c.style.lineHeight)

	if c.isPopped {
		ctx.Save()
		ctx.BeginPath()
		ctx.Rect(c.x+c.labelw, c.y+c.style.lineHeight,
			float32(c.valuew), float32(len(c.choices))*c.style.lineHeight)
		ctx.SetFillColor(nanovgo.RGBA(255, 255, 225, 255))
		ctx.Fill()
		ctx.Restore()
		// Draw the popped-up list of choices
		y = c.y + c.style.lineHeight
		for _, choice := range c.choices {
			ctx.Text(c.x+c.labelw, y, choice)
			y += c.style.lineHeight
		}
	}
}

func drawIcon(ctx *nanovgo.Context, icon int, x, y float32) {
	ctx.Save()
	ctx.SetFontFace("icons")
	ctx.SetTextAlign(nanovgo.AlignLeft | nanovgo.AlignMiddle)
	ctx.Text(x, y, cpToUTF8(icon))
	ctx.Restore()
}

func cpToUTF8(cp int) string {
	return string([]rune{rune(cp)})
}

func (c *VizCombo) addValue(s string) {
	c.choices = append(c.choices, s)
	c.recomputeChoicesWidth = true
}
