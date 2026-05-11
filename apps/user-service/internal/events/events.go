package events

import (
	"encoding/json"
	"time"
	"user-service/internal/domain"

	"github.com/google/uuid"
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

// ============================================================================
// СОБЫТИЕ: USER CREATED
// ============================================================================

// UserCreatedEvent - событие, которое публикуется при создании нового пользователя.
type UserCreatedEvent struct {
	BaseEvent                      // Встраиваем базовые поля
	Data      UserCreatedEventData `json:"data"` // Специфичные для события данные
}

// UserCreatedEventData - данные события user.created.
type UserCreatedEventData struct {
	UserID         string `json:"user_id"`         // UUID пользователя
	OrganizationID string `json:"organization_id"` // UUID организации
	Email          string `json:"email"`           // Email пользователя
	FullName       string `json:"full_name"`       // Полное имя
	Role           string `json:"role"`            // OWNER/MANAGER/EMPLOYEE
	Status         string `json:"status"`          // Всегда "ACTIVE" при создании
}

// NewUserCreatedEvent - конструктор события user.created.
// Принимает доменную модель User и возвращает готовое к публикации событие.
func NewUserCreatedEvent(user *domain.User) *UserCreatedEvent {
	return &UserCreatedEvent{
		BaseEvent: BaseEvent{
			EventID:   uuid.New().String(), // Генерируем уникальный ID
			EventType: "user.created",      // Тип события
			Service:   "user-service",      // Наш сервис
			Version:   1,                   // Начинаем с версии 1
			Timestamp: time.Now().UTC(),    // Текущее время в UTC
		},
		Data: UserCreatedEventData{
			UserID:         user.ID,
			OrganizationID: user.OrganizationID,
			Email:          user.Email,
			FullName:       user.FullName,
			Role:           string(user.Role),
			Status:         string(domain.UserStatusActive), // "ACTIVE"
		},
	}
}

// Marshal переопределяем, так как нужно сериализовать вместе с Data
func (e *UserCreatedEvent) Marshal() ([]byte, error) {
	return json.Marshal(e) // сериализует всю структуру с Data
}

// ============================================================================
// СОБЫТИЕ: USER TELEGRAM LINKED
// ============================================================================

// TelegramLinkedEvent - событие, которое публикуется при привязке Telegram.
type TelegramLinkedEvent struct {
	BaseEvent
	Data TelegramLinkedEventData `json:"data"`
}

// TelegramLinkedEventData - данные события user.telegram_linked.
type TelegramLinkedEventData struct {
	UserID         string `json:"user_id"`
	OrganizationID string `json:"organization_id"`
	TelegramID     int64  `json:"telegram_id"`
	Email          string `json:"email"`
}

// NewTelegramLinkedEvent - конструктор события user.telegram_linked.
func NewTelegramLinkedEvent(user *domain.User) *TelegramLinkedEvent {
	telegramID := int64(0)
	if user.TelegramID != nil {
		telegramID = *user.TelegramID
	}

	return &TelegramLinkedEvent{
		BaseEvent: BaseEvent{
			EventID:   uuid.New().String(),
			EventType: "user.telegram_linked",
			Service:   "user-service",
			Version:   1,
			Timestamp: time.Now().UTC(),
		},
		Data: TelegramLinkedEventData{
			UserID:         user.ID,
			OrganizationID: user.OrganizationID,
			TelegramID:     telegramID,
			Email:          user.Email,
		},
	}
}

func (e *TelegramLinkedEvent) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
