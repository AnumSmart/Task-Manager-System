package service

import (
	"global_models/global_db"
	"pkg/auth"
	"pkg/outbox"
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
func NewUserService(repo *repository.UserServiceRepository, auth auth.AuthInterface, pool global_db.Pool, outboxRepo outbox.OutboxRepository) *UserService {
	return &UserService{
		User:         NewUserLayer(repo, auth, pool, outboxRepo),
		Organization: NewOrganisationLayer(repo),
		Analytics:    NewAnalyticsLayer(repo),
		Task:         NewTaskLayer(repo),
		Telegram:     NewTelegramLayer(repo, auth, pool, outboxRepo),
		HealthCheck:  NewHealthCheck(repo, auth),
	}
}
