package gui

import (
	"log"

	// Don't be tempted to use go-gl
	"github.com/goxjs/glfw"
)

func init() {
	ButtonDown = make([]bool, 3)
}

func letterWidth() float32 {
	return 12.0
}

/*
func drawRunningValue(ctx *nanovgo.Context, label, exe string, x, y float32) {
	drawLabel(ctx, label, x, y)
	status := "NOT RUNNING"
	if util.IsRunning(exe) {
		status = "RUNNING"
	}
	x += 8 * letterWidth()
	drawLabel(ctx, status, x, y)
}

func renderStatus(ctx *nanovgo.Context, x, y float32) {
	dy := float32(20)

	drawRunningValue(ctx, "Viz Engine:", "vizengine.exe", x, y)
	y += dy
	drawRunningValue(ctx, "NATS Server:", "nats-server.exe", x, y)
	y += dy
	drawRunningValue(ctx, "Viz Hub:", "vizhub.exe", x, y)
	y += dy
	drawRunningValue(ctx, "Element:", "Element.exe", x, y)
	y += dy
	drawRunningValue(ctx, "Morph:", "morph.exe", x, y)
	y += dy
	drawRunningValue(ctx, "Viz Input:", "vizio.exe", x, y)
	y += dy
	drawRunningValue(ctx, "Viz Spout:", "vizspout.exe", x, y)
}

func renderLog(ctx *nanovgo.Context, x, y float32) {
	dy := float32(20)

	drawLabel(ctx, "Log:", x, y)
	y += dy
	x += float32(50)
	drawLabel(ctx, "Line 1:", x, y)
}
*/

// ButtonDown xxx
var ButtonDown []bool

// MouseX xxx
var MouseX float32

// MouseY xxx
var MouseY float32

// Mousebutton xxx
func Mousebutton(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	if button < 0 || int(button) >= len(ButtonDown) {
		log.Printf("mousebutton: unexpected button=%d\n", button)
		return
	}
	if action == 1 {
		ButtonDown[button] = true
	} else {
		ButtonDown[button] = false
	}
}

// Mousepos xxx
func Mousepos(w *glfw.Window, xpos float64, ypos float64, xdelta float64, ydelta float64) {
	// log.Printf("mousepos callback xypos = %f,%f  xydelta=%f,%f\n", xpos, ypos, xdelta, ydelta)
	MouseX = float32(xpos)
	MouseY = float32(ypos)
}
