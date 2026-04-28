package jwt

import "errors"

// JWT-specific errors
var (
	// Базовые ошибки токенов
	ErrInvalidToken     = errors.New("invalid token")
	ErrExpiredToken     = errors.New("token has expired")
	ErrMalformedToken   = errors.New("malformed token")
	ErrInvalidSignature = errors.New("invalid token signature")
	ErrTokenNotValidYet = errors.New("token is not valid yet")
	ErrTokenRevoked     = errors.New("token has been revoked")

	// Ошибки refresh токенов
	ErrNotRefreshToken      = errors.New("token is not a refresh token")
	ErrInvalidRefreshToken  = errors.New("invalid refresh token")
	ErrRefreshTokenExpired  = errors.New("refresh token has expired")
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	ErrRefreshTokenMismatch = errors.New("refresh token does not match user")

	// Ошибки присутствия токена
	ErrMissingToken      = errors.New("missing token")
	ErrEmptyToken        = errors.New("empty token")
	ErrInvalidAuthHeader = errors.New("invalid authorization header format, expected 'Bearer <token>'")

	// Ошибки claims
	ErrMissingUserID         = errors.New("user_id is missing in token claims")
	ErrMissingRole           = errors.New("role is missing in token claims")
	ErrMissingOrganizationID = errors.New("organization_id is missing in token claims")

	// Ошибки конфигурации
	ErrInvalidConfig     = errors.New("invalid jwt config")
	ErrEmptySecretKey    = errors.New("jwt secret key is empty")
	ErrWeakSecretKey     = errors.New("jwt secret key must be at least 32 characters")
	ErrInvalidAccessTTL  = errors.New("access token TTL must be positive")
	ErrInvalidRefreshTTL = errors.New("refresh token TTL must be positive")

	// Ошибки парсинга
	ErrUnexpectedSigningMethod = errors.New("unexpected signing method")
	ErrParseToken              = errors.New("failed to parse token")
)
