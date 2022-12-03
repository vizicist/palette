package agent

import "github.com/vizicist/palette/engine"

var AllAgents = map[string]engine.Agent{}

func GetAgent(name string) engine.Agent {
	r, ok := AllAgents[name]
	if !ok {
		return nil
	}
	return r
}
func RegisterAgent(name string, agent engine.Agent) {
	engine.AddAgent(name, agent)
	AllAgents[name] = agent
}

func init() {
}
