package blacklist

import (
	"context"
	"fmt"
	"time"

	"global_models/global_cache"
)

// RedisBlacklist хранит отозванные токены в Redis
type RedisBlacklist struct {
	cache global_cache.Cache
}

// NewRedisBlacklist создает новый экземпляр черного списка
func NewRedisBlacklist(cache global_cache.Cache) *RedisBlacklist {
	return &RedisBlacklist{
		cache: cache,
	}
}

// RevokeToken добавляет токен в черный список
func (b *RedisBlacklist) RevokeToken(ctx context.Context, token string, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	key := b.getKey(token)
	return b.cache.Set(ctx, key, []byte("revoked"), ttl)
}

// IsRevoked проверяет, отозван ли токен
func (b *RedisBlacklist) IsRevoked(ctx context.Context, token string) (bool, error) {
	key := b.getKey(token)
	return b.cache.Exists(ctx, key)
}

// HealthCheck проверяет здоровье Redis
func (b *RedisBlacklist) HealthCheck(ctx context.Context) error {
	testKey := "auth_health_check"
	err := b.cache.Set(ctx, testKey, []byte("ok"), 5*time.Second)
	if err != nil {
		return err
	}
	return b.cache.Delete(ctx, testKey)
}

// getKey формирует ключ для токена в Redis
func (b *RedisBlacklist) getKey(token string) string {
	return fmt.Sprintf("jwt:blacklist:%s", token)
}
