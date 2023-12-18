package kit

import (
	"path/filepath"
	"sync"
	"time"
	"runtime/debug"
	"fmt"

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
		TheEngine.SendOsc(b.client, msg)
	}
}

func (b *Bidule) ProcessInfo() *ProcessInfo {
	bidulePath, err := GetParam("global.bidulepath")
	if err != nil {
		LogIfError(err)
		return EmptyProcessInfo()
	}
	if !FileExists(bidulePath) {
		LogWarn("No bidule found, looking for", "path", bidulePath)
		return EmptyProcessInfo()
	}
	exe := filepath.Base(bidulePath)

	bidulefile, err := GetParam("global.bidulefile")
	if err != nil {
		LogIfError(err)
		return EmptyProcessInfo()
	}
	filepath := ConfigFilePath(bidulefile)
	return NewProcessInfo(exe, bidulePath, filepath, b.Activate)
}

func (b *Bidule) Reset() {

	defer func() {
		if r := recover(); r != nil {
			// Print stack trace in the error messages
			stacktrace := string(debug.Stack())
			// First to stdout, then to log file
			fmt.Printf("PANIC: recover in Bidule.Reset called, r=%+v stack=%v", r, stacktrace)
			err := fmt.Errorf("PANIC: recover in Bidule.Reset has been called")
			LogError(err, "r", r, "stack", stacktrace)
		}
	}()


	b.mutex.Lock()
	defer b.mutex.Unlock()

	msg := osc.NewMessage("/play")
	msg.Append(int32(0))
	LogOfType("bidule", "Bidule.Reset is sending", "msg", msg)
	TheEngine.SendOsc(b.client, msg)

	// Give Bidule time to react
	time.Sleep(400 * time.Millisecond)
	msg = osc.NewMessage("/play")
	msg.Append(int32(1))
	LogOfType("bidule", "Bidule.Reset is sending", "msg", msg)
	TheEngine.SendOsc(b.client, msg)
}
