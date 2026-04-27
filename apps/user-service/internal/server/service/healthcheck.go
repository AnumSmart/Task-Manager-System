package service

import (
	"pkg/auth"
	"user-service/internal/server/repository"
)

// структруа подслоя сервисного слоя, которая отвечает за логику HealthCheck
type HealthCheckLayer struct {
	DB          repository.HealthCheckDBRepo    // использую интерфейс из repo слоя
	Cache       repository.HealthCheckCacheRepo // использую интерфейс из repo слоя
	AuthService auth.AuthInterface              // логика авторизации из пакета pkg/auth
}

// конструктор для HealthCheckLayer
func NewHealthCheck(repo *repository.UserServiceRepository, auth auth.AuthInterface) *HealthCheckLayer {
	return &HealthCheckLayer{
		DB:          repo.DBRepo,
		Cache:       repo.CacheRepo,
		AuthService: auth,
	}
}
