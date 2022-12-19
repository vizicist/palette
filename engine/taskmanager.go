package engine

import (
	"fmt"
)

// type AgentInfo struct {
// }
func TheAgentManager() *AgentManager {
	return TheEngine().AgentManager
}

func NewAgent(apiFunc AgentFunc) *AgentContext {
	return &AgentContext{
		api:     apiFunc,
		sources: map[string]bool{},
		// scheduler: NewScheduler(),
		params: NewParamValues(),
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
		tm.agents[name] = NewAgent(apiFunc)
		LogInfo("Registering Agent", "agent", name)
	}
}

func (tm *AgentManager) StartAgent(name string) error {
	agent, err := tm.GetAgent(name)
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

func (pm *AgentManager) GetAgent(name string) (*AgentContext, error) {
	ctx, ok := pm.agents[name]
	if !ok {
		return nil, fmt.Errorf("no agent named %s", name)
	} else {
		return ctx, nil
	}
}

func (tm *AgentManager) handleCursorEvent(e CursorEvent) {
	for _, agent := range tm.agents {
		if agent.IsSourceAllowed(e.Source) {
			agent.api(agent, "event", e.ToMap())
		}
	}
}

func (tm *AgentManager) handleMidiEvent(e MidiEvent) {
	for _, agent := range tm.agents {
		agent.api(agent, "event", e.ToMap())
	}
}

func (tm *AgentManager) handleClickEvent(e ClickEvent) {
	for _, agent := range tm.agents {
		agent.api(agent, "event", e.ToMap())
	}
}
