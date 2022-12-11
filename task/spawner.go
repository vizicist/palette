package agent

import (
	"context"

	"github.com/vizicist/palette/engine"
)

func init() {
	RegisterTask("spawner", &Spawner{}, nil)
}

type Spawner struct{}

func (spawner *Spawner) OnEvent(ctx context.Context, task engine.TaskInterface, e engine.Event) (string, error) {
	if ce, ok := e.(engine.ClickEvent); ok {
		task.Info("Agent_processes.OnEvent", "click", ce.Click)
	}
	return "", nil
}
