package refresh

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"global_models/global_cache"
	"time"
)

// структура хранилища для refresh jwt токенов (хэш токена)
type RefreshTokenStorage struct {
	cache global_cache.Cache
}

// конструктор для хранилища
func NewRefreshTokenStorage(cache global_cache.Cache) *RefreshTokenStorage {
	return &RefreshTokenStorage{cache: cache}
}

// Store сохраняет refresh токен (хеш) для пользователя (формат сохранения {refresh:sessionID}:{tokenHash})
func (s *RefreshTokenStorage) Store(ctx context.Context, sessionID string, refreshToken string, ttl time.Duration) error {
	// валидация
	if sessionID == "" {
		return fmt.Errorf("sessionID cannot be empty")
	}
	if refreshToken == "" {
		return fmt.Errorf("refreshToken cannot be empty")
	}
	if ttl <= 0 {
		return fmt.Errorf("ttl must be positive: %v", ttl)
	}

	tokenHash := sha256.Sum256([]byte(refreshToken))
	key := s.buildKey(sessionID)
	return s.cache.Set(ctx, key, tokenHash[:], ttl)
}

// Validate проверяет, существует ли сессия и соответствует ли refresh токен
func (s *RefreshTokenStorage) ValidateInStorage(ctx context.Context, sessionID string, refreshToken string) (bool, error) {
	if sessionID == "" || refreshToken == "" {
		return false, fmt.Errorf("sessionID and refreshToken cannot be empty")
	}

	key := s.buildKey(sessionID)
	// получаем сохранённый хэш (по построенному ключу)
	data, err := s.cache.Get(ctx, key)
	if err != nil {
		return false, err
	}

	// Проверяем хеш
	tokenHash := sha256.Sum256([]byte(refreshToken))

	return bytes.Equal([]byte(data), tokenHash[:]), nil
}

// Exists проверяет только существование сессии (без проверки токена)
func (s *RefreshTokenStorage) Exists(ctx context.Context, sessionID string) (bool, error) {
	if sessionID == "" {
		return false, fmt.Errorf("sessionID cannot be empty")
	}

	key := s.buildKey(sessionID)
	return s.cache.Exists(ctx, key)
}

// Revoke удаляет refresh токен при logout (по sessionID)
func (s *RefreshTokenStorage) Revoke(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("sessionID cannot be empty")
	}

	key := s.buildKey(sessionID)
	return s.cache.Delete(ctx, key)
}

// вспомогательная функция для построения правильного ключа
func (s *RefreshTokenStorage) buildKey(sessionID string) string {
	return fmt.Sprintf("refresh:%s", sessionID)
}
