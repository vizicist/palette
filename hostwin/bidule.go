package hostwin

import (
	"path/filepath"
	"time"

	"github.com/hypebeast/go-osc/osc"
	"github.com/vizicist/palette/kit"
)

const BidulePort = 3210

func (h HostWin) ActivateAudio() {
	msg := osc.NewMessage("/play")
	msg.Append(int32(1)) // turn it on
	for i := 0; i < 10; i++ {
		dt := 5 * time.Second
		time.Sleep(dt)
		LogOfType("bidule", "Bidule.Activate is sending", "msg", msg)
		h.SendOsc(h.biduleClient, msg)
	}
}

func (h HostWin) ProcessInfoBidule() *ProcessInfo {
	bidulePath, err := kit.GetParam("engine.bidulepath")
	if err != nil {
		LogIfError(err)
		return nil
	}
	if !FileExists(bidulePath) {
		LogWarn("No bidule found, looking for", "path", bidulePath)
		return nil
	}
	exe := filepath.Base(bidulePath)

	bidulefile, err := kit.GetParam("engine.bidulefile")
	if err != nil {
		LogIfError(err)
		return nil
	}
	filepath := ConfigFilePath(bidulefile)
	return NewProcessInfo(exe, bidulePath, filepath, h.ActivateAudio)
}

func (h HostWin) ResetAudio() {

	msg := osc.NewMessage("/play")
	msg.Append(int32(0))
	LogOfType("bidule", "Bidule.Reset is sending", "msg", msg)
	h.SendOsc(h.biduleClient, msg)

	// Give Bidule time to react
	time.Sleep(400 * time.Millisecond)
	msg = osc.NewMessage("/play")
	msg.Append(int32(1))
	LogOfType("bidule", "Bidule.Reset is sending", "msg", msg)
	h.SendOsc(h.biduleClient, msg)
}
