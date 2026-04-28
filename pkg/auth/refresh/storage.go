package refresh

import (
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

// Store сохраняет refresh токен (хеш) для пользователя
func (s *RefreshTokenStorage) Store(ctx context.Context, userID string, refreshToken string, ttl time.Duration) error {
	// валидация
	if userID == "" {
		return fmt.Errorf("userID cannot be empty")
	}
	if refreshToken == "" {
		return fmt.Errorf("refreshToken cannot be empty")
	}
	if ttl <= 0 {
		return fmt.Errorf("ttl must be positive: %v", ttl)
	}

	tokenHash := sha256.Sum256([]byte(refreshToken))
	key := s.getKey(userID, tokenHash[:])
	return s.cache.Set(ctx, key, []byte(userID), ttl)
}

// Validate проверяет существование refresh токена
func (s *RefreshTokenStorage) ValidateInStorage(ctx context.Context, userID string, refreshToken string) (bool, error) {
	if userID == "" || refreshToken == "" {
		return false, fmt.Errorf("userID and refreshToken cannot be empty")
	}

	tokenHash := sha256.Sum256([]byte(refreshToken))
	key := s.getKey(userID, tokenHash[:])
	return s.cache.Exists(ctx, key)
}

// Revoke удаляет refresh токен при logout
func (s *RefreshTokenStorage) Revoke(ctx context.Context, userID string, refreshToken string) error {
	if userID == "" || refreshToken == "" {
		return fmt.Errorf("userID and refreshToken cannot be empty")
	}

	tokenHash := sha256.Sum256([]byte(refreshToken))
	key := s.getKey(userID, tokenHash[:])
	return s.cache.Delete(ctx, key)
}

// вспомогательная функция для построения правильного ключа
func (s *RefreshTokenStorage) getKey(userID string, tokenHash []byte) string {
	return fmt.Sprintf("refresh:%s:%x", userID, tokenHash)
}
