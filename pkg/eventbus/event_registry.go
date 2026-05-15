package eventbus

import (
	"encoding/json"
	"fmt"
	"pkg/events"
	"sync"
)

// EventFactory - фабрика для создания пустого события
type EventFactory func() events.Event

// EventRegistry - реестр для десериализации событий
type EventRegistry struct {
	factories map[string]EventFactory
	mu        sync.RWMutex
}

func NewEventRegistry() *EventRegistry {
	return &EventRegistry{
		factories: make(map[string]EventFactory),
	}
}

func (r *EventRegistry) Register(eventType string, factory EventFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[eventType] = factory
}

func (r *EventRegistry) UnmarshalPayload(eventType string, data []byte) (events.Event, error) {
	r.mu.RLock()
	factory, ok := r.factories[eventType]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}

	event := factory()
	if err := json.Unmarshal(data, event); err != nil {
		return nil, fmt.Errorf("unmarshal event: %w", err)
	}

	return event, nil
}
