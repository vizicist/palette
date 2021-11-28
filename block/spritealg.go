package block

import (
	"fmt"
	"log"

	"github.com/hypebeast/go-osc/osc"
	"github.com/vizicist/palette/engine"
)

func init() {
	engine.RegisterBlock("spritealg1", NewSpriteAlg1)
}

// SpriteAlg1 is a trivial visualization, for development
type SpriteAlg1 struct {
	context   engine.EContext
	oscPort   int
	oscClient *osc.Client
	layer     string // A, B, C, ...
}

// NewSpriteAlg1 xxx
func NewSpriteAlg1(ctx *engine.EContext) engine.Block {
	return &SpriteAlg1{
		oscPort: 0,
	}
}

// AcceptEngineMsg xxx
func (alg *SpriteAlg1) AcceptEngineMsg(ctx *engine.EContext, cmd engine.Cmd) string {

	switch cmd.Subj {

	case "setparam":
		name := cmd.ValuesString("name", "")
		value := cmd.ValuesString("value", "")
		log.Printf("SpriteAlg1: setparam %s=%s\n", name, value)
		switch name {
		case "oscport":
			i := cmd.ValuesInt("value", 0)
			if i <= 0 {
				e := fmt.Sprintf("SpriteAlg1: Bad value of oscport (%s)\n", value)
				return engine.ErrorResult(e)
			}
			if i != alg.oscPort {
				alg.oscClient = osc.NewClient("127.0.0.1", i)
				alg.oscPort = i
			}
		case "layer":
			alg.layer = value
		default:
			e := fmt.Sprintf("SpriteAlg1: Unknown parameter (%s)\n", name)
			return engine.ErrorResult(e)
		}

	case "cursor3d":
		// send an OSC message to Resolume
		oscmsg := osc.NewMessage("/api")
		oscmsg.Append("sprite")
		id := cmd.ValuesString("id", "")
		x := cmd.ValuesFloat("x", 0.0)
		y := cmd.ValuesFloat("y", 0.0)
		z := cmd.ValuesFloat("z", 0.0)
		oscmsg.Append(fmt.Sprintf("{ \"layer\": \"%s\", \"id\": \"%s\", \"x\": \"%f\", \"y\": \"%f\", \"z\": \"%f\" }",
			alg.layer, id, x, y, z))
		// log.Printf("SpriteAlg1: OSC = %s\n", oscmsg)
		alg.oscClient.Send(oscmsg)
	}
	return ""
}

//////////////////////////////////////////////////////////////////////
