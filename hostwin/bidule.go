package hostwin

import (
	"path/filepath"
	"time"

	"github.com/hypebeast/go-osc/osc"
	"github.com/vizicist/palette/kit"
)

const BidulePort = 3210
var BiduleClient *osc.Client

func (h HostWin) ActivateAudio() {
	msg := osc.NewMessage("/play")
	msg.Append(int32(1)) // turn it on
	for i := 0; i < 10; i++ {
		dt := 5 * time.Second
		time.Sleep(dt)
		kit.LogOfType("bidule", "Bidule.Activate is sending", "msg", msg)
		kit.SendOsc(BiduleClient, msg)
	}
}

func ProcessInfoBidule() *ProcessInfo {
	bidulePath, err := kit.GetParam("engine.bidulepath")
	if err != nil {
		LogIfError(err)
		return nil
	}
	if !kit.TheHost.FileExists(bidulePath) {
		LogWarn("No bidule found, looking for", "path", bidulePath)
		return nil
	}
	exe := filepath.Base(bidulePath)

	bidulefile, err := kit.GetParam("engine.bidulefile")
	if err != nil {
		LogIfError(err)
		return nil
	}
	filepath := kit.TheHost.ConfigFilePath(bidulefile)
	return NewProcessInfo(exe, bidulePath, filepath, kit.TheHost.ActivateAudio)
}

func (h HostWin) ResetAudio() {

	msg := osc.NewMessage("/play")
	msg.Append(int32(0))
	kit.LogOfType("bidule", "Bidule.Reset is sending", "msg", msg)
	kit.SendOsc(BiduleClient, msg)

	// Give Bidule time to react
	time.Sleep(400 * time.Millisecond)
	msg = osc.NewMessage("/play")
	msg.Append(int32(1))
	kit.LogOfType("bidule", "Bidule.Reset is sending", "msg", msg)
	kit.SendOsc(BiduleClient, msg)
}
