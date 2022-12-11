package engine

import (
	"context"
	"fmt"
)

type TaskFunc func(ctx *TaskContext, e Event) (string, error)

type TaskData any

type TaskManager struct {
	tasks       map[string]TaskFunc
	taskContext map[string]*TaskContext
	// activeAgents  map[string]Agent
}

func NewTaskManager() *TaskManager {
	return &TaskManager{
		tasks:       make(map[string]TaskFunc),
		taskContext: make(map[string]*TaskContext),
	}
}

func (rm *TaskManager) RegisterTask(taskName string, task TaskFunc, taskContext context.Context) {
	_, ok := rm.taskContext[taskName]
	if !ok {
		Info("Registering Task", "task", taskName)
		rm.taskContext[taskName] = NewEngineContext(task, taskContext)
	} else {
		Warn("RegisterTask can't overwriting existing", "task", taskName)
	}
}

/*
func (rm *TaskManager) handleCursorEvent(ce CursorEvent) {
	for name, agent := range rm.agents {
		DebugLogOfType("agent", "CallAgents", "name", name)
		context, ok := rm.agentsContext[name]
		if !ok {
			Warn("TaskManager.handle: no context", "name", name)
		} else {
			agent.OnCursorEvent(context, ce)
		}
	}
}

func (rm *TaskManager) handleMidiEvent(me MidiEvent) {
	for name, agent := range rm.agents {
		context, ok := rm.agentsContext[name]
		if !ok {
			Warn("TaskManager.handle: no context", "name", name)
		} else {
			agent.OnMidiEvent(context, me)
		}
	}
}
*/

/*
func (pm *TaskManager) StartTask(name string) {
	ctx, ok := pm.agentsContext[name]
	if !ok {
		Warn("StartTask no such Agent", "agent", name)
	} else {
		ctx.agent.Start(ctx)
	}
}
*/

/*
func (pm *TaskManager) ApplyToAllAgents(f func(agent Agent)) {
	for _, agent := range pm.agents {
		f(agent)
	}
}
*/

/*
func (pm *TaskManager) ApplyToAgentsNamed(taskName string, f func(agent Agent)) {
	for name, ctx := range pm.agentsContext {
		if taskName == name {
			f(agent)
		}
	}
}

func (pm *TaskManager) GetAgent(taskName string) (Agent, error) {
	agent, ok := pm.agents[taskName]
	if !ok {
		return nil, fmt.Errorf("no agent named %s", taskName)
	} else {
		return agent, nil
	}
}
*/

func (pm *TaskManager) GetTaskContext(taskName string) (*TaskContext, error) {
	ctx, ok := pm.taskContext[taskName]
	if !ok {
		return nil, fmt.Errorf("no agent named %s", taskName)
	} else {
		return ctx, nil
	}
}

func (pm *TaskManager) handleCursorEvent(ce CursorEvent) {
	for _, ctx := range pm.taskContext {
		if ctx.IsSourceAllowed(ce.Source) {
			ctx.taskFunc(ctx, ctx.taskData)
		}
	}
}

func (pm *TaskManager) handleMidiEvent(me MidiEvent) {
	for _, ctx := range pm.taskContext {
		ctx.taskFunc(ctx,ctx.taskData)
	}
}
