package main

import (
	_ "embed"
	"fmt"
	"time"

	. "modernc.org/tk9.0"
	_ "modernc.org/tk9.0/themes/azure"
)

type Gui struct {
	nextMode    string
	currentMode string
}

var PaletteApp = &Gui{}
var UpdateChan = make(chan func())


var (
	//go:embed palette.png
	paletteicon []byte
)

func main() {

	guisize := "small"
	err := ActivateTheme("azure light")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		panic(err)
	}

	switch guisize {
	case "palette":
		WmGeometry(App, "800x1280")
	case "small":
		WmGeometry(App, "400x640")
	case "medium":
		WmGeometry(App, "500x800")
	default:
		App.Exit()
	}

	PaletteApp.nextMode = ""
	PaletteApp.currentMode = ""

	hello := Label(Txt("Hello, World!"))

	// Create a button to update the label's text
	button := TButton(Txt("Update Text"), Command(func() {
		hello.Configure(Txt("Updated Text"))
	}))

	Pack(hello, TExit(), button)

	App.IconPhoto(NewPhoto(Data(paletteicon)))

	go func() {
		time.Sleep(time.Second * 3)
		UpdateChan <- func() {
			hello.Configure(Txt("FIRST 3 SECS"))
		}
		time.Sleep(time.Second * 3)
		UpdateChan <- func() {
			hello.Configure(Txt("SECOND 3 SECS"))
		}
	}()

	var f func()
	_, _ = NewTicker(1000*time.Millisecond, func() {
		select {
		case value := <-UpdateChan:
			f = value
			f()
		default:
			hello.Configure(Txt(time.Now().Format(time.DateTime)))
		}
	})

	App.Center().Wait()
}
