package engine

import (
	"fmt"
	"sync"
)

type ResponderManager struct {
	responders        map[string]Responder
	respondersContext map[string]*ResponderContext
	activeResponders  map[string]Responder
	mutex             sync.RWMutex
}

type ResponderContext struct {
	scheduler *Scheduler
}

type Responder interface {
	OnCursorEvent(ctx *ResponderContext, e CursorEvent)
}

func NewResponderManager() *ResponderManager {
	return &ResponderManager{
		responders:        make(map[string]Responder),
		respondersContext: make(map[string]*ResponderContext),
		activeResponders:  make(map[string]Responder),
		mutex:             sync.RWMutex{},
	}
}

func NewResponderContext() *ResponderContext {
	return &ResponderContext{
		scheduler: TheEngine().Scheduler,
	}
}

func (ctx *ResponderContext) CurrentClick() Clicks {
	return CurrentClick()
}

func (ctx *ResponderContext) ScheduleDebug() string {
	return fmt.Sprintf("%s", ctx.scheduler)
}

func (ctx *ResponderContext) ScheduleNoteNow(nt *Note) {
	click := CurrentClick()
	ctx.scheduler.ScheduleNoteAt(nt, click)
}

func (ctx *ResponderContext) ScheduleNoteAt(nt *Note, click Clicks) {
	if nt == nil {
		Warn("ResponderContext.ScheduleNoteAt: nt == nil?")
		return
	}
	ctx.scheduler.ScheduleNoteAt(nt, click)
}

func (rm *ResponderManager) AddResponder(name string, responder Responder) {
	_, ok := rm.responders[name]
	if !ok {
		Info("Adding new Responder", "name", name)
		rc := NewResponderContext()
		rm.responders[name] = responder
		rm.respondersContext[name] = rc
	} else {
		Warn("ResponderManager.AddResponder can't overwriting existing", "responder", name)
	}
}

func (rm *ResponderManager) ActivateResponder(name string) error {
	resp, ok := rm.responders[name]
	if !ok {
		return fmt.Errorf("no responder named %s", name)
	}
	Info("ActivateResponder", "name", name)
	rm.activeResponders[name] = resp
	return nil
}

func (rm *ResponderManager) DeactivateResponder(name string) error {
	_, ok := rm.responders[name]
	if !ok {
		return fmt.Errorf("DeactivateResponder: no responder named %s", name)
	}
	delete(rm.activeResponders, name)
	return nil
}

func (rm *ResponderManager) handleCursorEvent(ce CursorEvent) {
	for name, responder := range rm.responders {
		Info("CallResponders", "name", name)
		context, ok := rm.respondersContext[name]
		if !ok {
			Warn("ResponderManager.handle: no context", "name", name)
		} else {
			responder.OnCursorEvent(context, ce)
		}
	}
}
