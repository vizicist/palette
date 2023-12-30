package main

import (
	"fyne.io/fyne/v2"           // Import the base Fyne package
	"fyne.io/fyne/v2/app"       // Import the Fyne app package
	"fyne.io/fyne/v2/container" // Import the Fyne container package for layouts
	"fyne.io/fyne/v2/widget"    // Import the Fyne widget package for widgets

	"github.com/vizicist/palette/kit"
)

var Categories = []string{"global", "quad", "patch", "sound", "visual", "effect"}
var TopButton = map[string]*widget.Button{}
var TopLabel *widget.Label
var SelectGrid *fyne.Container

func SelectSaved(category string, saved string) {
	kit.LogInfo("GRID button", "category", category, "saved", saved)
	TopLabel.SetText("category=" + category + " file=" + saved)
}

func SelectCategory(category string) {
	TopLabel.SetText("category = " + category)
	list, err := kit.SavedFileList(category)
	if err != nil {
		kit.LogError(err)
	} else {
		SelectGrid.RemoveAll()
		for _, savedName := range list {
			s := savedName
			c := category
			SelectGrid.Add(widget.NewButton(s, func() {
				SelectSaved(c, s)
			}))
		}
	}
}

func main() {

	kit.InitLog("remote")
	kit.LogInfo("Creating app")
	kit.InitMisc()

	type FileList []string
	saved := map[string]FileList{}

	for _, s := range Categories {
		list, err := kit.SavedFileList(s)
		if err != nil {
			kit.LogError(err)
		} else {
			saved[s] = list
		}
	}

	myApp := app.New()                            // Create a new app
	myWindow := myApp.NewWindow("Palette Remote") // Create a new window

	appSize := fyne.Size{Width: 300, Height: 600}
	gridsize := fyne.Size{
		Width:  200,
		Height: 40,
	}

	myWindow.Resize(fyne.NewSize(appSize.Width, appSize.Height)) // Resize the window

	// Create a label (widget) with initial text
	TopLabel = widget.NewLabel("Palette Remote 2024")

	SelectGrid = container.NewGridWrap(gridsize)

	// Create the buttons along the top
	container_topbuttons := container.NewHBox()
	for _, category := range Categories {
		c := category
		TopButton[c] = widget.NewButton(c, func() {
			SelectCategory(c)
		})
		container_topbuttons.Add(TopButton[c])
	}

	// Use a container to arrange the label and button vertically
	content := container.NewVBox(
		container.NewCenter(TopLabel),
		container_topbuttons,
		SelectGrid,
	)

	myWindow.SetContent(content) // Set the window content to be the container
	myWindow.ShowAndRun()        // Show and run the application
}
