package domain

import "errors"

// Доменные ошибки для пользователя
var (
	ErrUserNotFound                = errors.New("user not found")                     // пользователь не найден
	ErrInvalidEmail                = errors.New("invalid email")                      // неверный email у пользоваетля
	ErrInvalidPassword             = errors.New("invalid password")                   // неверный пароль
	ErrUserAlreadyExists           = errors.New("user already exists")                // пользователь уже существует
	ErrPermissionDenied            = errors.New("permission denied")                  // отказано в разрешении
	ErrInvalidUserID               = errors.New("invalid user id")                    // неверный ID пользователя
	ErrInvalidCredentials          = errors.New("invalid credentials")                // неверные атрибуты позователя
	ErrUserSuspended               = errors.New("invalid user status")                // неверный статус пользователя
	ErrInvalidInput                = errors.New("not equal organization")             // разные организации
	ErrUserNotBelongToOrganization = errors.New("user do not belong to organization") // пользователь не принадлежит организации
	ErrUserIsNotOwner              = errors.New("user is not OWNER")                  // пользователь не является владельцем организации
)

// Доменные ошибки для организации
var (
	ErrOrganizationNotFound      = errors.New("organization not found")             // организация не найдена
	ErrOrganizationAlreadyExists = errors.New("organization already exists")        // организация уже существует
	ErrOrganizationInvalidName   = errors.New("organization name is required")      // у организации проблемы с именем
	ErrOrganizationAlreadyActive = errors.New("organization is already active")     // организация уже активирована
	ErrOrganizationNotActive     = errors.New("organization is not active")         // организация НЕ активирована
	ErrOnlyOwnerCanModify        = errors.New("only owner can modify organization") // только владелец имеет права на изменения
	ErrInvalidOrganizationID     = errors.New("invalid organization id")            // неверный ID организации
	ErrOrganizationHasUsers      = errors.New("organization still has users")       // в организации ещё есть пользователи
	ErrOrganizationNoOwner       = errors.New("no OWNER in organization")           // для организации не прописан владелец
	ErrNotOrganizationOwner      = errors.New("wrong organization OWNER")           // не тот владелец организации
)

// Доменные ошидки для запросов
var (
	ErrInvalidRequest            = errors.New("request could not be nil")                      // запрос не может быть nil
	ErrInvalidReqEmail           = errors.New("email in request could not be empty")           // в запросе не должен быть пустой Email
	ErrReqFullNameRequired       = errors.New("full name in request could not be empty")       // в запросе не должно быть пустое поле для полного имени
	ErrReqPasswordRequired       = errors.New("password string in request could not be empty") // строка пароля не должна быть пустой
	ErrReqOrganizationIDRequired = errors.New("organization ID could not be empty")            // строка ID организации не должна быть пустой
	ErrReqPasswordTooShort       = errors.New("password string is too short")                  // строка пароля - очень короткая
)
