package spout

import (
	"github.com/vizicist/palette/internal/libspout"
)

// Config returns a config string
func Config() string {
	return "spout config"
}

// Sender is a spout sender
type Sender struct {
	sender libspout.Sender
}

// CreateSender returns a Sender
func CreateSender(name string, width int, height int) *Sender {
	var s Sender
	s.sender = libspout.CreateSender(name, width, height)
	return &s
}

// SendTexture sends a texture
func SendTexture(s Sender, texture uint32, width int, height int) bool {
	return libspout.SendTexture(s.sender, texture, width, height)
}

// CreateReceiver creates a Receiver
func CreateReceiver(sendername string, width *int, height *int, bUseActive bool) bool {
	b := libspout.CreateReceiver(sendername, width, height, bUseActive)
	return b
}

// ReleaseReceiver releases things
func ReleaseReceiver() {
	libspout.ReleaseReceiver()
}

// ReceiveTexture receives a texture
func ReceiveTexture(sendername string, width *int, height *int, textureID int, textureTarget int, bInvert bool, hostFBO int) bool {
	b := libspout.ReceiveTexture(sendername, width, height, textureID, textureTarget, bInvert, hostFBO)
	return b
}
