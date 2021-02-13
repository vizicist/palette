package isf

import (
	"github.com/vizicist/palette/glhf"
)

// Vizlet xxx
type Vizlet interface {
	Name() string
	DoVizlet() error
	OutputTexture() *glhf.Texture
	SetSpoutIn(string)
	SetSpoutOut(string)
	SetInputFromVizlet(Vizlet)
	AddSprite(Sprite) // Eventually this should go elsewhere, when cursor handling gets put into Vizlet interface
	AgeSprites()      // Ditto, when Vizlet interface starts handling click advance
}
