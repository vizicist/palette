package engine

import (
	"fmt"
)

// type TaskInfo struct {
// }
func TheTaskManager() *TaskManager {
	return TheEngine().TaskManager
}

func NewTask(methods TaskMethods) *Task {
	return &Task{
		methods: methods,
		sources: map[string]bool{},
		// scheduler: NewScheduler(),
		params: NewParamValues(),
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

func (tm *TaskManager) RegisterTask(name string, methods TaskMethods) {
	_, ok := tm.tasks[name]
	if ok {
		LogWarn("RegisterTask: existing task", "task", name)
	} else {
		tm.tasks[name] = NewTask(methods)
		LogInfo("Registering Task", "task", name)
	}
}

func (tm *TaskManager) StartTask(name string) error {
	task, err := tm.GetTask(name)
	if err != nil {
		return err
	}
	task.methods.Start(task)
	return nil
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

func (tm *TaskManager) handleCursorEvent(e CursorEvent) {
	for _, task := range tm.tasks {
		if task.IsSourceAllowed(e.Source) {
			task.methods.OnEvent(task, e)
		}
	}
}

func (tm *TaskManager) handleMidiEvent(e MidiEvent) {
	for _, task := range tm.tasks {
		task.methods.OnEvent(task, e)
	}
}

func (tm *TaskManager) handleClickEvent(e ClickEvent) {
	for _, task := range tm.tasks {
		task.methods.OnEvent(task, e)
	}
}
