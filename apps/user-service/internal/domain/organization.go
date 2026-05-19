package domain

import (
	"errors"
	"fmt"
	"time"
)

// Organization - доменная модель организации
type Organization struct {
	ID        string
	Name      string
	IsActive  bool
	OwnerID   *string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// Конструктор для создания новой организации
func NewOrganization(name, ownerID string) *Organization {
	var ownerIDPtr *string
	if ownerID != "" {
		ownerIDPtr = &ownerID
	}
	return &Organization{
		ID:        GenerateID(), // используйте uuid.New().String()
		Name:      name,
		IsActive:  false, // только что создана, еще не активирована
		OwnerID:   ownerIDPtr,
		CreatedAt: Now(),
		UpdatedAt: Now(),
	}
}

// Бизнес-методы доменной модели

// Activate - активация организации
func (o *Organization) Activate() error {
	if o.IsActive {
		return ErrOrganizationAlreadyActive
	}
	o.IsActive = true
	o.UpdatedAt = time.Now()
	return nil
}

// Deactivate - деактивация организации
func (o *Organization) Deactivate() error {
	if !o.IsActive {
		return ErrOrganizationNotActive
	}
	o.IsActive = false
	o.UpdatedAt = Now()
	return nil
}

// UpdateName - обновление названия организации
func (o *Organization) UpdateName(newName string, requestingUserID string) error {
	// Владельца может не быть (nil), тогда никто не может менять имя?
	// Или только пользователи с ролью OWNER? Смотрите бизнес-логику.
	if o.OwnerID == nil || *o.OwnerID != requestingUserID {
		return ErrOnlyOwnerCanModify
	}

	if newName == "" {
		return ErrOrganizationInvalidName
	}

	o.Name = newName
	o.UpdatedAt = Now()
	return nil
}

// ChangeOwner - смена владельца организации
func (o *Organization) ChangeOwner(newOwnerID string, requestingUserID string) error {
	// Владельца может не быть (nil), тогда никто не может менять имя?
	// Или только пользователи с ролью OWNER? Смотрите бизнес-логику.
	if o.OwnerID == nil || *o.OwnerID != requestingUserID {
		return ErrOnlyOwnerCanModify
	}

	if newOwnerID == "" {
		return errors.New("new owner id is required")
	}

	o.OwnerID = &newOwnerID
	o.UpdatedAt = Now()
	return nil
}

// IsActiveOrganization - проверка активности
func (o *Organization) IsActiveOrganization() bool {
	return o.IsActive
}

// Validate - валидация обязательных полей
func (o *Organization) Validate() error {
	if o.ID == "" {
		return ErrInvalidOrganizationID
	}
	if o.Name == "" {
		return ErrOrganizationInvalidName
	}

	/*
		if o.OwnerID == nil || *o.OwnerID == "" {
			return errors.New("owner id is required")
		}
	*/

	return nil
}

// CanUserManage - может ли пользователь управлять организацией
func (o *Organization) CanUserManage(userID string, userRole Role) bool {
	// Владелец организации или пользователь с ролью OWNER в системе
	return (o.OwnerID != nil && *o.OwnerID == userID) || userRole == RoleOwner
}

// String - для логирования (опционально)
func (o *Organization) String() string {
	return fmt.Sprintf("Organization{ID: %s, Name: %s, IsActive: %t, OwnerID: %v}",
		o.ID, o.Name, o.IsActive, o.OwnerID)
}
