package service

import "user-service/internal/server/repository"

// структура части сервисного слоя, которая отвечает за работу с задачами
type TaskLayer struct {
	TaskRepo repository.UserDBRepository
}

// констркутор для части сервисного слоя (задачи)
// в конструктор передаём составной репозиторий (на будущее)
func NewTaskLayer(repo *repository.UserServiceRepository) *TaskLayer {
	return &TaskLayer{
		TaskRepo: repo.DBRepo,
	}
}
