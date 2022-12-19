package agent

import (
	"github.com/vizicist/palette/engine"
)

func RegisterAgent(name string, agentFunc engine.AgentFunc) {
	engine.RegisterAgent(name, agentFunc)
}

func init() {
}
