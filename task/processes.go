package agent

import (
	"github.com/vizicist/palette/engine"
)

type ProcessContext struct {
	dummy string
}

func init() {
	ctx := context.WithValue(
		context.Background(),
		ProcessContext{"hello"},
		"context")
	RegisterTask("processes", Processes_OnEvent, ctx)
}

func Processes_OnEvent(ctx *engine.TaskContext, e engine.Event) (string, error) {
	if ce,ok := e.(ctx.Engine.ClickEvent) {
		ctx.Engine.Info("Agent_processes.OnEvent", "click", e.click)
	}
	return "", nil
}
