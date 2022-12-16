package engine

import (
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

func NewBidule() *Bidule {
	return &Bidule{
		client: osc.NewClient(LocalAddress, BidulePort),
		port:   3210,
	}
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
