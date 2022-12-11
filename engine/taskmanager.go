package engine

import (
	"fmt"
)

// type TaskInfo struct {
// }

func NewTask(methods TaskMethods) *Task {
	return &Task{
		methods:   methods,
		sources:   map[string]bool{},
		scheduler: NewScheduler(),
		params:    NewParamValues(),
	}
}

type TaskManager struct {
	tasks map[string]*Task
}

func NewTaskManager() *TaskManager {
	return &TaskManager{
		tasks: make(map[string]*Task),
		// taskContext: make(map[string]*TaskContext),
	}
}

func (rm *TaskManager) RegisterTask(name string, methods TaskMethods) {
	_, ok := rm.tasks[name]
	if ok {
		Warn("RegisterTask: existing task", "task", name)
	} else {
		rm.tasks[name] = NewTask(methods)
		Info("Registering Task", "task", name)
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

func (pm *TaskManager) GetTask(name string) (*Task, error) {
	ctx, ok := pm.tasks[name]
	if !ok {
		return nil, fmt.Errorf("no task named %s", name)
	} else {
		return ctx, nil
	}
}

func (pm *TaskManager) handleCursorEvent(e CursorEvent) {
	for _, task := range pm.tasks {
		if task.IsSourceAllowed(e.Source) {
			task.methods.OnEvent(task, e)
		}
	}
}

func (pm *TaskManager) handleMidiEvent(e MidiEvent) {
	for _, task := range pm.tasks {
		task.methods.OnEvent(task, e)
	}
}

func (pm *TaskManager) handleClickEvent(e ClickEvent) {
	for _, task := range pm.tasks {
		task.methods.OnEvent(task, e)
	}
}
