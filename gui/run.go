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

	// demo MSAA
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

			screen, err = NewScreen("root")
			if err != nil {
				log.Printf("NewScreen: err=%s\n", err)
			}

			err = LoadFonts(screen.ctx)
			if err != nil {
				log.Printf("gui.Run: err=%s\n", err)
			}

			glfwWindow.SetKeyCallback(key)
			glfwWindow.SetMouseButtonCallback(screen.callbackForMousebutton)
			glfwWindow.SetMouseMovementCallback(screen.callbackForMousepos)
		}

		if newWidth != screen.rect.Dx() || newHeight != screen.rect.Dy() {
			screen.BuildScreenImage(newWidth, newHeight)
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

		screen.ctx.BeginFrame(screen.rect.Dx(), screen.rect.Dy(), pixelRatio)

		screen.CheckMouseInput()

		screen.Draw()

		screen.ctx.Restore()

		screen.ctx.EndFrame()

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

func key(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	// log.Printf("key=%d is ignored\n", key)
	/*
		if key == glfw.KeyEscape && action == glfw.Press {
			w.SetShouldClose(true)
		}
	*/
}
