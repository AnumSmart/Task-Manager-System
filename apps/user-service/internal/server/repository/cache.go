package repository

import (
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
