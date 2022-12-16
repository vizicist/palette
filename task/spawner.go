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
	// if _, ok := e.(engine.ClickEvent); ok {
	// No need to log Click or Uptime, the log already includes them
	// task.LogInfo("Spawner.OnEvent")
	// }
}

func (spawner *Spawner) Start(task *engine.Task) error {
	return nil
}

func (spawner *Spawner) Stop(task *engine.Task) {
}

func (spawner *Spawner) Api(task *engine.Task, api string, apiargs map[string]string) (string, error) {
	return "", fmt.Errorf("Spawner.Api: no apis")
}
