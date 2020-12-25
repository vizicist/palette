package gui

import (
	"fmt"
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

	window, err := glfw.CreateWindow(600, 800, "Palette", nil, nil)
	if err != nil {
		panic(err)
	}

	window.MakeContextCurrent()

	ctx, err := nanovgo.NewContext(0 /*nanovgo.AntiAlias | nanovgo.StencilStrokes | nanovgo.Debug*/)
	defer ctx.Delete()

	if err != nil {
		panic(err)
	}

	screen := NewScreen(ctx)

	window.SetKeyCallback(key)
	window.SetMouseButtonCallback(screen.Mousebutton)
	window.SetMouseMovementCallback(screen.Mousepos)

	glfw.SwapInterval(0)

	frametime, _ := time.ParseDuration("33ms") // 30fps
	tm := time.Now()
	previousTime := tm

	err = LoadFonts(ctx)
	if err != nil {
		log.Printf("gui.Run: err=%s\n", err)
	}

	for !window.ShouldClose() {

		// t, _ := fps.UpdateGraph()

		fbWidth, fbHeight := window.GetFramebufferSize()
		newWidth, newHeight := window.GetSize()

		if newWidth != screen.width || newHeight != screen.height {
			screen.Resize(newWidth, newHeight)
			if len(screen.wind) == 0 {
				err := BuildInitialScreen(screen)
				if err != nil {
					log.Printf("BuildInitialScreen: err=%s\n", err)
				}
			}
		}

		pixelRatio := float32(fbWidth) / float32(screen.width)
		gl.Viewport(0, 0, fbWidth, fbHeight)

		// background color
		gl.ClearColor(0.8, 0.8, 0.8, 1)

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT | gl.STENCIL_BUFFER_BIT)
		gl.Enable(gl.BLEND)
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
		gl.Enable(gl.CULL_FACE)
		gl.Disable(gl.DEPTH_TEST)

		ctx.BeginFrame(screen.width, screen.height, pixelRatio)

		screen.Do()

		ctx.EndFrame()

		gl.Enable(gl.DEPTH_TEST)
		window.SwapBuffers()
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
