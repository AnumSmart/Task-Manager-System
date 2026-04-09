package domain

import "errors"

// Доменные ошибки для пользователя
var (
	ErrUserNotFound       = errors.New("user not found")         // пользователь не найден
	ErrInvalidEmail       = errors.New("invalid email")          // неверный email у пользоваетля
	ErrInvalidPassword    = errors.New("invalid password")       // неверный пароль
	ErrUserAlreadyExists  = errors.New("user already exists")    // пользователь уже существует
	ErrPermissionDenied   = errors.New("permission denied")      // отказано в разрешении
	ErrInvalidUserID      = errors.New("invalid user id")        // неверный ID пользователя
	ErrInvalidCredentials = errors.New("invalid credentials")    // неверные атрибуты позователя
	ErrUserSuspended      = errors.New("invalid user status")    // неверный статус пользователя
	ErrInvalidInput       = errors.New("not equal organization") // разные организации
)

// Доменные ошибки для организации
var (
	ErrOrganizationNotFound      = errors.New("organization not found")
	ErrOrganizationAlreadyExists = errors.New("organization already exists")
	ErrOrganizationInvalidName   = errors.New("organization name is required")
	ErrOrganizationAlreadyActive = errors.New("organization is already active")
	ErrOrganizationNotActive     = errors.New("organization is not active")
	ErrOnlyOwnerCanModify        = errors.New("only owner can modify organization")
	ErrInvalidOrganizationID     = errors.New("invalid organization id")
)
