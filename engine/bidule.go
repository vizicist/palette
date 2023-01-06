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
			port:   3210,
		}
	}
	return theBidule
}

func (b *Bidule) Activate() {
	addr := "127.0.0.1"
	bidulePort := 3210
	biduleClient := osc.NewClient(addr, bidulePort)
	msg := osc.NewMessage("/play")
	msg.Append(int32(1)) // turn it on
	for i := 0; i < 10; i++ {
		dt := 5 * time.Second
		time.Sleep(dt)
		_ = biduleClient.Send(msg)
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

func (b *Bidule) Reset() {

	b.mutex.Lock()
	defer b.mutex.Unlock()

	msg := osc.NewMessage("/play")
	msg.Append(int32(0))
	b.client.Send(msg)
	// Give Bidule time to react
	time.Sleep(400 * time.Millisecond)
	msg = osc.NewMessage("/play")
	msg.Append(int32(1))
	b.client.Send(msg)
}
