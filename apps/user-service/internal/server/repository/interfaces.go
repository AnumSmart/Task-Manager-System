package repository

import (
	"context"
	"user-service/internal/domain"
)

// Интерфейсы ТОЛЬКО для сервисного слоя (для тестов)
type UserDBRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id string) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context, offset, limit int) ([]*domain.User, error)
}

type UserCacheRepository interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl int) error
	Delete(ctx context.Context, key string) error
}

// Убеждаемся, что структуры реализуют интерфейсы
var _ UserDBRepository = (*UserServiceDBRepository)(nil)
var _ UserCacheRepository = (*UserServiceCacheRepository)(nil)
