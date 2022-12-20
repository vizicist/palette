package agent

import (
	"strings"
	"sync"
	"time"

	"github.com/hypebeast/go-osc/osc"
	"github.com/vizicist/palette/engine"
)

type Bidule struct {
	mutex  sync.Mutex
	ctx  *engine.AgentContext
	client *osc.Client
	port   int
}

const BidulePort = 3210

func NewBidule(ctx *engine.AgentContext) *Bidule {
	return &Bidule{
		ctx:  ctx,
		client: osc.NewClient(engine.LocalAddress, BidulePort),
		port:   3210,
	}
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

func (b *Bidule) ProcessInfo() *processInfo {
	agent := b.ctx
	bidulePath := agent.ConfigValueWithDefault("bidule", "")
	if bidulePath == "" {
		bidulePath = "C:\\Program Files\\Plogue\\Bidule\\Bidule.exe"
		agent.LogWarn("No bidule value in settings, using default", "path", bidulePath)
	}
	if !agent.FileExists(bidulePath) {
		agent.LogWarn("No bidule found, looking for", "path", bidulePath)
		return nil
	}
	exe := bidulePath
	lastslash := strings.LastIndex(exe, "\\")
	if lastslash > 0 {
		exe = exe[lastslash+1:]
	}
	bidulefile := agent.ConfigValueWithDefault("bidulefile", "")
	if bidulefile == "" {
		bidulefile = "default.bidule"
	}
	filepath := agent.ConfigFilePath(bidulefile)
	return &processInfo{exe, bidulePath, filepath, b.Activate}
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
