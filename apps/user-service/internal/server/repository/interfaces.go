package repository

import (
	"context"
	"user-service/internal/domain"
)

// Интерфейс ТОЛЬКО для сервисного слоя (для тестов), для логики пользователей
type UserDBRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id string) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context, offset, limit int) ([]*domain.User, error)
	Count(ctx context.Context) (int, error)
}

// Интерфейс ТОЛЬКО для сервисного слоя (для тестов), для логики кэша
type UserCacheRepository interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl int) error
	Delete(ctx context.Context, key string) error
}

// Интерфейс ТОЛЬКО для сервисного слоя (для тестов), для логики организации
type OrganizationDBRepository interface {
	ExistsAny(ctx context.Context) (bool, error)                                         // ExistsAny - проверяет наличие хотя-бы 1 записи
	CreateOrg(ctx context.Context, org *domain.Organization) error                       // CreateOrg - создаёт запись об организации в БД
	DeleteOrg(ctx context.Context, orgID string) error                                   // DeleteOrganization - удаляет организацию из базы по ID
	GetOrganizationByID(ctx context.Context, orgID string) (*domain.Organization, error) // GetOrganizationByID - получаем организацию из БД
	UpdateOwner(ctx context.Context, orgID, ownerID string) error                        // UpdateOwner - обновляет владельца организации
	Activate(ctx context.Context, orgID string) error                                    // Activate - устанавливает is_active = true
}

// Интерфейс ТОЛЬКО для сервисного слоя (для тестов), для логики HealthCheckDB
type HealthCheckDBRepo interface {
	PingDB(ctx context.Context) error
}

// Интерфейс ТОЛЬКО для сервисного слоя (для тестов), для логики HealthCheckCache
type HealthCheckCacheRepo interface {
	PingCache(ctx context.Context) error
}

// Убеждаемся, что структуры реализуют интерфейсы
var _ UserDBRepository = (*UserServiceDBRepository)(nil)
var _ UserCacheRepository = (*UserServiceCacheRepository)(nil)
