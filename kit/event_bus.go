package kit

import "sync"

type EventBus[T comparable] struct {
	mutex       sync.RWMutex
	subscribers map[T][]func()
}

func NewEventBus[T comparable]() *EventBus[T] {
	return &EventBus[T]{subscribers: map[T][]func(){}}
}

func (bus *EventBus[T]) Subscribe(event T, handler func()) {
	if bus == nil || handler == nil {
		return
	}
	bus.mutex.Lock()
	bus.subscribers[event] = append(bus.subscribers[event], handler)
	bus.mutex.Unlock()
}

func (bus *EventBus[T]) Publish(event T) {
	if bus == nil {
		return
	}
	bus.mutex.RLock()
	handlers := append([]func(){}, bus.subscribers[event]...)
	bus.mutex.RUnlock()
	for _, handler := range handlers {
		handler()
	}
}
