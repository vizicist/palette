package main

import (
	"fmt"
	"image/color"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/vizicist/palette/kit"
)

var Categories = []string{"global", "quad", "patch", "sound", "visual", "effect"}
var CategoryButton = map[string]*widget.Button{}
var CurrentCategory = "global"
var StatusLabel *widget.Label
var SelectGrid *fyne.Container
var SelectList *fyne.Container
var SelectButton = map[string]*widget.Button{}
var SelectButtonName string
var ActionButtons = map[string]*ActionButton{}
var BottomButtons *fyne.Container
var CurrentContentType string
var RemoteWindow fyne.Window
var TopButtons *fyne.Container
var AppSize fyne.Size

func SelectSaved(category string, saved string) {
	kit.LogInfo("GRID button", "category", category, "saved", saved)
	StatusLabel.SetText("category=" + category + " file=" + saved)
	if SelectButtonName != "" {
		// deselect
		SelectButton[SelectButtonName].Importance = widget.MediumImportance
		SelectButton[SelectButtonName].Refresh()
	}
	SelectButtonName = saved
	SelectButton[saved].Importance = widget.SuccessImportance
	SelectButton[saved].Refresh()
}

var LastSelectedCategory = ""

func SelectCategory(category string) {
	if LastSelectedCategory == category {
		ToggleContent()
	}
	LastSelectedCategory = category
	StatusLabel.SetText("category = " + category)
	list, err := kit.SavedFileList(category)
	if err != nil {
		kit.LogError(err)
	} else {
		// Clear select stuff
		SelectButtonName = ""
		SelectGrid.RemoveAll()
		SelectButton = map[string]*widget.Button{}
		if CurrentCategory != "" {
			CategoryButton[CurrentCategory].Importance = widget.MediumImportance
			CategoryButton[CurrentCategory].Refresh()
		}
		CurrentCategory = category
		CategoryButton[CurrentCategory].Importance = widget.SuccessImportance
		CategoryButton[CurrentCategory].Refresh()
		for _, savedName := range list {
			s := savedName
			c := category
			b := widget.NewButton(s, func() { SelectSaved(c, s) })
			SelectButton[s] = b
			SelectGrid.Add(b)
		}
	}
}

type ActionButton struct {
	*widget.Button
}

func (b *ActionButton) Tapped(*fyne.PointEvent) {
	b.Importance = widget.SuccessImportance
	b.Refresh()
	defer func() { // TODO move to a real animation
		time.Sleep(time.Millisecond * 400)
		b.Importance = widget.MediumImportance
		b.Refresh()
	}()
	b.Refresh()
	if b.OnTapped != nil && !b.Disabled() {
		b.OnTapped()
	}
}

func NewActionButton(nm string, f func()) *ActionButton {
	a := &ActionButton{widget.NewButton(nm, f)}
	ActionButtons[nm] = a
	return a
}

func GenerateEmptyPage() *fyne.Container {
	label := widget.NewLabel("This Page Is Blank?")
	return container.NewVBox(
		TopButtons,
		container.NewCenter(label),
		NewMouseContainer(),
		BottomButtons,
	)
}

func NewMouseContainer() *fyne.Container {
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	mousewidget := canvas.NewRectangle(red)
	mousewidget.Resize(fyne.NewSize(200, 200))
	mousewidget.SetMinSize(fyne.NewSize(200, 200))
	mousewidget2 := canvas.NewRectangle(red)
	mousewidget2.Resize(fyne.NewSize(100, 100))
	mousewidget2.SetMinSize(fyne.NewSize(100, 100))

	realwidget := NewMouseWidget()
	// return container.NewWithoutLayout( mousewidget, mousewidget2, realwidget)

	return container.NewWithoutLayout(realwidget)
}

func GenerateSelectPageForCategory(category string) *fyne.Container {

	scrollableGrid := container.NewVScroll(SelectGrid)
	scrollableGrid.SetMinSize(fyne.Size{
		Width:  AppSize.Width,
		Height: AppSize.Height,
	})

	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	top := canvas.NewRectangle(red)
	bottom := canvas.NewRectangle(red)
	left := canvas.NewRectangle(red)
	right := canvas.NewRectangle(red)
	middle := container.NewBorder(top, bottom, left, right, scrollableGrid)

	// Use a container to arrange the label and button vertically
	return container.NewVBox(
		TopButtons,
		middle,
		NewMouseContainer(),
		BottomButtons,
		container.NewCenter(StatusLabel),
	)
}

func GenerateEditPageForCategory(category string) *fyne.Container {

	scrollableList := container.NewVScroll(SelectList)
	scrollableList.SetMinSize(fyne.Size{
		Width:  AppSize.Width,
		Height: AppSize.Height,
	})

	SelectList.RemoveAll()

	/*
		// items := container.NewVBox()
		paramlist, err := kit.LoadParamsMapOfCategory(category, "_Current")
		if err != nil {
			SelectList.Add(widget.NewLabel("NO PARAMETERS OF THAT CATEGORY?"))
			kit.LogError(err)
			return nil
		}
	*/
	paramlist := []string{}
	for nm, def := range kit.ParamDefs {
		if strings.HasPrefix(nm, category+"._") {
			continue
		}
		if def.Category == category {
			paramlist = append(paramlist, nm)
		}
	}
	slices.Sort(paramlist)
	for _, nm := range paramlist {
		kit.LogInfo("nm=" + nm)
		SelectList.Add(widget.NewLabel(nm))
	}
	SelectList.Refresh()

	return container.NewVBox(
		TopButtons,
		scrollableList,
		BottomButtons,
		container.NewCenter(StatusLabel),
	)
}

var EditPage = map[string]*fyne.Container{}

func GenerateEditPages() {
	for _, category := range []string{"global",
		"misc", "sound", "visual", "effect"} {
		EditPage[category] = GenerateEditPageForCategory(category)
	}
}

func SetCurrentContent(contentType string) {

	var content *fyne.Container
	switch contentType {
	case "edit":
		content = EditPage[CurrentCategory]
	case "select":
		content = GenerateSelectPageForCategory(CurrentCategory)
	}
	if content == nil {
		kit.LogWarn("No content of that type", "contentType", contentType)
		CurrentContentType = "empty"
		RemoteWindow.SetContent(GenerateEmptyPage())
	} else {
		CurrentContentType = contentType
		RemoteWindow.SetContent(content) // Set the window content to be the container
	}
}

func ToggleContent() {
	if CurrentContentType == "select" {
		SetCurrentContent("edit")
	} else {
		SetCurrentContent("select")

	}
}

func oldmain() {

	kit.InitLog("remote")
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

	myApp := app.New()                               // Create a new app
	RemoteWindow = myApp.NewWindow("Palette Remote") // Create a new window

	AppSize = fyne.Size{Width: 300, Height: 600}
	gridsize := fyne.Size{
		Width:  200,
		Height: 40,
	}
	listsize := fyne.Size{
		Width:  200,
		Height: 30,
	}

	RemoteWindow.Resize(fyne.NewSize(AppSize.Width, AppSize.Height)) // Resize the window

	// Create a label (widget) with initial text
	StatusLabel = widget.NewLabel("Palette Remote 2024")

	SelectGrid = container.NewGridWrap(gridsize)
	SelectList = container.NewGridWrap(listsize)

	// Create the buttons along the top
	TopButtons = container.NewHBox()
	for _, category := range Categories {
		c := category
		CategoryButton[c] = widget.NewButton(c, func() {
			SelectCategory(c)
		})
		TopButtons.Add(CategoryButton[c])
	}

	BottomButtons = container.NewHBox()
	BottomButtons.Add(NewActionButton("Complete Reset", func() {
		kit.LogInfo("Should be doing Complete reset")
	}))
	BottomButtons.Add(NewActionButton("Clear", func() {
		kit.LogInfo("Should be doing Clear")
	}))
	BottomButtons.Add(NewActionButton("Help", func() {
		kit.LogInfo("Should be doing Help")
	}))

	GenerateEditPages()

	SetCurrentContent("select")
	SelectCategory("quad")

	RemoteWindow.ShowAndRun() // Show and run the application
}

// MouseWidget is a widget that implements the desktop.Mouseable interface
type MouseWidget struct {
	rect *canvas.Rectangle
}

func (w *MouseWidget) Refresh() {
	w.rect.Refresh()
}

func (w *MouseWidget) Hide() {
	w.rect.Hide()
}

func (w *MouseWidget) Show() {
	w.rect.Show()
}
func (w *MouseWidget) Visible() bool {
	return w.rect.Visible()
}

func (w *MouseWidget) Resize(size fyne.Size) {
	w.rect.Resize(size)
}
func (w *MouseWidget) MinSize() fyne.Size {
	return w.rect.MinSize()
}
func (w *MouseWidget) Size() fyne.Size {
	return w.rect.Size()
}
func (w *MouseWidget) Position() fyne.Position {
	return w.rect.Position()
}
func (w *MouseWidget) Move(pos fyne.Position) {
	w.rect.Move(pos)
}

// MouseDown is called when a mouse button is pressed
func (w *MouseWidget) MouseDown(ev *desktop.MouseEvent) {
	fmt.Printf("Mouse down at %v with button %v and modifier %v\n", ev.Position, ev.Button, ev.Modifier)
}

// MouseUp is called when a mouse button is released
func (w *MouseWidget) MouseUp(ev *desktop.MouseEvent) {
	fmt.Printf("Mouse up at %v with button %v and modifier %v\n", ev.Position, ev.Button, ev.Modifier)
}

// MouseMoved is called when the mouse pointer is moved
func (w *MouseWidget) MouseMoved(ev *desktop.MouseEvent) {
	fmt.Printf("Mouse moved at %v with button %v and modifier %v\n", ev.Position, ev.Button, ev.Modifier)
}

// NewMouseWidget creates a new mouse widget
func NewMouseWidget() *MouseWidget {
	red := color.RGBA{R: 0, G: 255, B: 0, A: 255}
	w := canvas.NewRectangle(red)
	// w := &MouseWidget{}
	w.SetMinSize(fyne.NewSize(300, 300))
	w.FillColor = color.RGBA{R: 0, G: 255, B: 0, A: 255}
	w.StrokeColor = color.RGBA{R: 0, G: 255, B: 0, A: 255}
	w.StrokeWidth = 5.0
	w.Refresh()
	w.Show()
	fmt.Printf("show=%v\n",w.Visible())
	return &MouseWidget{rect: w}
}

func main() {
	a := app.New()
	w := a.NewWindow("Mouse Events")

	w.SetContent(NewMouseWidget())
	w.ShowAndRun()
}
