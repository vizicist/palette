package agent

import (
	"github.com/vizicist/palette/engine"
)

func RegisterTask(name string, task engine.TaskMethods) {
	engine.RegisterTask(name, task)
}

func init() {
}
