package engine

import (
	"fmt"
)

// type AgentInfo struct {
// }
func TheAgentManager() *AgentManager {
	return TheEngine().AgentManager
}

func NewAgentContext(apiFunc AgentFunc) *AgentContext {
	return &AgentContext{
		api:           apiFunc,
		cursorManager: NewCursorManager(),
		params:        NewParamValues(),
		sources:       map[string]bool{},
	}
}

type AgentManager struct {
	agents map[string]*AgentContext
}

func NewAgentManager() *AgentManager {
	return &AgentManager{
		agents: make(map[string]*AgentContext),
		// agentContext: make(map[string]*AgentContext),
	}
}

func (tm *AgentManager) RegisterAgent(name string, apiFunc AgentFunc) {
	_, ok := tm.agents[name]
	if ok {
		LogWarn("RegisterAgent: existing agent", "agent", name)
	} else {
		tm.agents[name] = NewAgentContext(apiFunc)
		LogInfo("Registering Agent", "agent", name)
	}
}

func (tm *AgentManager) StartAgent(name string) error {
	agent, err := tm.GetAgentContext(name)
	if err != nil {
		return err
	}
	_, err = agent.api(agent, "start", nil)
	return err
}

/*
func (rm *AgentManager) handleCursorEvent(ce CursorEvent) {
	for name, agent := range rm.agents {
		DebugLogOfType("agent", "CallAgents", "name", name)
		context, ok := rm.agentsContext[name]
		if !ok {
			Warn("AgentManager.handle: no context", "name", name)
		} else {
			agent.OnCursorEvent(context, ce)
		}
	}
}

func (rm *AgentManager) handleMidiEvent(me MidiEvent) {
	for name, agent := range rm.agents {
		context, ok := rm.agentsContext[name]
		if !ok {
			Warn("AgentManager.handle: no context", "name", name)
		} else {
			agent.OnMidiEvent(context, me)
		}
	}
}
*/

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

func (pm *AgentManager) GetAgentContext(name string) (*AgentContext, error) {
	ctx, ok := pm.agents[name]
	if !ok {
		return nil, fmt.Errorf("no agent named %s", name)
	} else {
		return ctx, nil
	}
}

func (tm *AgentManager) handleCursorEvent(e CursorEvent) {
	for _, ctx := range tm.agents {
		if ctx.IsSourceAllowed(e.Source) {
			ctx.api(ctx, "event", e.ToMap())
		}
	}
}

func (tm *AgentManager) handleMidiEvent(e MidiEvent) {
	for _, ctx := range tm.agents {
		ctx.api(ctx, "event", e.ToMap())
	}
}

func (tm *AgentManager) handleClickEvent(e ClickEvent) {
	for _, ctx := range tm.agents {
		ctx.api(ctx, "event", e.ToMap())
	}
}
