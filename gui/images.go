package gui

import (
	"github.com/blizzy78/ebitenui/image"
	"github.com/blizzy78/ebitenui/widget"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/vizicist/palette/engine"
)

func imageFromFile(fname string) (image *ebiten.Image, err error) {
	path := engine.PalettePath(fname)
	image, _, err = ebitenutil.NewImageFromFile(path)
	return image, err
}

func loadGraphicImages(idle string, disabled string) (*widget.ButtonImageImage, error) {
	idleImage, err := imageFromFile(idle)
	if err != nil {
		return nil, err
	}

	var disabledImage *ebiten.Image
	if disabled != "" {
		disabledImage, err = imageFromFile(disabled)
		if err != nil {
			return nil, err
		}
	}

	return &widget.ButtonImageImage{
		Idle:     idleImage,
		Disabled: disabledImage,
	}, nil
}

func loadImageNineSlice(path string, centerWidth int, centerHeight int) (*image.NineSlice, error) {
	i, err := imageFromFile(path)
	if err != nil {
		return nil, err
	}

	w, h := i.Size()
	return image.NewNineSlice(i,
			[3]int{(w - centerWidth) / 2, centerWidth, w - (w-centerWidth)/2 - centerWidth},
			[3]int{(h - centerHeight) / 2, centerHeight, h - (h-centerHeight)/2 - centerHeight}),
		nil
}
