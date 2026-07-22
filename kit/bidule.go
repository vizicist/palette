package kit

import (
	"fmt"
	"path/filepath"
	"runtime/debug"
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
	time.Sleep(2 * time.Second)
	b.Reset()
	msg := bidulePlayMessage(true)
	for i := 0; i < 10; i++ {
		dt := 5 * time.Second
		time.Sleep(dt)
		LogOfType("bidule", "Bidule.Activate is sending", "msg", msg)
		theEngine.SendOsc(b.client, msg)
	}
}

func (b *Bidule) ProcessInfo() *ProcessInfo {
	bidulePath, err := GetParam("global.bidulepath")
	if err != nil {
		LogIfError(err)
		return EmptyProcessInfo()
	}
	if !FileExists(bidulePath) {
		// The configured default is a Windows path, so fall back to the
		// usual install locations for this OS before giving up.
		found := firstExistingPath(BiduleCandidatePaths())
		if found == "" {
			LogWarn("No bidule found, looking for", "path", bidulePath)
			return EmptyProcessInfo()
		}
		LogInfo("Bidule found at its default install location", "path", found)
		bidulePath = found
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

	SendAllNotesOffToSynths()

	msg := bidulePlayMessage(false)
	LogOfType("bidule", "Bidule.Reset is sending", "msg", msg)
	theEngine.SendOsc(b.client, msg)

	// Give Bidule time to react
	time.Sleep(400 * time.Millisecond)
	SendAllNotesOffToSynths()
	msg = bidulePlayMessage(true)
	LogOfType("bidule", "Bidule.Reset is sending", "msg", msg)
	theEngine.SendOsc(b.client, msg)
}

func bidulePlayMessage(on bool) *osc.Message {
	msg := osc.NewMessage("/play")
	if on {
		msg.Append(int32(1))
	} else {
		msg.Append(int32(0))
	}
	return msg
}
