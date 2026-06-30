package domain

import (
	"errors"
	"strings"
	"time"
)

// BotUser - модель пользователя для телеграм бота
// Использует те же enum'ы, что и в common.v1
type BotUser struct {
	ID               string
	Email            string
	TelegramID       *int64
	TelegramUsername *string
	Role             UserRole
	Status           UserStatus
	FullName         string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	LastLoginAt      *time.Time
}

// UserRole - соответствует common.v1.Role
type UserRole string

const (
	RoleUnspecified UserRole = "ROLE_UNSPECIFIED"
	RoleOwner       UserRole = "ROLE_OWNER"
	RoleManager     UserRole = "ROLE_MANAGER"
	RoleEmployee    UserRole = "ROLE_EMPLOYEE"
)

// UserStatus - соответствует common.v1.UserStatus
type UserStatus string

const (
	UserStatusUnspecified UserStatus = "USER_STATUS_UNSPECIFIED"
	UserStatusActive      UserStatus = "USER_STATUS_ACTIVE"
	UserStatusInactive    UserStatus = "USER_STATUS_INACTIVE"
	UserStatusSuspended   UserStatus = "USER_STATUS_SUSPENDED"
)

// BotOrganization - упрощенная модель организации
type BotOrganization struct {
	ID        string
	Name      string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
	OwnerID   string
}

// JWTInfo - информация о JWT токене
type JWTInfo struct {
	Token        string
	RefreshToken string
	ExpiresIn    int64
	Claims       *JWTClaims
}

// JWTClaims - соответствует common.v1.JWTClaims
type JWTClaims struct {
	UserID         string
	Email          string
	Role           UserRole
	OrganizationID string
	TelegramID     *int64
	TokenID        string
	IssuedAt       int64
	ExpiresAt      int64
}

// ===== Методы с валидацией =====

func (u *BotUser) IsActive() bool {
	return u.Status == UserStatusActive
}

func (u *BotUser) IsTelegramLinked() bool {
	return u.TelegramID != nil
}

func (u *BotUser) CanManageUsers() bool {
	return u.IsActive() &&
		(u.Role == RoleOwner || u.Role == RoleManager)
}

func (u *BotUser) LinkTelegram(telegramID int64, username *string) error {
	if u.IsTelegramLinked() {
		return errors.New("telegram already linked")
	}
	if !u.IsActive() {
		return errors.New("cannot link telegram: user is not active")
	}
	if telegramID <= 0 {
		return errors.New("invalid telegram id")
	}

	u.TelegramID = &telegramID
	u.TelegramUsername = username
	u.UpdatedAt = time.Now()
	return nil
}

func (u *BotUser) UnlinkTelegram() error {
	if !u.IsTelegramLinked() {
		return errors.New("telegram not linked")
	}
	u.TelegramID = nil
	u.TelegramUsername = nil
	u.UpdatedAt = time.Now()
	return nil
}

func (u *BotUser) UpdateProfile(fullName string) error {
	if strings.TrimSpace(fullName) == "" {
		return errors.New("full name cannot be empty")
	}
	if len(fullName) > 100 {
		return errors.New("full name too long (max 100 chars)")
	}
	u.FullName = fullName
	u.UpdatedAt = time.Now()
	return nil
}
