package domain

import (
	"errors"
	"time"
)

// Role - доменная роль
type Role string

const (
	RoleOwner    Role = "OWNER"
	RoleManager  Role = "MANAGER"
	RoleEmployee Role = "EMPLOYEE"
)

// UserStatus - доменный статус
type UserStatus string

const (
	UserStatusActive    UserStatus = "ACTIVE"
	UserStatusInactive  UserStatus = "INACTIVE"
	UserStatusSuspended UserStatus = "SUSPENDED"
)

// User - чистая доменная модель (не зависит от protobuf)
type User struct {
	ID               string
	Email            string
	TelegramID       *int64
	TelegramUsername *string
	Role             Role
	Status           UserStatus
	FullName         string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	LastLoginAt      *time.Time
	PasswordHash     string // Только для внутреннего использования, не отправляется в API
}

func NewUser(email, fullName string, role Role) *User {
	return &User{
		ID:        GenerateID(),
		Email:     email,
		FullName:  fullName,
		Role:      role,
		Status:    UserStatusActive,
		CreatedAt: Now(),
		UpdatedAt: Now(),
	}
}

// Бизнес-методы доменной модели

// активен ли пользователь
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

// может липользователь делать изменения
func (u *User) CanManageUsers() bool {
	return u.Role == RoleOwner || u.Role == RoleManager
}

// может ли пользователь удалять других пользователей
func (u *User) CanDeleteUsers() bool {
	return u.Role == RoleOwner
}

// связан ли пользователь с телешраммом
func (u *User) IsTelegramLinked() bool {
	return u.TelegramID != nil && *u.TelegramID != 0
}

// валидация пользователя
func (u *User) Validate() error {
	if u.ID == "" {
		return ErrInvalidUserID
	}
	if u.Email == "" {
		return ErrInvalidEmail
	}
	if u.FullName == "" {
		return errors.New("full name is required")
	}
	if u.Role == "" {
		return errors.New("role is required")
	}
	return nil
}

// Метод для безопасного получения Telegram ID
func (u *User) GetTelegramID() int64 {
	if u.TelegramID != nil {
		return *u.TelegramID
	}
	return 0
}
