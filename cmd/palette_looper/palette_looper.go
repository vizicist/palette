package main

import (
	"image/color"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

    // _ "github.com/vizicist/palette/twinsys"
)

type customRect struct {
	widget.BaseWidget
	color color.Color
}

func newCustomRect(color color.Color) *customRect {
	r := &customRect{color: color}
	r.ExtendBaseWidget(r)
	return r
}

func (r *customRect) CreateRenderer() fyne.WidgetRenderer {
	rect := canvas.NewRectangle(r.color)
	objects := []fyne.CanvasObject{rect}
	return widget.NewSimpleRenderer(objects[0])
}

func (r *customRect) MouseDown(*desktop.MouseEvent) {
	log.Println("Mouse down event detected")
}

func (r *customRect) MouseUp(*desktop.MouseEvent) {
	log.Println("MouseUp")
}

func (r *customRect) MouseIn(*desktop.MouseEvent) {
	log.Println("MouseIn")
}

func (r *customRect) MouseMoved(*desktop.MouseEvent) {
	log.Println("MouseMoved")
}

func (r *customRect) MouseOut() {
	log.Println("MouseOUt")
}

func main() {

	myApp := app.New()
	myWindow := myApp.NewWindow("Red Square with MouseDown Event")

	redSquare := newCustomRect(color.NRGBA{R: 255, G: 0, B: 0, A: 255})
	redSquare.Resize(fyne.NewSize(100, 100))
	redSquare.Move(fyne.Position{X: 100, Y: 100})

	greenSquare := newCustomRect(color.NRGBA{R: 0, G: 255, B: 0, A: 255})
	greenSquare.Resize(fyne.NewSize(100, 100))

	text1 := canvas.NewText("Hello", color.White)
	text1.Move(fyne.Position{X: 100, Y: 100})
	text1.TextSize = 100
	text1.Color = color.RGBA{R: 0, G: 255, B: 255, A: 255}

	content := container.NewWithoutLayout(redSquare, greenSquare)
	content.Add(text1)

	myWindow.SetContent(content)
	myWindow.Resize(fyne.NewSize(400, 640))

	myWindow.ShowAndRun()
}
