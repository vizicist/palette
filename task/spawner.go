package agent

import (
	"fmt"

	"github.com/vizicist/palette/engine"
)

func init() {
	RegisterTask("spawner", &Spawner{})
}

type Spawner struct{}

func (spawner *Spawner) OnEvent(task *engine.Task, e engine.Event) {
	if ce, ok := e.(engine.ClickEvent); ok {
		task.LogInfo("Agent_processes.OnEvent", "click", ce.Click)
	}
}
func (spawner *Spawner) Start(task *engine.Task) {
}
func (spawner *Spawner) Stop(task *engine.Task) {
}
func (spawner *Spawner) Api(task *engine.Task, api string, apiargs map[string]string) (string, error) {
	return "", fmt.Errorf("Spawner.Api: no apis")
}
