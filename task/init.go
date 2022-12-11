package agent

import (
	"context"

	"github.com/vizicist/palette/engine"
)

func RegisterTask(name string, task TaskFunc, taskContext context.Context) {
	engine.RegisterTask(name, task, taskContext)
}

func init() {
}
