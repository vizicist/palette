package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func main() {
	a := app.New()
	w := a.NewWindow("Rich Text Editor")

	// create a rich text widget with some initial content
	rt := widget.NewRichText(
		&widget.TextSegment{
			Text: "Hello, world!\n",
			Style: widget.RichTextStyle{
				Inline: true,
				ColorName: theme.ColorNameForeground,
				TextStyle: fyne.TextStyle{
					Bold: true,
				},
			},
		},
		&widget.TextSegment{
			Text: "This is a sample program that uses the widget.RichText struct of the fyne toolkit.\n",
			Style: widget.RichTextStyle{
				Inline: true,
				ColorName: theme.ColorNameForeground,
				TextStyle: fyne.TextStyle{
					Italic: true,
				},
			},
		},
		&widget.TextSegment{
			Text: "You can edit the text and apply different styles using the buttons below.\n",
			Style: widget.RichTextStyle{
				Inline: true,
				ColorName: theme.ColorNameForeground,
			},
		},
	)

	// create some buttons to change the text style
	boldBtn := widget.NewButton("Bold", func() {
		rt.Segments = append(rt.Segments, &widget.TextSegment{
			Text: "",
			Style: widget.RichTextStyle{
				Inline: true,
				TextStyle: fyne.TextStyle{
					Bold: true,
				},
			},
		})
		rt.Refresh()
	})

	italicBtn := widget.NewButton("Italic", func() {
		rt.Segments = append(rt.Segments, &widget.TextSegment{
			Text: "",
			Style: widget.RichTextStyle{
				Inline: true,
				TextStyle: fyne.TextStyle{
					Italic: true,
				},
			},
		})
		rt.Refresh()
	})

	redBtn := widget.NewButton("Red", func() {
		rt.Segments = append(rt.Segments, &widget.TextSegment{
			Text: "",
			Style: widget.RichTextStyle{
				Inline: true,
				ColorName: theme.ColorNameForeground,
			},
		})
		rt.Refresh()
	})

	blueBtn := widget.NewButton("Blue", func() {
		rt.Segments = append(rt.Segments, &widget.TextSegment{
			Text: "",
			Style: widget.RichTextStyle{
				Inline: true,
				ColorName: theme.ColorNameForeground,
			},
		})
		rt.Refresh()
	})

	// create a container to hold the rich text widget and the buttons
	c := container.NewBorder(nil, container.NewHBox(boldBtn, italicBtn, redBtn, blueBtn), nil, nil, rt)

	w.SetContent(c)
	w.ShowAndRun()
}
