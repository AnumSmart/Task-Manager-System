package auth

import (
	"context"
	"fmt"
	"pkg/auth/blacklist"
	"pkg/auth/jwt"
	"time"
)

type Auth struct {
	jwt             *jwt.Manager
	blacklist       *blacklist.RedisBlacklist
	enableBlacklist bool
}

// New принимает готовые зависимости, а не конфиг
func New(jwtManager *jwt.Manager, blacklist *blacklist.RedisBlacklist, enableBlacklist bool) *Auth {
	return &Auth{
		jwt:             jwtManager,
		blacklist:       blacklist,
		enableBlacklist: enableBlacklist,
	}
}

// GenerateTokenPair делегирует jwt менеджеру
func (a *Auth) GenerateTokenPair(userID, role, organizationID string) (*jwt.TokenPair, error) {
	return a.jwt.GenerateTokenPair(userID, role, organizationID)
}

// ValidateToken - добавляем метод с context
func (a *Auth) ValidateToken(ctx context.Context, tokenString string) (*jwt.CustomClaims, error) {
	// Сначала проверяем в черном списке
	if a.enableBlacklist && a.blacklist != nil {
		revoked, err := a.blacklist.IsRevoked(ctx, tokenString)
		if err != nil {
			return nil, fmt.Errorf("check blacklist: %w", err)
		}
		if revoked {
			return nil, fmt.Errorf("token is revoked")
		}
	}

	// Валидируем токен
	return a.jwt.ValidateToken(tokenString)
}

// RevokeToken - добавляем метод с context
func (a *Auth) RevokeToken(ctx context.Context, tokenString string) error {
	if !a.enableBlacklist || a.blacklist == nil {
		return fmt.Errorf("blacklist is not enabled")
	}

	// Получаем TTL из токена
	claims, err := a.jwt.ValidateToken(tokenString)
	if err != nil {
		return fmt.Errorf("validate token before revoke: %w", err)
	}

	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	return a.blacklist.RevokeToken(ctx, tokenString, ttl)
}

// RefreshAccessToken - делегируем jwt менеджеру
func (a *Auth) RefreshAccessToken(refreshTokenString string) (*jwt.TokenPair, error) {
	return a.jwt.RefreshAccessToken(refreshTokenString)
}

// ValidateServiceAPIKey - нужно реализовать
func (a *Auth) ValidateServiceAPIKey(serviceName, apiKey string) bool {
	// Реализуйте логику проверки API ключей сервисов
	// Например, проверка по конфигу или базе данных
	return false // временно
}

// ExtractTokenFromBearer - делегируем jwt менеджеру
func (a *Auth) ExtractTokenFromBearer(authHeader string) (string, error) {
	return a.jwt.ExtractTokenFromBearer(authHeader)
}

// HealthCheck - уже есть
func (a *Auth) HealthCheck(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("context canceled")
	default:
	}

	if a.blacklist == nil {
		return fmt.Errorf("blacklist is not enabled or not initialized")
	}

	return a.blacklist.HealthCheck(ctx)
}
