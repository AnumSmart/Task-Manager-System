package service

import (
	"context"
	"pkg/auth"
	"user-service/internal/server/repository"
)

// структруа подслоя сервисного слоя, которая отвечает за логику HealthCheck
type HealthCheckLayer struct {
	db          repository.HealthCheckDBRepo    // использую интерфейс из repo слоя
	cache       repository.HealthCheckCacheRepo // использую интерфейс из repo слоя
	authService auth.AuthInterface              // логика авторизации из пакета pkg/auth
}

// конструктор для HealthCheckLayer
func NewHealthCheck(repo *repository.UserServiceRepository, auth auth.AuthInterface) *HealthCheckLayer {
	return &HealthCheckLayer{
		db:          repo.DBRepo,
		cache:       repo.CacheRepo,
		authService: auth,
	}
}

// метод проверки Health Check БД
func (h *HealthCheckLayer) HealthCheckDB(ctx context.Context) error {
	err := h.db.PingDB(ctx)
	if err != nil {
		return err
	}
	return nil
}

// метод проверки Health Check кэша (Redis)
func (h *HealthCheckLayer) HealthCheckCache(ctx context.Context) error {
	err := h.cache.PingCache(ctx)
	if err != nil {
		return err
	}
	return nil
}

// метод проверки Health Check blacklist (Redis)
func (h *HealthCheckLayer) HealthCheckBlackList(ctx context.Context) error {
	err := h.authService.HealthCheck(ctx)
	if err != nil {
		return err
	}
	return nil
}
