package auth

import (
	"context"
	"fmt"
	"pkg/auth/blacklist"
	"pkg/auth/jwt"
	"pkg/auth/refresh"
	"time"
)

// Нужно проверить, что Auth реализует AuthInterface
var _ AuthInterface = (*Auth)(nil) // Compile-time check

type Auth struct {
	jwt             *jwt.Manager
	blacklist       BlacklistInterface
	refreshStorage  RefreshTokenStorageInterface // ← отдельное хранилище
	enableBlacklist bool
}

// New принимает готовые зависимости, а не конфиг
func New(jwtManager *jwt.Manager, blacklist *blacklist.RedisBlacklist, storage *refresh.RefreshTokenStorage, enableBlacklist bool) *Auth {
	return &Auth{
		jwt:             jwtManager,
		blacklist:       blacklist,
		refreshStorage:  storage,
		enableBlacklist: enableBlacklist,
	}
}

// GenerateTokenPair делегирует jwt менеджеру
func (a *Auth) GenerateTokenPair(ctx context.Context, userID, role, organizationID string) (*jwt.TokenPair, error) {
	// 1. Генерируем токены
	tokens, err := a.jwt.GenerateTokenPair(userID, role, organizationID)
	if err != nil {
		return nil, err
	}

	// 2. Сохраняем refresh token в хранилище (Redis Storage)
	err = a.refreshStorage.Store(ctx, userID, tokens.RefreshToken, 7*24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return tokens, err
}

// ValidateToken проверяет токен (подпись + blacklist)
func (a *Auth) ValidateToken(ctx context.Context, tokenString string) (*jwt.CustomClaims, error) {
	// 1. Проверяем в черном списке
	if a.enableBlacklist && a.blacklist != nil {
		revoked, err := a.blacklist.IsRevoked(ctx, tokenString)
		if err != nil {
			return nil, fmt.Errorf("check blacklist: %w", err)
		}
		if revoked {
			return nil, jwt.ErrTokenRevoked
		}
	}

	// 2. Валидируем токен
	return a.jwt.ValidateToken(tokenString)
}

// RevokeToken добавляет access токен в черный список
func (a *Auth) RevokeToken(ctx context.Context, tokenString string) error {
	if !a.enableBlacklist || a.blacklist == nil {
		return fmt.Errorf("blacklist is not enabled")
	}

	// Получаем оставшееся время жизни
	claims, err := a.jwt.ValidateToken(tokenString)
	if err != nil {
		// Если токен уже невалиден, не добавляем в blacklist
		if err == jwt.ErrExpiredToken {
			return nil
		}
		return fmt.Errorf("validate token before revoke: %w", err)
	}

	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		return nil // Токен уже истек
	}

	return a.blacklist.RevokeToken(ctx, tokenString, ttl)
}

// Logout завершает сессию пользователя
func (a *Auth) Logout(ctx context.Context, userID string, refreshToken string) error {
	if userID == "" || refreshToken == "" {
		return fmt.Errorf("userID and refreshToken cannot be empty")
	}

	// Удаляем refresh token из хранилища
	if err := a.refreshStorage.Revoke(ctx, userID, refreshToken); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	// Access token не трогаем — он истечёт сам
	return nil
}

// RefreshAccessToken обновляет access токен
func (a *Auth) RefreshAccessToken(ctx context.Context, userID string, refreshTokenString string) (*jwt.TokenPair, error) {
	if userID == "" || refreshTokenString == "" {
		return nil, fmt.Errorf("userID and refreshToken cannot be empty")
	}

	// 1. Проверяем существование refresh token в storage
	exists, err := a.refreshStorage.ValidateInStorage(ctx, userID, refreshTokenString)
	if err != nil {
		return nil, fmt.Errorf("validate refresh token: %w", err)
	}
	if !exists {
		return nil, jwt.ErrRefreshTokenNotFound
	}

	// 2. Генерируем новую пару токенов
	tokens, err := a.jwt.RefreshAccessToken(refreshTokenString)
	if err != nil {
		return nil, err
	}

	// 3. Ротация refresh токена (удаляем старый, сохраняем новый)
	if err := a.refreshStorage.Revoke(ctx, userID, refreshTokenString); err != nil {
		return nil, fmt.Errorf("revoke old refresh token: %w", err)
	}
	if err := a.refreshStorage.Store(ctx, userID, tokens.RefreshToken, 7*24*time.Hour); err != nil {
		return nil, fmt.Errorf("store new refresh token: %w", err)
	}

	return tokens, nil
}

// ExtractTokenFromBearer извлекает токен из заголовка
func (a *Auth) ExtractTokenFromBearer(authHeader string) (string, error) {
	return a.jwt.ExtractTokenFromBearer(authHeader)
}

// HealthCheck проверяет здоровье сервиса
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
