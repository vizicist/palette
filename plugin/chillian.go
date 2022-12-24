package plugin

import (
	"fmt"

	"github.com/vizicist/palette/engine"
)

func init() {
	chillian := &Chillian{}
	engine.RegisterPlugin("chillian", chillian.Api)
}

type Chillian struct {
}

func (chillian *Chillian) Api(ctx *engine.PluginContext, api string, apiargs map[string]string) (string, error) {
	switch api {
	case "start":
		ctx.LogInfo("Chillian start")
	case "event":
		// ctx.LogInfo("Chillian event")
		// ctx.ScheduleBytesNow(me.Bytes)
	default:
		return "", fmt.Errorf("unrecognized api %s", api)
	}
	return "", nil
}
