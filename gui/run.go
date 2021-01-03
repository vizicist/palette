package gui

import (
	"image"
	"log"
	"math"
	"time"

	// Don't be tempted to use go-gl

	"github.com/goxjs/gl"
	"github.com/goxjs/glfw"
)

// RunContext xxx
type RunContext struct {
	screen       *Screen
	glfwMousePos image.Point
}

// Run xxx
func Run() {

	err := glfw.Init(gl.ContextWatcher)
	if err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.Samples, 4)

	glfwWindow, err := glfw.CreateWindow(600, 800, "Palette", nil, nil)
	if err != nil {
		panic(err)
	}

	glfwWindow.MakeContextCurrent()

	if err != nil {
		panic(err)
	}

	glfw.SwapInterval(0)

	frametime, _ := time.ParseDuration("33ms") // 30fps
	tm := time.Now()
	previousTime := tm

	framenum := 0

	runcontext := &RunContext{
		screen:       nil,
		glfwMousePos: image.Point{},
	}

	glfwWindow.SetKeyCallback(runcontext.gotKey)
	glfwWindow.SetMouseButtonCallback(runcontext.gotMouseButton)
	glfwWindow.SetMouseMovementCallback(runcontext.gotMousePos)

	for !glfwWindow.ShouldClose() {

		fbWidth, fbHeight := glfwWindow.GetFramebufferSize()
		newWidth, newHeight := glfwWindow.GetSize()

		if runcontext.screen == nil {
			runcontext.screen, err = NewScreen(newWidth, newHeight)
			if err != nil {
				log.Printf("NewScreen: err=%s\n", err)
			}
		}

		if newWidth != runcontext.screen.rect.Dx() || newHeight != runcontext.screen.rect.Dy() {
			runcontext.screen.Resize(newWidth, newHeight)
		}

		gl.Viewport(0, 0, fbWidth, fbHeight)

		// background color
		gl.ClearColor(0.8, 0.8, 0.8, 1)

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT | gl.STENCIL_BUFFER_BIT)
		gl.Enable(gl.BLEND)
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
		gl.Enable(gl.CULL_FACE)
		gl.Disable(gl.DEPTH_TEST)

		runcontext.screen.DoOneFrame(fbWidth, fbHeight)

		gl.Enable(gl.DEPTH_TEST)
		glfwWindow.SwapBuffers()
		glfw.PollEvents()

		previousTime = tm
		tm = time.Now()
		dt := tm.Sub(previousTime)
		if dt < frametime {
			tosleep := frametime - dt
			time.Sleep(tosleep)
		}

		framenum++
	}
	log.Printf("End of gui.Run()\n")
}

func (runcontext *RunContext) gotMouseButton(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	if button < 0 || int(button) >= len(runcontext.screen.mouseButtonDown) {
		log.Printf("mousebutton: unexpected value button=%d\n", button)
		return
	}
	down := false
	if action == 1 {
		down = true
	}
	runcontext.screen.QueueMouseEvent(MouseEvent{
		pos:      runcontext.glfwMousePos,
		button:   int(button),
		down:     down,
		modifier: int(mods),
	})
}

func (runcontext *RunContext) gotMousePos(w *glfw.Window, xpos float64, ypos float64, xdelta float64, ydelta float64) {
	// All palette.gui coordinates are integer, but some glfw platforms may support
	// sub-pixel cursor positions.
	if math.Mod(xpos, 1.0) != 0.0 || math.Mod(ypos, 1.0) != 0.0 {
		log.Printf("Mousepos: we're getting sub-pixel mouse coordinates!\n")
	}
	runcontext.glfwMousePos = image.Point{X: int(xpos), Y: int(ypos)}
}

func (runcontext *RunContext) gotKey(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	log.Printf("key=%d is ignored\n", key)
}
