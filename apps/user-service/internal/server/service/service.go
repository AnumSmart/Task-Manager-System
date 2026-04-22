package service

import (
	"pkg/auth"
	"user-service/internal/server/repository"
)

// Services - агрегатор всех сервисов (бизнес-логика)
type UserService struct {
	User         *UserServiceLayer
	Organization *OrganizationLayer
	Analytics    *AnalyticsLayer
	Task         *TaskLayer
	Telegram     *TelegramLayer
}

// конструктор для сервиного слоя (в качестве параметра передаём составной репозиторий)
func NewUserService(repo *repository.UserServiceRepository, auth auth.AuthInterface) *UserService {
	return &UserService{
		User:         NewUserLayer(repo, auth),
		Organization: NewOrganisationLayer(repo),
		Analytics:    NewAnalyticsLayer(repo),
		Task:         NewTaskLayer(repo),
		Telegram:     NewTelegramLayer(repo),
	}
}
