package auth

import (
	"context"
	"pkg/auth/jwt"
	"time"
)

// AuthInterface определяет контракт для сервиса авторизации (User Service)
type AuthInterface interface {
	// Генерация токенов
	GenerateTokenPair(ctx context.Context, userID, role, organizationID string) (*jwt.TokenPair, string, error)

	// Валидация токенов
	ValidateToken(ctx context.Context, tokenString string) (*jwt.CustomClaims, error)

	// Поместиить токен в черный список
	RevokeToken(ctx context.Context, tokenString string) error

	// логаут, удаление refresh токена
	Logout(ctx context.Context, sessionID string) error

	// RefreshAccessToken создает новый access token из refresh token
	RefreshAccessToken(ctx context.Context, userID string, refreshTokenString string) (*jwt.TokenPair, error)

	// Вспомогательные методы
	ExtractTokenFromBearer(authHeader string) (string, error)

	// HeathCheck
	HealthCheck(ctx context.Context) error
}

// TokenValidatorInterface для сервисов которым нужна ТОЛЬКО валидация
// (Task Service, Analytics Service, Notification Service)
type TokenValidatorInterface interface {
	ValidateToken(ctx context.Context, tokenString string) (*jwt.CustomClaims, error)
	ExtractTokenFromBearer(authHeader string) (string, error)
	IsTokenRevoked(ctx context.Context, tokenString string) (bool, error) // ← добавить
}

// TokenGeneratorInterface для сервисов которым нужна генерация (только User Service)
type TokenGeneratorInterface interface {
	GenerateTokenPair(ctx context.Context, userID, role, organizationID string) (*jwt.TokenPair, error)
	RefreshAccessToken(ctx context.Context, userID string, refreshTokenString string) (*jwt.TokenPair, error)
}

// APIKeyValidator для сервис-ту-сервис аутентификации
type APIKeyValidator interface {
	ValidateServiceAPIKey(ctx context.Context, serviceName, apiKey string) bool
}

// BlacklistInterface для управления черным списком
type BlacklistInterface interface {
	RevokeToken(ctx context.Context, token string, ttl time.Duration) error // добавляет токен в черный список
	IsRevoked(ctx context.Context, token string) (bool, error)              // проверяет, что токен в черном списке
	HealthCheck(ctx context.Context) error                                  // health check
}

// RefreshTokenStorageInterface для управления refresh токенами
type RefreshTokenStorageInterface interface {
	Store(ctx context.Context, sessionID string, refreshToken string, ttl time.Duration) error  // сохраняем refresh токен в хранилище
	ValidateInStorage(ctx context.Context, sessionID string, refreshToken string) (bool, error) // проверяем наличие refresh токена в хранилище
	Revoke(ctx context.Context, sessionID string) error                                         // удаляем refresh токен из хранилища
	Exists(ctx context.Context, sessionID string) (bool, error)                                 // Exists проверяет только существование сессии (без проверки токена)
}
