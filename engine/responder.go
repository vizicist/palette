package engine

type ResponderManager struct {
	responders map[string]Responder
	// respondersContext map[string]*ResponderContext
	// activeResponders  map[string]Responder
}

type Responder interface {
	OnCursorEvent(ctx *ResponderContext, e CursorEvent)
	OnMidiEvent(ctx *ResponderContext, e MidiEvent)
}

func NewResponderManager() *ResponderManager {
	return &ResponderManager{
		responders: make(map[string]Responder),
		// respondersContext: make(map[string]*ResponderContext),
		// activeResponders:  make(map[string]Responder),
	}
}

func (rm *ResponderManager) GetResponder(name string) Responder {
	r, ok := rm.responders[name]
	if !ok {
		return nil
	}
	return r
}

func (rm *ResponderManager) AddResponder(name string, responder Responder) {
	_, ok := rm.responders[name]
	if !ok {
		Info("Adding new Responder", "name", name)
		rm.responders[name] = responder
		// rc := NewResponderContext()
		// rm.respondersContext[name] = rc
	} else {
		Warn("ResponderManager.AddResponder can't overwriting existing", "responder", name)
	}
}

/*
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
*/

/*
func (rm *ResponderManager) handleCursorEvent(ce CursorEvent) {
	for name, responder := range rm.responders {
		DebugLogOfType("responder", "CallResponders", "name", name)
		context, ok := rm.respondersContext[name]
		if !ok {
			Warn("ResponderManager.handle: no context", "name", name)
		} else {
			responder.OnCursorEvent(context, ce)
		}
	}
}

func (rm *ResponderManager) handleMidiEvent(me MidiEvent) {
	for name, responder := range rm.responders {
		context, ok := rm.respondersContext[name]
		if !ok {
			Warn("ResponderManager.handle: no context", "name", name)
		} else {
			responder.OnMidiEvent(context, me)
		}
	}
}
*/
