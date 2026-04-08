package repository

import (
	"context"
	"fmt"
	"global_models/global_cache"
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

// метод для получения значения из кэша по ключу
func (r *UserServiceCacheRepository) Get(ctx context.Context, key string, dest interface{}) error {
	// полный ключ с префиксом
	//fullKey := r.prefix + ":" + key
	// -------------------------- в разработке --------------------------

	return nil
}

// метод для записи пары ключ-значения в кэш с TTL
func (r *UserServiceCacheRepository) Set(ctx context.Context, key string, value interface{}, ttl int) error {
	// полный ключ с префиксом
	//fullKey := r.prefix + ":" + key
	// -------------------------- в разработке --------------------------

	return nil
}

// метод для удаления записи из кэша по ключу
func (r *UserServiceCacheRepository) Delete(ctx context.Context, key string) error {
	// полный ключ с префиксом
	//fullKey := r.prefix + ":" + key
	// -------------------------- в разработке --------------------------

	return nil
}
