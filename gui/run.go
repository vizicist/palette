package gui

import (
	"log"
	"time"

	// Don't be tempted to use go-gl

	"github.com/goxjs/gl"
	"github.com/goxjs/glfw"
	"github.com/nats-io/nats.go"
)

func midiHandler(msg *nats.Msg) {
	log.Printf("midiHandler: " + msg.Subject + " " + string(msg.Data))
}

func cursorHandler(msg *nats.Msg) {
	log.Printf("cursorHandler: " + msg.Subject + " " + string(msg.Data))
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

	var screen *Screen

	for !glfwWindow.ShouldClose() {

		// t, _ := fps.UpdateGraph()

		fbWidth, fbHeight := glfwWindow.GetFramebufferSize()
		newWidth, newHeight := glfwWindow.GetSize()

		if screen == nil {
			screen, err = NewScreen(glfwWindow, newWidth, newHeight)
			if err != nil {
				log.Printf("NewScreen: err=%s\n", err)
			}
		}

		if newWidth != screen.rect.Dx() || newHeight != screen.rect.Dy() {
			screen.Resize(newWidth, newHeight)
		}

		pixelRatio := float32(fbWidth) / float32(screen.rect.Dx())
		gl.Viewport(0, 0, fbWidth, fbHeight)

		// background color
		gl.ClearColor(0.8, 0.8, 0.8, 1)

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT | gl.STENCIL_BUFFER_BIT)
		gl.Enable(gl.BLEND)
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
		gl.Enable(gl.CULL_FACE)
		gl.Disable(gl.DEPTH_TEST)

		screen.nanoctx.BeginFrame(screen.rect.Dx(), screen.rect.Dy(), pixelRatio)

		screen.CheckMouseInput()

		screen.Draw()

		screen.nanoctx.Restore()

		screen.nanoctx.EndFrame()

		gl.Enable(gl.DEPTH_TEST)
		glfwWindow.SwapBuffers()
		glfw.PollEvents()

		previousTime = tm
		tm = time.Now()
		dt := tm.Sub(previousTime)
		if dt < frametime {
			tosleep := frametime - dt
			// log.Printf("Sleeping for %s\n", tosleep)
			time.Sleep(tosleep)
		}

		framenum++
	}
	log.Printf("End of gui.Run()\n")
}
