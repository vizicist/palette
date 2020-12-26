package gui

import (
	"fmt"
	"image"
	"log"
	"time"

	// Don't be tempted to use go-gl
	"github.com/goxjs/gl"
	"github.com/goxjs/glfw"
	"github.com/micaelAlastor/nanovgo"
	"github.com/nats-io/nats.go"
	"github.com/vizicist/palette/engine"
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

	ctx, err := nanovgo.NewContext(0 /*nanovgo.AntiAlias | nanovgo.StencilStrokes | nanovgo.Debug*/)
	defer ctx.Delete()

	if err != nil {
		panic(err)
	}

	screen := NewScreen(ctx)

	glfwWindow.SetKeyCallback(key)
	glfwWindow.SetMouseButtonCallback(screen.Mousebutton)
	glfwWindow.SetMouseMovementCallback(screen.Mousepos)

	glfw.SwapInterval(0)

	frametime, _ := time.ParseDuration("33ms") // 30fps
	tm := time.Now()
	previousTime := tm

	err = LoadFonts(ctx)
	if err != nil {
		log.Printf("gui.Run: err=%s\n", err)
	}

	for !glfwWindow.ShouldClose() {

		// t, _ := fps.UpdateGraph()

		fbWidth, fbHeight := glfwWindow.GetFramebufferSize()
		newWidth, newHeight := glfwWindow.GetSize()

		if len(screen.Objects()) == 0 {
			screen.AddObject("root", NewContainer(screen))
			BuildInitialScreen(screen)
		}
		root := screen.Objects()["root"]
		if newWidth != root.Rect().Dx() || newHeight != root.Rect().Dy() {
			screen.Resize(image.Rect(0, 0, newWidth, newHeight))
		}

		pixelRatio := float32(fbWidth) / float32(root.Rect().Dx())
		gl.Viewport(0, 0, fbWidth, fbHeight)

		// background color
		gl.ClearColor(0.8, 0.8, 0.8, 1)

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT | gl.STENCIL_BUFFER_BIT)
		gl.Enable(gl.BLEND)
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
		gl.Enable(gl.CULL_FACE)
		gl.Disable(gl.DEPTH_TEST)

		ctx.BeginFrame(screen.Rect().Dx(), screen.Rect().Dy(), pixelRatio)

		screen.Draw(ctx)

		ctx.EndFrame()

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
	}
	log.Printf("End of gui.Run()\n")
}

// LoadFonts xxx
func LoadFonts(ctx *nanovgo.Context) error {
	fonts := map[string]string{
		"icons":  "entypo.ttf",
		"lucida": "lucon.ttf",
	}
	hadErr := false
	for name, filename := range fonts {
		path := engine.ConfigFilePath(filename)
		f := ctx.CreateFont(name, path)
		if f == -1 {
			log.Printf("LoadFonts: could not add font name=%s path=%s", name, path)
			hadErr = true
		}
	}
	if hadErr {
		return fmt.Errorf("LoadFonts: unable to load all fonts")
	}
	return nil
}

func key(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	// log.Printf("key=%d is ignored\n", key)
	/*
		if key == glfw.KeyEscape && action == glfw.Press {
			w.SetShouldClose(true)
		}
	*/
}
