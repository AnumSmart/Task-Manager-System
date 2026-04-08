package service

import "user-service/internal/server/repository"

// структура части сервисного слоя, которая отвечает за работу с пользователями
type UserLayer struct {
	UserRepo repository.UserDBRepository
}

// констркутор для части сервисного слоя (пользователи)
// в конструктор передаём составной репозиторий (на будущее)
func NewUserLayer(repo *repository.UserServiceRepository) *UserLayer {
	return &UserLayer{
		UserRepo: repo.DBRepo,
	}
}
