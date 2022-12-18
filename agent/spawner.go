package agent

/*
import (
	"fmt"

	"github.com/vizicist/palette/engine"
)

func init() {
	RegisterAgent("spawner", &Spawner{})
}

type Spawner struct {
	processManager *ProcessManager
}

func (spawner *Spawner) OnEvent(agent *engine.Agent, e engine.Event) {
	// if _, ok := e.(engine.ClickEvent); ok {
	// No need to log Click or Uptime, the log already includes them
	// agent.LogInfo("Spawner.OnEvent")
	// }
}

func (spawner *Spawner) Start(agent *engine.Agent) error {
	spawner.processManager = NewProcessManager(agent)

	return nil
}

func (spawner *Spawner) Stop(agent *engine.Agent) {
	if agent.ConfigBoolWithDefault("killonstartup", true) {
		spawner.processManager.killAll()
	}
}

func (spawner *Spawner) Api(agent *engine.Agent, api string, apiargs map[string]string) (result string, err error) {

	result = ""

	switch api {
	default:
		err = fmt.Errorf("Spawner.Api: no api %s", api)
	}

	return result, err
}

*/
