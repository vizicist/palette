package main

import (
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/vizicist/palette/kit"
)

var Categories = []string{"global", "quad", "patch", "sound", "visual", "effect"}
var CategoryButton = map[string]*widget.Button{}
var CurrentCategory = "global"
var StatusLabel *widget.Label
var SelectGrid *fyne.Container
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
		BottomButtons,
		container.NewCenter(StatusLabel),
	)
}

func GenerateEditPageForCategory(category string) *fyne.Container {
	items := container.NewVBox()
	items.Add(widget.NewLabel("ITEM1"))
	items.Add(widget.NewLabel("ITEM2"))
	items.Add(widget.NewLabel("ITEM3"))
	paramlist, err := kit.LoadParamsMapOfCategory(category,"_Current")
	if err != nil {
		items.Add(widget.NewLabel("NO PARAMETERS OF THAT CATEGORY?"))
		kit.LogError(err)
		return nil
	}
	limit := 10
	for nm := range paramlist {
		limit--
		if limit <= 0 {
			break
		}
		kit.LogInfo("nm="+nm)
		items.Add(widget.NewLabel(nm))
	}
	items.Refresh()
	return container.NewVBox(
		TopButtons,
		items,
		BottomButtons,
		container.NewCenter(StatusLabel),
	)
}

func SetCurrentContent(nm string) {

	var content *fyne.Container
	switch nm {
	case "edit":
		content = GenerateEditPageForCategory(CurrentCategory)
	case "select":
		content = GenerateSelectPageForCategory("select")

	}
	if content == nil {
		kit.LogWarn("No content with that name", "name", nm)
	} else {
		CurrentContentType = nm
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

func main() {

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

	RemoteWindow.Resize(fyne.NewSize(AppSize.Width, AppSize.Height)) // Resize the window

	// Create a label (widget) with initial text
	StatusLabel = widget.NewLabel("Palette Remote 2024")

	SelectGrid = container.NewGridWrap(gridsize)

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

	// The "edit" Content page is dynamically generated
	// the first time it's viewed

	SetCurrentContent("select")
	SelectCategory("quad")

	RemoteWindow.ShowAndRun() // Show and run the application
}
