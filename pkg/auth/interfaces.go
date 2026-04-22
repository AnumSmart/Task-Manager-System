package auth

import (
	"context"
	"pkg/auth/jwt"
)

// AuthInterface определяет контракт для сервиса авторизации
type AuthInterface interface {
	// Генерация токенов
	GenerateTokenPair(userID, role, organizationID string) (*jwt.TokenPair, error)

	// Валидация токенов
	ValidateToken(ctx context.Context, tokenString string) (*jwt.CustomClaims, error)

	// Отзыв токенов
	RevokeToken(ctx context.Context, tokenString string) error

	// RefreshAccessToken создает новый access token из refresh token
	RefreshAccessToken(refreshTokenString string) (*jwt.TokenPair, error)

	// API Key валидация
	ValidateServiceAPIKey(serviceName, apiKey string) bool

	// Вспомогательные методы
	ExtractTokenFromBearer(authHeader string) (string, error)

	// HealthCheck
	HealthCheck(ctx context.Context) error
}

// TokenValidatorInterface для сервисов которым нужна только валидация
type TokenValidatorInterface interface {
	ValidateToken(ctx context.Context, tokenString string) (*jwt.CustomClaims, error)
	ExtractTokenFromBearer(authHeader string) (string, error)
}

// TokenGeneratorInterface для сервисов которым нужна генерация
type TokenGeneratorInterface interface {
	GenerateTokenPair(userID, role, organizationID string) (*jwt.TokenPair, error)
	RefreshAccessToken(refreshTokenString string) (*jwt.TokenPair, error)
}
