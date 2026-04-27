package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"global_models/global_cache"
	"reflect"
	"time"
)

// создаём репозиторий кэша (тут редис) на базе глобального интерфейса

// Реализуем ТОЛЬКО то, что нужно слоя service
type UserServiceCacheRepository struct {
	cache  global_cache.Cache // создаём на базе глобального интерфейса
	prefix string
}

// конструктор для репозитория черного списка (использует интерфейс для глобального кэша)
func NewUserServiceCacheRepo(cache global_cache.Cache, prefix string) (*UserServiceCacheRepository, error) {
	// Проверяем обязательные зависимости
	if cache == nil {
		return nil, fmt.Errorf("cache cannot be nil")
	}

	// Проверяем префикс
	if prefix == "" {
		return nil, fmt.Errorf("prefix cannot be empty")
	}
	return &UserServiceCacheRepository{
		cache:  cache,
		prefix: prefix,
	}, nil
}

func (r *UserServiceCacheRepository) Get(ctx context.Context, key string, dest interface{}) error {
	// Проверяем, что dest - указатель (защита от ошибок)
	if reflect.TypeOf(dest).Kind() != reflect.Ptr {
		return fmt.Errorf("dest must be a pointer, got %T", dest)
	}

	// полный ключ с префиксом
	fullKey := r.prefix + ":" + key

	// Получаем данные из Redis
	data, err := r.cache.GetBytes(ctx, fullKey)
	if err != nil {
		return fmt.Errorf("cache get failed for key %s: %w", fullKey, err)
	}

	// Десериализуем JSON в dest (который должен быть указателем)
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("failed to unmarshal cache value for key %s: %w", fullKey, err)
	}

	return nil
}

// Set - запись значения с автоматической сериализацией
func (r *UserServiceCacheRepository) Set(ctx context.Context, key string, value interface{}, ttl int) error {
	fullKey := r.prefix + ":" + key

	// Сериализуем value в JSON
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value for key %s: %w", fullKey, err)
	}

	// Сохраняем в кэш
	if err := r.cache.Set(ctx, fullKey, data, time.Duration(ttl)*time.Second); err != nil {
		return fmt.Errorf("cache set failed for key %s: %w", fullKey, err)
	}

	return nil
}

// Delete - удаление записи
func (r *UserServiceCacheRepository) Delete(ctx context.Context, key string) error {
	fullKey := r.prefix + ":" + key

	if err := r.cache.Delete(ctx, fullKey); err != nil {
		return fmt.Errorf("cache delete failed for key %s: %w", fullKey, err)
	}

	return nil
}

// PingCache - health check запрос к кэшу
func (r *UserServiceCacheRepository) PingCache(ctx context.Context) error {
	key := "health_check" + r.prefix

	// Проверка отмены контекста
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := r.cache.Set(ctx, key, []byte("ok"), 1); err != nil {
		return fmt.Errorf("redis cache health check failed: %w", err)
	}

	// Проверка отмены контекста перед Delete
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return r.cache.Delete(ctx, key)
}
