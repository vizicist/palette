package engine

import (
	"fmt"
)

func (pm *AgentManager) Start() {
	/*
		for _, agent := range pm.agents {
			agent.restoreCurrentSnap()
		}
	*/
}

func (pm *AgentManager) ApplyToAllAgents(f func(agent Agent)) {
	for _, agent := range pm.agents {
		f(agent)
	}
}

func (pm *AgentManager) ApplyToAgentsNamed(agentName string, f func(agent Agent)) {
	for name, agent := range pm.agents {
		if agentName == name {
			f(agent)
		}
	}
}

func (pm *AgentManager) GetAgent(agentName string) (Agent, error) {
	agent, ok := pm.agents[agentName]
	if !ok {
		return nil, fmt.Errorf("no agent named %s", agentName)
	} else {
		return agent, nil
	}
}

func (pm *AgentManager) GetAgentContext(agentName string) (*AgentContext, error) {
	ctx, ok := pm.agentsContext[agentName]
	if !ok {
		return nil, fmt.Errorf("no agent named %s", agentName)
	} else {
		return ctx, nil
	}
}

func (ctx *AgentContext) IsSourceAllowed(source string) bool {
	_, ok := ctx.sources[source]
	return ok
}

func (pm *AgentManager) handleCursorEvent(ce CursorEvent) {
	for name, agent := range pm.agents {
		ctx := pm.agentsContext[name]
		_, allowed := ctx.sources[ce.Source]
		if allowed {
			agent.OnCursorEvent(ctx, ce)
		}
	}
}

func (pm *AgentManager) handleMidiEvent(me MidiEvent) {
	for name, agent := range pm.agents {
		ctx := pm.agentsContext[name]
		agent.OnMidiEvent(ctx, me)
	}
}
