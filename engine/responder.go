package engine

import (
	"fmt"
	"log"
	"sync"
)

type ResponderManager struct {
	responders       map[string]Responder
	activeResponders map[string]Responder
	mutex            sync.RWMutex
}

type Responder interface {
	OnCursorEvent(e CursorEvent, rm *ResponderManager)
}

func NewResponderManager() *ResponderManager {
	return &ResponderManager{
		responders:       make(map[string]Responder),
		activeResponders: make(map[string]Responder),
		mutex:            sync.RWMutex{},
	}
}

func (rm *ResponderManager) AddResponder(name string, resp Responder) {
	_, ok := rm.responders[name]
	if !ok {
		rm.responders[name] = resp
	} else {
		log.Printf("Warning: ResponderManager.AddResponder is overwriting old name=%s\n", name)
	}
}

func (rm *ResponderManager) ActivateResponder(name string) error {
	resp, ok := rm.responders[name]
	if !ok {
		return fmt.Errorf("no responder named %s", name)
	}
	log.Printf("ActivateResponder: name=%s\n", name)
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
		log.Printf("CallResponders: name=%s\n", name)
		responder.OnCursorEvent(ce, rm)
	}
}
