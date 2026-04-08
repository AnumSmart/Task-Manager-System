package service

import "user-service/internal/server/repository"

// структура части сервисного слоя, которая отвечает за работу телеграмм
type TelegramLayer struct {
	TeleRepo repository.UserDBRepository
}

// констркутор для части сервисного слоя (телеграмм)
// в конструктор передаём составной репозиторий (на будущее)
func NewTelegramLayer(repo *repository.UserServiceRepository) *TelegramLayer {
	return &TelegramLayer{
		TeleRepo: repo.DBRepo,
	}
}
