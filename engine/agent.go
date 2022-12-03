package engine

type AgentManager struct {
	agents        map[string]Agent
	agentsContext map[string]*AgentContext
	// activeAgents  map[string]Agent
}

type Agent interface {
	OnCursorEvent(ctx *AgentContext, e CursorEvent)
	OnMidiEvent(ctx *AgentContext, e MidiEvent)
}

func NewAgentManager() *AgentManager {
	return &AgentManager{
		agents:        make(map[string]Agent),
		agentsContext: make(map[string]*AgentContext),
		// activeAgents:  make(map[string]Agent),
	}
}

func (rm *AgentManager) AddAgent(name string, agent Agent) {
	_, ok := rm.agents[name]
	if !ok {
		Info("Adding new Agent", "name", name)
		rm.agents[name] = agent
		// rc := NewAgentContext()
		// rm.agentsContext[name] = rc
	} else {
		Warn("AgentManager.AddAgent can't overwriting existing", "agent", name)
	}
}

/*
func (rm *AgentManager) ActivateAgent(name string) error {
	resp, ok := rm.agents[name]
	if !ok {
		return fmt.Errorf("no agent named %s", name)
	}
	Info("ActivateAgent", "name", name)
	rm.activeAgents[name] = resp
	return nil
}

func (rm *AgentManager) DeactivateAgent(name string) error {
	_, ok := rm.agents[name]
	if !ok {
		return fmt.Errorf("DeactivateAgent: no agent named %s", name)
	}
	delete(rm.activeAgents, name)
	return nil
}
*/

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
