package domain

import (
	"context"
	"errors"
	"time"
)

// UserService - интерфейс сервиса пользователей
// Определен в domain для соблюдения принципа инверсии зависимостей
type UserService interface {
	// Основные CRUD операции
	CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error)
	GetUser(ctx context.Context, req *GetUserRequest) (*User, error)
	UpdateUser(ctx context.Context, req *UpdateUserRequest) error
	DeleteUser(ctx context.Context, req *DeleteUserRequest) error
	ListUsers(ctx context.Context, req *ListUsersRequest) (*ListUsersResponse, error)

	// Операции аутентификации
	Authenticate(ctx context.Context, req *AuthenticateRequest) (*User, error)
	ChangePassword(ctx context.Context, req *ChangePasswordRequest) error

	// Ролевые операции
	TransferOwnership(ctx context.Context, req *TransferOwnershipRequest) error

	// Утилиты
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	InvalidateUserCache(ctx context.Context, userID string) error
}

// ==================== БАЗОВЫЕ ТИПЫ ====================

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

// ==================== ОСНОВНАЯ ДОМЕННАЯ МОДЕЛЬ ====================

// User - чистая доменная модель (не зависит от protobuf)
type User struct {
	ID               string
	Email            string
	TelegramID       *int64
	TelegramUsername *string
	Role             Role
	Status           UserStatus
	FullName         string
	OrganizationID   string // Добавляем связь с организацией
	CreatedAt        time.Time
	UpdatedAt        time.Time
	LastLoginAt      *time.Time
	PasswordHash     string // Только для внутреннего использования, не отправляется в API
}

// ==================== КОНСТРУКТОРЫ ====================

func NewUser(email, fullName string, role Role, organizationID string) *User {
	return &User{
		ID:             GenerateID(),
		Email:          email,
		FullName:       fullName,
		Role:           role,
		Status:         UserStatusActive,
		OrganizationID: organizationID,
		CreatedAt:      Now(),
		UpdatedAt:      Now(),
	}
}

// ==================== БИЗНЕС-МЕТОДЫ ДОМЕННОЙ МОДЕЛИ ====================

// IsActive - активен ли пользователь
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

// IsSuspended - заблокирован ли пользователь
func (u *User) IsSuspended() bool {
	return u.Status == UserStatusSuspended
}

// CanManageUsers - может ли пользователь управлять другими пользователями
func (u *User) CanManageUsers() bool {
	return u.Role == RoleOwner || u.Role == RoleManager
}

// CanDeleteUsers - может ли пользователь удалять других пользователей
func (u *User) CanDeleteUsers() bool {
	return u.Role == RoleOwner
}

// CanViewUser - может ли пользователь видеть другого пользователя
func (u *User) CanViewUser(targetUserID string) bool {
	// EMPLOYEE видит только себя
	if u.Role == RoleEmployee {
		return u.ID == targetUserID
	}
	// MANAGER и OWNER видят всех
	return true
}

// CanUpdateUser - может ли пользователь обновлять данные другого пользователя
func (u *User) CanUpdateUser(targetUserID string, updates map[string]interface{}) bool {
	// OWNER может обновлять любого
	if u.Role == RoleOwner {
		return true
	}

	// EMPLOYEE может обновлять только себя и только определенные поля
	if u.Role == RoleEmployee {
		if u.ID != targetUserID {
			return false
		}
		// Проверяем, что EMPLOYEE не пытается изменить роль или статус
		if _, ok := updates["role"]; ok {
			return false
		}
		if _, ok := updates["status"]; ok {
			return false
		}
		return true
	}

	// MANAGER может обновлять EMPLOYEE, но не OWNER
	if u.Role == RoleManager {
		// TODO: Нужно получить роль целевого пользователя
		// Сейчас упрощенно: MANAGER не может обновлять OWNER
		return true // В реальности нужно проверить роль целевого
	}

	return false
}

// CanDeleteUser - может ли пользователь удалить другого пользователя
func (u *User) CanDeleteUser(targetUserID string, targetRole Role) bool {
	// Только OWNER может удалять пользователей
	if u.Role != RoleOwner {
		return false
	}

	// OWNER не может удалить самого себя
	if u.ID == targetUserID {
		return false
	}

	// OWNER может удалить кого угодно
	return true
}

// PromoteToManager - повысить до MANAGER
func (u *User) PromoteToManager() error {
	if u.Role == RoleOwner {
		return errors.New("cannot promote OWNER to MANAGER")
	}
	u.Role = RoleManager
	u.UpdatedAt = Now()
	return nil
}

// PromoteToOwner - повысить до OWNER
func (u *User) PromoteToOwner() error {
	// Логика повышения до OWNER (обычно через TransferOwnership)
	u.Role = RoleOwner
	u.UpdatedAt = Now()
	return nil
}

// DemoteToEmployee - понизить до EMPLOYEE
func (u *User) DemoteToEmployee() error {
	if u.Role == RoleOwner {
		return errors.New("cannot demote OWNER to EMPLOYEE, transfer ownership first")
	}
	u.Role = RoleEmployee
	u.UpdatedAt = Now()
	return nil
}

// Suspend - заморозить пользователя
func (u *User) Suspend() error {
	if u.Role == RoleOwner {
		return errors.New("cannot suspend OWNER")
	}
	u.Status = UserStatusSuspended
	u.UpdatedAt = Now()
	return nil
}

// Activate - активировать пользователя
func (u *User) Activate() error {
	u.Status = UserStatusActive
	u.UpdatedAt = Now()
	return nil
}

// UpdateProfile - обновление профиля (безопасные поля)
func (u *User) UpdateProfile(fullName string) {
	if fullName != "" {
		u.FullName = fullName
	}
	u.UpdatedAt = Now()
}

// UpdateEmail - обновление email (с подтверждением)
func (u *User) UpdateEmail(email string) error {
	if email == "" {
		return ErrInvalidEmail
	}
	u.Email = email
	u.UpdatedAt = Now()
	return nil
}

// LinkTelegram - привязать Telegram
func (u *User) LinkTelegram(telegramID int64, username string) {
	u.TelegramID = &telegramID
	u.TelegramUsername = &username
	u.UpdatedAt = Now()
}

// UnlinkTelegram - отвязать Telegram
func (u *User) UnlinkTelegram() {
	u.TelegramID = nil
	u.TelegramUsername = nil
	u.UpdatedAt = Now()
}

// IsTelegramLinked - связан ли пользователь с Telegram
func (u *User) IsTelegramLinked() bool {
	return u.TelegramID != nil && *u.TelegramID != 0
}

// GetTelegramID - безопасное получение Telegram ID
func (u *User) GetTelegramID() int64 {
	if u.TelegramID != nil {
		return *u.TelegramID
	}
	return 0
}

// Validate - валидация пользователя
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
	if u.OrganizationID == "" {
		return errors.New("organization id is required")
	}
	return nil
}

// ==================== DTO ДЛЯ ОПЕРАЦИЙ ====================

// CreateUserRequest - DTO для создания пользователя
type CreateUserRequest struct {
	OrganizationID string `json:"organization_id" validate:"required"`
	Email          string `json:"email" validate:"required,email"`
	FullName       string `json:"full_name" validate:"required,min=2,max=100"`
	Role           Role   `json:"role" validate:"required,oneof=OWNER MANAGER EMPLOYEE"`
	CreatedBy      string `json:"created_by" validate:"required"` // ID создающего пользователя
}

// GetUserRequest - DTO для получения пользователя
type GetUserRequest struct {
	UserID      string `json:"user_id" validate:"required"`
	RequesterID string `json:"requester_id" validate:"required"`
}

// UpdateUserRequest - DTO для обновления пользователя
type UpdateUserRequest struct {
	UserID      string                 `json:"user_id" validate:"required"`
	RequesterID string                 `json:"requester_id" validate:"required"`
	Updates     map[string]interface{} `json:"updates"`
}

// DeleteUserRequest - DTO для удаления пользователя
type DeleteUserRequest struct {
	UserID      string `json:"user_id" validate:"required"`
	RequesterID string `json:"requester_id" validate:"required"`
	HardDelete  bool   `json:"hard_delete"`
}

// ListUsersRequest - DTO для получения списка пользователей
type ListUsersRequest struct {
	OrganizationID string            `json:"organization_id" validate:"required"`
	RequesterID    string            `json:"requester_id" validate:"required"`
	Filters        map[string]string `json:"filters"`
	Pagination     Pagination        `json:"pagination"`
}

// Pagination - параметры пагинации
type Pagination struct {
	Offset int `json:"offset" validate:"min=0"`
	Limit  int `json:"limit" validate:"min=1,max=100"`
}

// ListUsersResponse - ответ со списком пользователей
type ListUsersResponse struct {
	Users      []*User `json:"users"`
	TotalCount int     `json:"total_count"`
	HasMore    bool    `json:"has_more"`
}

// AuthenticateRequest - DTO для аутентификации
type AuthenticateRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// ChangePasswordRequest - DTO для смены пароля
type ChangePasswordRequest struct {
	UserID      string `json:"user_id" validate:"required"`
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6"`
}

// TransferOwnershipRequest - DTO для передачи прав OWNER
type TransferOwnershipRequest struct {
	CurrentOwnerID string `json:"current_owner_id" validate:"required"`
	NewOwnerID     string `json:"new_owner_id" validate:"required"`
	OrganizationID string `json:"organization_id" validate:"required"`
}
