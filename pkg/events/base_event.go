package events

import (
	"encoding/json"
	"time"
)

// Event - интерфейс, которому удовлетворяют все события.
// Это позволяет EventPublisher работать с любым событием.
type Event interface {
	// GetEventID - возвращает ID события (для логов и идемпотентности)
	GetEventID() string

	// GetEventType - возвращает тип события (user.created, user.telegram_linked)
	GetEventType() string

	// RoutingKey - возвращает ключ маршрутизации для RabbitMQ
	RoutingKey() string

	// Marshal - сериализует событие в JSON
	Marshal() ([]byte, error)
}

// ============================================================================
// БАЗОВАЯ СТРУКТУРА ДЛЯ ВСЕХ СОБЫТИЙ
// ============================================================================

// BaseEvent - базовая структура, которая встраивается во все события.
// Содержит общие для всех событий поля.
type BaseEvent struct {
	EventID   string    `json:"event_id"`   // Уникальный ID события (UUID)
	EventType string    `json:"event_type"` // Тип события: "user.created", "user.telegram_linked"
	Service   string    `json:"service"`    // Сервис-источник: "user-service"
	Version   int       `json:"version"`    // Версия схемы события (для обратной совместимости)
	Timestamp time.Time `json:"timestamp"`  // Время создания события
}

// RoutingKey возвращает ключ маршрутизации для RabbitMQ.
// Обычно routing_key совпадает с event_type.
// Это позволяет consumer'ам подписываться на "user.*" или конкретные события.
func (e *BaseEvent) RoutingKey() string {
	return e.EventType
}

// Marshal сериализует событие в JSON.
// Ошибка может возникнуть только если структура содержит несериализуемые поля.
// В нашем случае все поля сериализуемы, поэтому ошибка маловероятна.
func (e *BaseEvent) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// GetEventID возвращает уникальный ключ события
func (e *BaseEvent) GetEventID() string { return e.EventID }

// GetEventType возвращает тип события
func (e *BaseEvent) GetEventType() string { return e.EventType }
