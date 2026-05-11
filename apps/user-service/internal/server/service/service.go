package service

import (
	"pkg/auth"
	"pkg/rabbitmq"
	"user-service/internal/server/repository"
)

// Services - агрегатор всех сервисов (бизнес-логика)
type UserService struct {
	User         *UserServiceLayer
	Organization *OrganizationLayer
	Analytics    *AnalyticsLayer
	Task         *TaskLayer
	Telegram     *TelegramLayer
	HealthCheck  *HealthCheckLayer
}

// конструктор для сервиного слоя (в качестве параметра передаём составной репозиторий)
func NewUserService(repo *repository.UserServiceRepository, auth auth.AuthInterface, broker *rabbitmq.Broker) *UserService {
	return &UserService{
		User:         NewUserLayer(repo, auth, broker),
		Organization: NewOrganisationLayer(repo),
		Analytics:    NewAnalyticsLayer(repo),
		Task:         NewTaskLayer(repo),
		Telegram:     NewTelegramLayer(repo, auth, broker),
		HealthCheck:  NewHealthCheck(repo, auth),
	}
}
