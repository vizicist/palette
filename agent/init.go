package agent

import (
	"github.com/vizicist/palette/engine"
)

func RegisterAgent(name string, agent engine.AgentMethods) {
	engine.RegisterAgent(name, agent)
}

func init() {
}
