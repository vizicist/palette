package engine

import (
	"strings"
	"sync"
	"time"

	"github.com/hypebeast/go-osc/osc"
)

type Bidule struct {
	mutex  sync.Mutex
	client *osc.Client
	port   int
}

const BidulePort = 3210

var theBidule *Bidule

func TheBidule() *Bidule {
	if theBidule == nil {
		theBidule = &Bidule{
			client: osc.NewClient(LocalAddress, BidulePort),
			port:   BidulePort,
		}
	}
	return theBidule
}

func (b *Bidule) Activate() {
	msg := osc.NewMessage("/play")
	msg.Append(int32(1)) // turn it on
	for i := 0; i < 10; i++ {
		dt := 5 * time.Second
		time.Sleep(dt)
		LogOfType("bidule", "Bidule.Activate is sending", "msg", msg)
		LogError(b.client.Send(msg))
	}
}

func (b *Bidule) ProcessInfo() *ProcessInfo {
	bidulePath := ConfigValueWithDefault("bidule", "")
	if bidulePath == "" {
		bidulePath = "C:\\Program Files\\Plogue\\Bidule\\Bidule.exe"
		LogWarn("No bidule value in settings, using default", "path", bidulePath)
	}
	if !FileExists(bidulePath) {
		LogWarn("No bidule found, looking for", "path", bidulePath)
		return nil
	}
	exe := bidulePath
	lastslash := strings.LastIndex(exe, "\\")
	if lastslash > 0 {
		exe = exe[lastslash+1:]
	}
	bidulefile := ConfigValueWithDefault("bidulefile", "")
	if bidulefile == "" {
		bidulefile = "default.bidule"
	}
	filepath := ConfigFilePath(bidulefile)
	return NewProcessInfo(exe, bidulePath, filepath, b.Activate)
}

func (b *Bidule) Reset() error {

	b.mutex.Lock()
	defer b.mutex.Unlock()

	msg := osc.NewMessage("/play")
	msg.Append(int32(0))
	LogOfType("bidule", "Bidule.Reset is sending", "msg", msg)
	err := b.client.Send(msg)
	if err != nil {
		return err
	}
	// Give Bidule time to react
	time.Sleep(400 * time.Millisecond)
	msg = osc.NewMessage("/play")
	msg.Append(int32(1))
	LogOfType("bidule", "Bidule.Reset is sending", "msg", msg)
	return b.client.Send(msg)
}