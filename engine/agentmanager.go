package engine

import (
	"fmt"
)

func (pm *AgentManager) StartAgent(name string) {
	ctx, ok := pm.agentsContext[name]
	if !ok {
		Warn("StartAgent no such Agent", "agent", name)
	} else {
		ctx.agent.Start(ctx)
	}
}

/*
func (pm *AgentManager) ApplyToAllAgents(f func(agent Agent)) {
	for _, agent := range pm.agents {
		f(agent)
	}
}
*/

/*
func (pm *AgentManager) ApplyToAgentsNamed(agentName string, f func(agent Agent)) {
	for name, ctx := range pm.agentsContext {
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
*/

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
	for _, ctx := range pm.agentsContext {
		_, allowed := ctx.sources[ce.Source]
		if allowed {
			ctx.agent.OnCursorEvent(ctx, ce)
		}
	}
}

func (pm *AgentManager) handleMidiEvent(me MidiEvent) {
	for _, ctx := range pm.agentsContext {
		ctx.agent.OnMidiEvent(ctx, me)
	}
}
