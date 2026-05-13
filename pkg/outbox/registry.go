package outbox

import (
	"encoding/json"
	"fmt"
	"pkg/events"
)

// EventFactory - фабрика для создания пустого события по типу
type EventFactory func() events.Event

// EventRegistry - реестр типов событий
type EventRegistry struct {
	factories map[string]EventFactory
}

// NewEventRegistry создаёт новый реестр
func NewEventRegistry() *EventRegistry {
	return &EventRegistry{
		factories: make(map[string]EventFactory), // инициализируем мапу
	}
}

// Register регистрирует фабрику для события с указанным типом
func (r *EventRegistry) Register(eventType string, factory EventFactory) {
	r.factories[eventType] = factory
}

// UnmarshalPayload десериализует JSON в конкретное событие
func (r *EventRegistry) UnmarshalPayload(eventType string, data []byte) (events.Event, error) {
	factory, ok := r.factories[eventType]
	if !ok {
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}

	//вызываем фабрику, которая создаёт событие нужного типа и в неё помещаем данный после анмаршалинга
	event := factory()
	if err := json.Unmarshal(data, event); err != nil {
		return nil, fmt.Errorf("unmarshal event: %w", err)
	}

	return event, nil
}
