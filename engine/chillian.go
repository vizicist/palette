package engine

import (
	"fmt"
)

func init() {
	chillian := &Chillian{}
	RegisterPlugin("chillian", chillian.Api)
}

type Chillian struct {
}

func (chillian *Chillian) Api(ctx *PluginContext, api string, apiargs map[string]string) (string, error) {
	switch api {
	case "start":
		LogInfo("Chillian start")
	case "event":
		// ctx.LogInfo("Chillian event")
		// ctx.ScheduleBytesNow(me.Bytes)
	default:
		return "", fmt.Errorf("unrecognized api %s", api)
	}
	return "", nil
}
