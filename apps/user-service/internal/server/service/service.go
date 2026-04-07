package service

import "user-service/internal/server/repository"

// структура сервисного слоя
type UserService struct {
	UserRepo *repository.UserServiceRepository
}

// конструктор для сервиного слоя
func NewUserService(repo *repository.UserServiceRepository) *UserService {
	return &UserService{
		UserRepo: repo,
	}
}
