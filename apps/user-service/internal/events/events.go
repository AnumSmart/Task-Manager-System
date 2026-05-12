package events

import (
	"encoding/json"
	"pkg/events"
	"time"
	"user-service/internal/domain"

	"github.com/google/uuid"
)

// ============================================================================
// СОБЫТИЕ: USER CREATED
// ============================================================================

// UserCreatedEvent - событие, которое публикуется при создании нового пользователя.
type UserCreatedEvent struct {
	events.BaseEvent                      // Встраиваем базовые поля
	Data             UserCreatedEventData `json:"data"` // Специфичные для события данные
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
		BaseEvent: events.BaseEvent{
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
	events.BaseEvent
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
		BaseEvent: events.BaseEvent{
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
