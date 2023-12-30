package main

import (
	"fyne.io/fyne/v2"           // Import the base Fyne package
	"fyne.io/fyne/v2/app"       // Import the Fyne app package
	"fyne.io/fyne/v2/container" // Import the Fyne container package for layouts
	"fyne.io/fyne/v2/widget" // Import the Fyne widget package for widgets
)

func main() {
	myApp := app.New()                   // Create a new app
	myWindow := myApp.NewWindow("Hello") // Create a new window

	myWindow.Resize(fyne.NewSize(300, 200)) // Resize the window

	// Create a label (widget) with initial text
	label := widget.NewLabel("Hello, Fyne!")

	// Create a button (widget)
	button := widget.NewButton("Click me!", func() {
		label.SetText("Button clicked!") // Change the label text when the button is clicked
	})

	// Use a container to arrange the label and button vertically
	content := container.NewVBox(
		label,
		button,
	)

	myWindow.SetContent(content) // Set the window content to be the container
	myWindow.ShowAndRun()        // Show and run the application
}
