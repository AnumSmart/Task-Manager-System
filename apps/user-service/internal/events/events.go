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

// ============================================================================
// СОБЫТИЕ: USER ROLE CHANGED
// ============================================================================

// UserRoleChangedEvent - событие, которое публикуется при изменении роли пользователя.
type UserRoleChangedEvent struct {
	events.BaseEvent
	Data UserRoleChangedEventData `json:"data"`
}

// UserRoleChangedEventData - данные события user.role_changed.
type UserRoleChangedEventData struct {
	UserID         string    `json:"user_id"`
	OrganizationID string    `json:"organization_id"`
	OldRole        string    `json:"old_role"`         // OWNER/MANAGER/EMPLOYEE
	NewRole        string    `json:"new_role"`         // OWNER/MANAGER/EMPLOYEE
	ChangedBy      string    `json:"changed_by"`       // ID пользователя, который изменил роль
	ChangedByRole  string    `json:"changed_by_role"`  // Роль того, кто изменил
	Reason         string    `json:"reason,omitempty"` // Причина изменения (опционально)
	Timestamp      time.Time `json:"timestamp"`
}

// NewUserRoleChangedEvent - конструктор события user.role_changed.
func NewUserRoleChangedEvent(oldUser, newUser *domain.User, changedBy string, changedByRole domain.Role) *UserRoleChangedEvent {
	return &UserRoleChangedEvent{
		BaseEvent: events.BaseEvent{
			EventID:   uuid.New().String(),
			EventType: "user.role_changed",
			Service:   "user-service",
			Version:   1,
			Timestamp: time.Now().UTC(),
		},
		Data: UserRoleChangedEventData{
			UserID:         newUser.ID,
			OrganizationID: newUser.OrganizationID,
			OldRole:        string(oldUser.Role),
			NewRole:        string(newUser.Role),
			ChangedBy:      changedBy,
			ChangedByRole:  string(changedByRole),
			Timestamp:      time.Now().UTC(),
		},
	}
}

func (e *UserRoleChangedEvent) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// ============================================================================
// СОБЫТИЕ: USER STATUS CHANGED
// ============================================================================

// UserStatusChangedEvent - событие, которое публикуется при изменении статуса пользователя.
type UserStatusChangedEvent struct {
	events.BaseEvent
	Data UserStatusChangedEventData `json:"data"`
}

// UserStatusChangedEventData - данные события user.status_changed.
type UserStatusChangedEventData struct {
	UserID         string    `json:"user_id"`
	OrganizationID string    `json:"organization_id"`
	OldStatus      string    `json:"old_status"`      // ACTIVE/SUSPENDED/INACTIVE
	NewStatus      string    `json:"new_status"`      // ACTIVE/SUSPENDED/INACTIVE
	ChangedBy      string    `json:"changed_by"`      // ID пользователя, который изменил статус
	ChangedByRole  string    `json:"changed_by_role"` // Роль того, кто изменил
	Timestamp      time.Time `json:"timestamp"`
}

// NewUserStatusChangedEvent - конструктор события user.status_changed.
func NewUserStatusChangedEvent(oldUser, newUser *domain.User, changedBy string, changedByRole domain.Role) *UserStatusChangedEvent {
	return &UserStatusChangedEvent{
		BaseEvent: events.BaseEvent{
			EventID:   uuid.New().String(),
			EventType: "user.status_changed",
			Service:   "user-service",
			Version:   1,
			Timestamp: time.Now().UTC(),
		},
		Data: UserStatusChangedEventData{
			UserID:         newUser.ID,
			OrganizationID: newUser.OrganizationID,
			OldStatus:      string(oldUser.Status),
			NewStatus:      string(newUser.Status),
			ChangedBy:      changedBy,
			ChangedByRole:  string(changedByRole),
			Timestamp:      time.Now().UTC(),
		},
	}
}

func (e *UserStatusChangedEvent) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// ============================================================================
// СОБЫТИЕ: USER EMAIL CHANGED
// ============================================================================

// UserEmailChangedEvent - событие, которое публикуется при изменении email пользователя.
type UserEmailChangedEvent struct {
	events.BaseEvent
	Data UserEmailChangedEventData `json:"data"`
}

// UserEmailChangedEventData - данные события user.email_changed.
type UserEmailChangedEventData struct {
	UserID         string    `json:"user_id"`
	OrganizationID string    `json:"organization_id"`
	OldEmail       string    `json:"old_email"`
	NewEmail       string    `json:"new_email"`
	ChangedBy      string    `json:"changed_by"`      // ID пользователя, который изменил email
	ChangedByRole  string    `json:"changed_by_role"` // Роль того, кто изменил
	Timestamp      time.Time `json:"timestamp"`
}

// NewUserEmailChangedEvent - конструктор события user.email_changed.
func NewUserEmailChangedEvent(oldUser, newUser *domain.User, changedBy string, changedByRole domain.Role) *UserEmailChangedEvent {
	return &UserEmailChangedEvent{
		BaseEvent: events.BaseEvent{
			EventID:   uuid.New().String(),
			EventType: "user.email_changed",
			Service:   "user-service",
			Version:   1,
			Timestamp: time.Now().UTC(),
		},
		Data: UserEmailChangedEventData{
			UserID:         newUser.ID,
			OrganizationID: newUser.OrganizationID,
			OldEmail:       oldUser.Email,
			NewEmail:       newUser.Email,
			ChangedBy:      changedBy,
			ChangedByRole:  string(changedByRole),
			Timestamp:      time.Now().UTC(),
		},
	}
}

func (e *UserEmailChangedEvent) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// ============================================================================
// СОБЫТИЕ: USER UPDATED (общее)
// ============================================================================

// UserUpdatedEvent - общее событие, которое публикуется при любом обновлении пользователя.
// Используется как fallback, если не нужно специфичное событие.
type UserUpdatedEvent struct {
	events.BaseEvent
	Data UserUpdatedEventData `json:"data"`
}

// UserUpdatedEventData - данные события user.updated.
type UserUpdatedEventData struct {
	UserID         string                 `json:"user_id"`
	OrganizationID string                 `json:"organization_id"`
	OldData        map[string]interface{} `json:"old_data"`       // Старые значения
	NewData        map[string]interface{} `json:"new_data"`       // Новые значения
	ChangedFields  []string               `json:"changed_fields"` // Какие поля изменились
	ChangedBy      string                 `json:"changed_by"`
	ChangedByRole  string                 `json:"changed_by_role"`
	Timestamp      time.Time              `json:"timestamp"`
}

// NewUserUpdatedEvent - конструктор общего события user.updated.
func NewUserUpdatedEvent(oldUser, newUser *domain.User, updates map[string]interface{}, changedBy string, changedByRole domain.Role) *UserUpdatedEvent {
	changedFields := make([]string, 0, len(updates))
	for field := range updates {
		changedFields = append(changedFields, field)
	}

	// Собираем старые данные только для изменённых полей
	oldData := make(map[string]interface{})
	newData := make(map[string]interface{})

	for _, field := range changedFields {
		switch field {
		case "full_name":
			oldData["full_name"] = oldUser.FullName
			newData["full_name"] = newUser.FullName
		case "email":
			oldData["email"] = oldUser.Email
			newData["email"] = newUser.Email
		case "role":
			oldData["role"] = string(oldUser.Role)
			newData["role"] = string(newUser.Role)
		case "status":
			oldData["status"] = string(oldUser.Status)
			newData["status"] = string(newUser.Status)
		}
	}

	return &UserUpdatedEvent{
		BaseEvent: events.BaseEvent{
			EventID:   uuid.New().String(),
			EventType: "user.updated",
			Service:   "user-service",
			Version:   1,
			Timestamp: time.Now().UTC(),
		},
		Data: UserUpdatedEventData{
			UserID:         newUser.ID,
			OrganizationID: newUser.OrganizationID,
			OldData:        oldData,
			NewData:        newData,
			ChangedFields:  changedFields,
			ChangedBy:      changedBy,
			ChangedByRole:  string(changedByRole),
			Timestamp:      time.Now().UTC(),
		},
	}
}

func (e *UserUpdatedEvent) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
