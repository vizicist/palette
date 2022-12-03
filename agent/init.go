package agent

import "github.com/vizicist/palette/engine"

func RegisterAgent(name string, agent engine.Agent) {
	engine.RegisterAgent(name, agent)
}

func init() {
}
